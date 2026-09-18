package climetrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
)

// MuseMeter reports Muse Code from the session.jsonl logs under
// ~/.local/share/muse/sessions, the same store clisession.MuseSource reads
// through the `muse export` projector. The dashboard does not shell out per
// session — it reads the log's own event envelopes directly, which carry
// the same kinds the exporter projects: user turns arrive as
// runtime.user_intent.accepted, assistant text as run events of kind
// assistant_message_committed, tool traffic as the commit batches, the
// model as the metadata record and model_completed.
//
// What the log never writes is the honest boundary of this meter: no token
// counts, no cost, no per-turn error state. Muse's "usage" samples are
// process resources (rss_self_bytes), not model tokens, and reading them
// as spend would be exactly the fabrication ADR-0097 refuses.
//
// Timestamps are epoch micros on every record (recorded_at), the same
// clock the session index keeps in created_at_us.
type MuseMeter struct{}

func (MuseMeter) CLI() string   { return "muse" }
func (MuseMeter) Label() string { return "Muse Code" }

// Fingerprint covers the session index and the log tree: the index moves on
// every bookkeeping write, the logs on every turn, and either one must
// invalidate the window.
func (MuseMeter) Fingerprint() string {
	root := clisession.MuseSessionsRoot()
	if root == "" {
		return ""
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return "0:0:0|0:0:0"
	}
	fp := dbFingerprint(clisession.MuseDBPath())
	var n, size, newest int64
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "session.jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		n++
		size += info.Size()
		if m := info.ModTime().UnixNano(); m > newest {
			newest = m
		}
		return nil
	})
	return fp + "|" + itoa64(n) + ":" + itoa64(size) + ":" + itoa64(newest)
}

func (m MuseMeter) Meter(req Request) (Window, error) {
	root := clisession.MuseSessionsRoot()
	if root == "" {
		return absentWindow(m, req), nil
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return absentWindow(m, req), nil
	}
	acc := newGuestAcc(req, m.CLI())
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "session.jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		// A log untouched since before the widened window cannot hold an
		// in-window turn: cheap to check, expensive to parse.
		if !req.PriorFrom.IsZero() && info.ModTime().Before(req.PriorFrom) {
			return nil
		}
		mtime := info.ModTime()
		replay(cachedParse(p, func(path string) *parsed { return museParse(path, mtime) }), acc, req)
		return nil
	})
	return Window{
		CLI:      m.CLI(),
		Stats:    acc.result(),
		Coverage: museCoverage(m, req.BillingFor(m.CLI()), acc),
	}, nil
}

// museCan is what Muse Code is capable of recording: prompts, turns, tools
// and the model — and nothing priced, tokenised or timed.
var museCan = map[Signal]bool{
	SigModel: true, SigMessages: true, SigTurns: true, SigTools: true,
}

func museCoverage(m MuseMeter, b Billing, acc *guestAcc) CoverageRow {
	return CoverageRow{
		CLI: m.CLI(), Label: m.Label(), Billing: b,
		Signals: acc.evidence(museCan, nil),
		Note:    "Its logs carry no token counts, cost or durations — its \"usage\" samples are process resources, not model tokens. It records no edit counts or quota windows.",
	}
}

// museEnvelope is one record of the log, top-level or unwrapped from a
// retained frame's children.
type museEnvelope struct {
	RecordedAt  int64           `json:"recorded_at"`
	PayloadType string          `json:"payload_type"`
	Payload     json.RawMessage `json:"payload"`
}

// museParse reads one session.jsonl into a window-independent parse. Text
// is never kept: counts, names and identities travel, message bodies stay
// in the file.
func museParse(path string, mtime time.Time) *parsed {
	f, err := os.Open(path)
	if err != nil {
		return &parsed{}
	}
	defer f.Close()

	out := &parsed{key: museKey(path), byPresence: true}
	cwd, model := "", ""
	// Tool calls arrive as their own events after the assistant message
	// that made them, so they attach to the most recent turn — the codex
	// adapter's pending list would charge them to the *next* turn here.
	attach := func(tools []string) {
		if len(tools) == 0 {
			return
		}
		for i := len(out.ents) - 1; i >= 0; i-- {
			if out.ents[i].role == "assistant" {
				out.ents[i].tools = append(out.ents[i].tools, tools...)
				return
			}
		}
	}
	feed := func(env museEnvelope) {
		at := mtime
		if env.RecordedAt > 0 {
			at = time.UnixMicro(env.RecordedAt)
		}
		switch env.PayloadType {
		case "runtime.user_intent.accepted":
			// materialized intents are the same turns filed twice;
			// accepted is the one count.
			out.ents = append(out.ents, guestEntry{at: at, key: out.key, cwd: cwd, role: "user"})
			out.units++ // presence-weighted: muse writes no tokens to prorate by
		case "runtime.session":
			var p struct {
				Kind   string `json:"kind"`
				Record *struct {
					Workspace string `json:"workspace_root"`
					ModelID   string `json:"model_id"`
				} `json:"record"`
				Event *struct {
					Kind  string          `json:"kind"`
					Text  string          `json:"text"`
					Model json.RawMessage `json:"model"`
					Calls []struct {
						Name string `json:"name"`
					} `json:"tool_calls"`
				} `json:"event"`
			}
			if json.Unmarshal(env.Payload, &p) != nil {
				return
			}
			switch p.Kind {
			case "metadata":
				if p.Record != nil {
					if cwd == "" {
						cwd = p.Record.Workspace
					}
					if model == "" {
						model = p.Record.ModelID
					}
				}
			case "run":
				if p.Event == nil {
					return
				}
				switch p.Event.Kind {
				case "assistant_message_committed":
					if strings.TrimSpace(p.Event.Text) == "" {
						return
					}
					out.ents = append(out.ents, guestEntry{
						at: at, key: out.key, cwd: cwd,
						role: "assistant", model: model, prov: "meta",
					})
					out.units++
				case "assistant_tool_calls_committed":
					var tools []string
					for _, c := range p.Event.Calls {
						if c.Name != "" {
							tools = append(tools, c.Name)
						}
					}
					attach(tools)
				case "model_completed":
					if model == "" {
						var mm string
						if json.Unmarshal(p.Event.Model, &mm) == nil && mm != "" {
							model = mm
						}
					}
				}
			}
		}
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var r struct {
			museEnvelope
			Children []struct {
				RecordJSON string `json:"record_json"`
			} `json:"children"`
		}
		if json.Unmarshal(line, &r) != nil {
			continue
		}
		if r.PayloadType != "" {
			feed(r.museEnvelope)
		}
		for _, ch := range r.Children {
			var inner museEnvelope
			if json.Unmarshal([]byte(ch.RecordJSON), &inner) != nil {
				continue
			}
			feed(inner)
		}
	}
	// The workspace only becomes known at the metadata record; entries
	// built before it borrow the latest truth.
	for i := range out.ents {
		if out.ents[i].cwd == "" {
			out.ents[i].cwd = cwd
		}
		if out.ents[i].model == "" {
			out.ents[i].model = model
		}
	}
	return out
}

// museKey is the session identity every entry in one log carries: the log's
// own directory name, which is the session uuid the index keys on.
func museKey(path string) string {
	return filepath.Base(filepath.Dir(path))
}
