package climetrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
)

// GrokMeter reports the Grok CLI from its own session directories. Four
// files carry four different answers and the meter keeps them apart:
//
//   - prompt_history.jsonl, one per cwd folder: a line per prompt, and the
//     only file that says how much someone *asked*. It records no tokens.
//   - summary.json, one per session: cwd, model and message count. Every
//     session in this machine's store has one (231 of 231, 2026-09-11).
//   - events.jsonl, one per session: the turn/tool timeline — turn_started,
//     loop_started (one per model call), first_token, tool_started /
//     tool_completed(duration_ms), turn_ended(outcome). 188 of 231 are
//     non-empty and 176 hold completed turns.
//   - usage.json, one per session: per-turn input/output/cache/reasoning
//     tokens, costUsdTicks (10^10 ticks per USD, per Grok's own user guide)
//     and the model behind the turn. Grok only began writing it in 1.0.x —
//     3 of 231 sessions here — which is why tokens and cost report as
//     *partial*, with both counts, never as a floor dressed up as a total.
//
// An earlier version of this meter read prompt_history.jsonl alone and told
// the dashboard Grok "records prompt history only". That was true of the
// install it was measured against and false of the one PiCode launched
// since; a surface answering "—" beside a CLI that wrote the number is the
// invisibility ADR-0097 exists to end.
type GrokMeter struct{}

func (GrokMeter) CLI() string   { return "grok" }
func (GrokMeter) Label() string { return "Grok" }

// grokCounted names every file whose change moves this meter's answer. The
// fingerprint and the parse cache both key on all of them: a turn appends
// to events.jsonl while it runs and writes usage.json when it ends, so a
// detector watching one would serve the other's stale half.
var grokCounted = map[string]bool{
	"prompt_history.jsonl": true,
	"summary.json":         true,
	"events.jsonl":         true,
	"usage.json":           true,
}

func (GrokMeter) Fingerprint() string {
	root := clisession.GrokSessionsRoot()
	if root == "" {
		return ""
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return "0:0:0"
	}
	var n, size, newest int64
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !grokCounted[d.Name()] {
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
	return itoa64(n) + ":" + itoa64(size) + ":" + itoa64(newest)
}

func (m GrokMeter) Meter(req Request) (Window, error) {
	root := clisession.GrokSessionsRoot()
	if root == "" {
		return absentWindow(m, req), nil
	}
	if _, err := os.Stat(root); err != nil {
		return absentWindow(m, req), nil
	}
	acc := newCliAcc(req, m.CLI())
	seen := map[string]bool{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == "prompt_history.jsonl" {
			replay(cachedParse(p, grokPromptParse), acc, req)
			return nil
		}
		if !grokSessionFiles[d.Name()] {
			return nil
		}
		// One parse per session directory, whichever of its files the walk
		// reached first.
		dir := filepath.Dir(p)
		if seen[dir] {
			return nil
		}
		seen[dir] = true
		key := statKey(
			filepath.Join(dir, "summary.json"),
			filepath.Join(dir, "events.jsonl"),
			filepath.Join(dir, "usage.json"),
		)
		replay(cachedParseKeyed(key, func() *parsed { return grokSessionParse(dir) }), acc, req)
		return nil
	})
	w := Window{
		CLI:      m.CLI(),
		Stats:    acc.result(),
		Coverage: grokCoverage(m, req.BillingFor(m.CLI()), acc),
	}
	if acc.timing != (Timing{}) {
		t := acc.timing
		w.Timing = &t
	}
	return w, nil
}

// grokSessionFiles are the files that describe a session directory. Any of
// them triggers the directory's single parse, so a session that has events
// but no summary (a live one) is still read once.
var grokSessionFiles = map[string]bool{"summary.json": true, "events.jsonl": true, "usage.json": true}

// grokCan is what the Grok CLI is capable of recording. It counts no lines
// changed, so impact stays out; whether it *did* record the rest in a window
// is the accumulator's evidence, not this table's claim.
var grokCan = map[Signal]bool{
	SigCost: true, SigTokens: true, SigModel: true, SigMessages: true,
	SigTurns: true, SigTools: true, SigErrors: true, SigTiming: true,
}

func grokCoverage(m GrokMeter, b Billing, acc *cliAcc) CoverageRow {
	sig := acc.evidence(grokCan, map[Signal]bool{SigTiming: acc.timing != (Timing{})})
	note := "Turns, tool calls and durations come from each session's events.jsonl; the model comes from summary.json; tokens and cost live only in usage.json, which Grok began writing in 1.0.x."
	turns, tok, cost := acc.seen[SigTurns], acc.seen[SigTokens], acc.seen[SigCost]
	switch {
	case turns == 0:
		// Nothing in this window; there is nothing to qualify.
	case tok == 0 && cost == 0:
		sig[SigTokens], sig[SigCost] = StateNotReported, StateNotReported
		note += " No turn here has a usage record, so its tokens and cost are unmeasured rather than zero."
	case tok < turns || cost < turns:
		sig[SigTokens], sig[SigCost] = StatePartial, StatePartial
		note += " " + itoa(tok) + " of " + itoa(turns) + " turns carry tokens and " +
			itoa(cost) + " carry a cost; the missing ones are not free."
	}
	return CoverageRow{CLI: m.CLI(), Label: m.Label(), Billing: b, Signals: sig, Note: note}
}

// grokPromptParse reads one folder's prompt history. The prompt text itself
// is never touched: only its timestamp and session id.
func grokPromptParse(path string) *parsed {
	out := &parsed{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	cwd := grokCwd(path)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 32*1024), 4*1024*1024)
	for sc.Scan() {
		var l struct {
			Timestamp string `json:"timestamp"`
			SessionID string `json:"session_id"`
		}
		if json.Unmarshal(sc.Bytes(), &l) != nil {
			continue
		}
		at, err := time.Parse(time.RFC3339, l.Timestamp)
		if err != nil {
			continue
		}
		key := l.SessionID
		if key == "" {
			key = path
		}
		out.ents = append(out.ents, cliEntry{at: at, key: key, cwd: cwd, role: "user"})
	}
	return out
}

// grokSessionParse reads one session directory in a single pass: summary.json
// for identity and model, events.jsonl for the timeline, usage.json for the
// numbers. It emits one entry per turn and never one per file — the two
// files describe the same turn from two angles, and counting it twice would
// inflate every messages and turns figure on the dashboard.
//
// Entries weigh one each (byPresence) rather than their token count: the
// timeline is what every session has, and prorating durations by tokens
// would drop them for the sessions that have no usage record yet.
func grokSessionParse(dir string) *parsed {
	out := &parsed{key: filepath.Base(dir), byPresence: true}
	cwd := grokDirCwd(dir)
	title, model := "", ""
	if sum := grokSummary(filepath.Join(dir, "summary.json")); sum != nil {
		if sum.ID != "" {
			out.key = sum.ID
		}
		if sum.Cwd != "" {
			cwd = sum.Cwd
		}
		title, model = sum.Title, sum.Model
	}

	for _, t := range grokMergeTurns(grokEventTurns(dir), grokUsageTurns(dir), model) {
		out.timing.APIMs += t.apiMs
		out.timing.ToolMs += t.toolMs
		out.timing.SessionMs += t.sessionMs
		if t.at.IsZero() {
			continue
		}
		out.units++
		out.ents = append(out.ents, cliEntry{
			at: t.at, key: out.key, cwd: cwd, name: title,
			role: "assistant", model: t.model, prov: grokProvider,
			cost: t.cost, toks: t.toks,
			tools: t.tools, results: t.results, errs: t.errs, abort: t.cancel,
		})
	}
	return out
}

// grokProvider is the vendor PiCode files these models under. Grok records
// no provider column of its own; its models are xAI's.
const grokProvider = "xai"

// grokCostTicksPerUSD is Grok's own unit for costUsdTicks — 10^10 ticks per
// dollar, from its user guide ("The grok usage Subcommand"). A conversion
// from the CLI's own arithmetic, never a price list this repo maintains.
const grokCostTicksPerUSD = 1e10

// grokSessionInfo is the part of summary.json this meter reads. The file
// also carries the title Grok shows in its own session list.
type grokSessionInfo struct {
	ID    string
	Cwd   string
	Model string
	Title string
}

func grokSummary(path string) *grokSessionInfo {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Info struct {
			ID  string `json:"id"`
			Cwd string `json:"cwd"`
		} `json:"info"`
		Summary string `json:"session_summary"`
		Model   string `json:"current_model_id"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	return &grokSessionInfo{ID: doc.Info.ID, Cwd: doc.Info.Cwd, Model: doc.Model, Title: doc.Summary}
}

// grokTurn is one turn as the two files describe it: the timeline from
// events.jsonl, the numbers from usage.json when it exists.
type grokTurn struct {
	at        time.Time // when the turn counts, for whichever window it falls in
	start     time.Time
	end       time.Time
	apiMs     int64
	toolMs    int64
	sessionMs int64
	cancel    bool
	hasTokens bool
	toks      session.TokenTotals
	cost      float64
	model     string
	tools     []string
	results   int
	errs      int
}

// grokEventTurns folds events.jsonl into one timeline per turn.
//
// A turn ends at turn_ended; one that never wrote it (a killed process) is
// closed at the last event it did write, so its tool calls and its time
// still count. A model window runs from loop_started to the tool that
// follows — or to the end of the turn — which is Grok's own shape: one
// loop per model call, several per turn.
func grokEventTurns(dir string) []grokTurn {
	f, err := os.Open(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []grokTurn
	var cur *grokTurn
	var loopStart time.Time
	var last time.Time
	ensure := func(at time.Time) *grokTurn {
		if cur == nil {
			out = append(out, grokTurn{start: at})
			cur = &out[len(out)-1]
		}
		return cur
	}
	// close settles the open turn at at: the model window still running ends
	// there, and the turn's own window is measured from its start.
	closeTurn := func(at time.Time) {
		if cur == nil {
			return
		}
		if !loopStart.IsZero() {
			cur.apiMs += at.Sub(loopStart).Milliseconds()
			loopStart = time.Time{}
		}
		if cur.at.IsZero() {
			cur.at, cur.end = at, at
		}
		if !cur.start.IsZero() {
			cur.sessionMs = at.Sub(cur.start).Milliseconds()
		}
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 32*1024), 4*1024*1024)
	for sc.Scan() {
		var ev struct {
			TS       string  `json:"ts"`
			Type     string  `json:"type"`
			Outcome  string  `json:"outcome"`
			Tool     string  `json:"tool_name"`
			Duration float64 `json:"duration_ms"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, ev.TS)
		if err != nil {
			continue
		}
		last = at
		switch ev.Type {
		case "turn_started":
			closeTurn(at)
			out = append(out, grokTurn{start: at})
			cur = &out[len(out)-1]
			loopStart = time.Time{}
		case "loop_started":
			ensure(at)
			// A model call starts here. One that never closed (a cancelled
			// stream) is closed at this boundary rather than dropped.
			if !loopStart.IsZero() {
				cur.apiMs += at.Sub(loopStart).Milliseconds()
			}
			loopStart = at
		case "tool_started":
			ensure(at)
			if ev.Tool != "" {
				cur.tools = append(cur.tools, ev.Tool)
			}
			if !loopStart.IsZero() {
				cur.apiMs += at.Sub(loopStart).Milliseconds()
				loopStart = time.Time{}
			}
		case "tool_completed":
			ensure(at)
			cur.results++
			if ev.Outcome != "" && ev.Outcome != "success" {
				cur.errs++
			}
			cur.toolMs += int64(ev.Duration)
		case "turn_ended":
			ensure(at)
			cur.cancel = ev.Outcome == "cancelled"
			closeTurn(at)
			cur = nil
		}
	}
	closeTurn(last)
	return out
}

// grokUsageTurns reads usage.json's per-turn numbers. Session totals are
// deliberately not read: they include history a resume or fork inherited,
// so summing them across a session chain would count the same tokens twice,
// while the turns are the disjoint unit a window can be cut from.
func grokUsageTurns(dir string) []grokTurn {
	raw, err := os.ReadFile(filepath.Join(dir, "usage.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Session struct {
			PrimaryModelID string `json:"primaryModelId"`
		} `json:"session"`
		Turns []struct {
			EndedAt             string `json:"endedAt"`
			InputTokens         int64  `json:"inputTokens"`
			OutputTokens        int64  `json:"outputTokens"`
			CachedReadTokens    int64  `json:"cachedReadTokens"`
			CacheCreationTokens int64  `json:"cacheCreationTokens"`
			ReasoningTokens     int64  `json:"reasoningTokens"`
			ModelCalls          int    `json:"modelCalls"`
			CostUsdTicks        int64  `json:"costUsdTicks"`
			PrimaryModelID      string `json:"primaryModelId"`
		} `json:"turns"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	out := make([]grokTurn, 0, len(doc.Turns))
	for _, t := range doc.Turns {
		if t.ModelCalls == 0 && t.InputTokens == 0 && t.OutputTokens == 0 {
			continue // a turn that called no model says nothing
		}
		end, _ := time.Parse(time.RFC3339Nano, t.EndedAt)
		model := t.PrimaryModelID
		if model == "" {
			model = doc.Session.PrimaryModelID
		}
		out = append(out, grokTurn{
			at: end, end: end, model: model, hasTokens: true,
			cost: float64(t.CostUsdTicks) / grokCostTicksPerUSD,
			toks: session.TokenTotals{
				Input:      t.InputTokens,
				Output:     t.OutputTokens,
				CacheRead:  t.CachedReadTokens,
				CacheWrite: t.CacheCreationTokens,
				Reasoning:  t.ReasoningTokens,
			},
		})
	}
	return out
}

// grokTurnSlack is how far a usage turn's own endedAt may sit from the
// turn_ended that closes the same turn. Grok numbers event turns from zero
// and usage turns from one, so the numbers cannot be paired — the
// timestamps can, and on a real store they agree to the millisecond.
const grokTurnSlack = 2 * time.Second

// grokMergeTurns joins the two timelines. The event timeline is the spine,
// since nearly every session has one; a usage turn is folded onto the event
// turn that ended at the same instant, and kept on its own when there is no
// such turn.
func grokMergeTurns(events, usage []grokTurn, model string) []grokTurn {
	out := append([]grokTurn(nil), events...)
	for _, u := range usage {
		match := -1
		best := grokTurnSlack
		for i := range out {
			if out[i].hasTokens || out[i].end.IsZero() || u.end.IsZero() {
				continue
			}
			d := out[i].end.Sub(u.end)
			if d < 0 {
				d = -d
			}
			if d <= best {
				best, match = d, i
			}
		}
		if match < 0 {
			out = append(out, u)
			continue
		}
		out[match].hasTokens = true
		out[match].toks, out[match].cost = u.toks, u.cost
		if u.model != "" {
			out[match].model = u.model
		}
	}
	for i := range out {
		if out[i].model == "" {
			out[i].model = model
		}
		if out[i].at.IsZero() {
			out[i].at = out[i].end
		}
	}
	return out
}

// grokCwd decodes the folder Grok url-encodes as a directory name. A name
// that does not decode to an absolute path stays as-is: it will match no
// claimed workspace, which is the right verdict for a folder we cannot
// place.
func grokCwd(path string) string {
	return grokDecodeDir(filepath.Base(filepath.Dir(path)))
}

// grokDirCwd is grokCwd for a path two levels below the encoded folder: a
// session directory is <encoded cwd>/<session id>/.
func grokDirCwd(dir string) string {
	return grokDecodeDir(filepath.Base(filepath.Dir(dir)))
}

func grokDecodeDir(name string) string {
	if dec := clisession.DecodePathDir(name); filepath.IsAbs(dec) {
		return dec
	}
	return name
}
