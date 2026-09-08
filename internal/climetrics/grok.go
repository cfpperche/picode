package climetrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
)

// GrokMeter reports the Grok CLI, which records prompt history and nothing
// else: one line per prompt with a timestamp and a session id, under
// ~/.grok/sessions/<encoded cwd>/prompt_history.jsonl.
//
// So this meter counts prompts and reports every other signal as
// not-reported. That is the whole point of the coverage contract: a CLI
// that is installed and nearly silent is information, and hiding its row
// would recreate exactly the invisibility ADR-0097 exists to fix. A Grok
// row reading "— activity only" tells the operator something true; no row
// at all tells them nothing.
type GrokMeter struct{}

func (GrokMeter) CLI() string   { return "grok" }
func (GrokMeter) Label() string { return "Grok" }

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
		if err != nil || d.IsDir() || d.Name() != "prompt_history.jsonl" {
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
	acc := newGuestAcc(req, m.CLI())
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "prompt_history.jsonl" {
			return nil
		}
		replay(cachedParse(p, grokParse), acc, req)
		return nil
	})
	return Window{
		CLI:      m.CLI(),
		Stats:    acc.result(),
		Coverage: grokCoverage(m, req.BillingFor(m.CLI()), acc),
	}, nil
}

// grokCan is what the Grok CLI records: prompts, and nothing else.
var grokCan = map[Signal]bool{SigMessages: true}

func grokCoverage(m GrokMeter, b Billing, acc *guestAcc) CoverageRow {
	return CoverageRow{
		CLI: m.CLI(), Label: m.Label(), Billing: b,
		Signals: acc.evidence(grokCan, nil),
		Note:    "Records prompt history only — no cost, tokens, model or tool calls reach disk. Prompt counts are real; everything else is unmeasured, not zero.",
	}
}

// grokParse reads one folder's prompt history. The prompt text itself is
// never touched: only its timestamp and session id.
func grokParse(path string) *parsed {
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
		out.ents = append(out.ents, guestEntry{at: at, key: key, cwd: cwd, role: "user"})
	}
	return out
}

// grokCwd decodes the folder Grok url-encodes as a directory name. A name
// that does not decode to an absolute path stays as-is: it will match no
// claimed workspace, which is the right verdict for a folder we cannot
// place.
func grokCwd(path string) string {
	name := filepath.Base(filepath.Dir(path))
	if dec := clisession.DecodePathDir(name); filepath.IsAbs(dec) {
		return dec
	}
	return name
}
