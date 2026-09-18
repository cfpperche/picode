package climetrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
)

// OmpMeter reports omp (oh-my-pi) from ~/.omp/agent/sessions, the same
// bucket-per-cwd JSONL tree clisession.OmpSource lists.
//
// omp is pi-family at schema version 3, and its assistant messages carry
// what pi's carry: provider, model, a usage block with a per-type cost
// split, a stopReason on every turn — plus two things pi never writes, a
// per-turn duration in milliseconds and ttft. Duration feeds SessionMs
// (agent-busy time, prorated against the window like every lifetime
// figure); ttft has no home in Timing and is not read.
//
// Timestamps are epoch millis on message records (JS Date.now()
// convention, like pi) and RFC3339 on session records; mtime stays the
// fallback for lines that carry none.
type OmpMeter struct{}

func (OmpMeter) CLI() string   { return "omp" }
func (OmpMeter) Label() string { return "Omp" }

func (OmpMeter) Fingerprint() string { return session.Fingerprint(clisession.OmpSessionsRoot()) }

func (m OmpMeter) Meter(req Request) (Window, error) {
	root := clisession.OmpSessionsRoot()
	if root == "" {
		return absentWindow(m, req), nil
	}
	ents, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return absentWindow(m, req), nil
	}
	if err != nil {
		return Window{}, err
	}

	acc := newGuestAcc(req, m.CLI())
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			// A file untouched since before the widened window cannot hold
			// an in-window message: cheap to check, expensive to parse.
			if !req.PriorFrom.IsZero() && info.ModTime().Before(req.PriorFrom) {
				continue
			}
			path := filepath.Join(dir, f.Name())
			mtime := info.ModTime()
			replay(cachedParse(path, func(p string) *parsed { return ompParse(p, mtime) }), acc, req)
		}
	}

	w := Window{CLI: m.CLI(), Stats: acc.result(), Coverage: ompCoverage(m, req.BillingFor(m.CLI()), acc)}
	if acc.timing != (Timing{}) {
		t := acc.timing
		w.Timing = &t
	}
	return w, nil
}

// ompCan is what omp is capable of recording: cost, tokens, tools and
// turns natively, turn durations, but no edit counts or quota windows.
var ompCan = map[Signal]bool{
	SigCost: true, SigTokens: true, SigModel: true, SigMessages: true,
	SigTurns: true, SigTools: true, SigErrors: true, SigTiming: true,
}

func ompCoverage(m OmpMeter, b Billing, acc *guestAcc) CoverageRow {
	return CoverageRow{
		CLI: m.CLI(), Label: m.Label(), Billing: b,
		Signals: acc.evidence(ompCan, map[Signal]bool{SigTiming: acc.timing.SessionMs != 0}),
		Note:    "Turn durations feed agent time; it records no edit counts or quota windows.",
	}
}

type ompMessage struct {
	Role       string      `json:"role"`
	Provider   string      `json:"provider"`
	Model      string      `json:"model"`
	StopReason string      `json:"stopReason"`
	Timestamp  json.Number `json:"timestamp"` // epoch millis, like pi
	Duration   float64     `json:"duration"`  // one turn's agent-busy ms
	Content    []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"content"`
	Usage *struct {
		Input      int64 `json:"input"`
		Output     int64 `json:"output"`
		CacheRead  int64 `json:"cacheRead"`
		CacheWrite int64 `json:"cacheWrite"`
		Reasoning  int64 `json:"reasoningTokens"`
		Cost       *struct {
			Input      float64 `json:"input"`
			Output     float64 `json:"output"`
			CacheRead  float64 `json:"cacheRead"`
			CacheWrite float64 `json:"cacheWrite"`
			Total      float64 `json:"total"`
		} `json:"cost"`
	} `json:"usage"`
}

// ompParse adapts omp's schema-v3 JSONL to the shared cache shape.
func ompParse(path string, mtime time.Time) *parsed {
	f, err := os.Open(path)
	if err != nil {
		return &parsed{}
	}
	defer f.Close()

	out := &parsed{}
	cwd, name, provider, model := "", "", "", ""
	add := func(e guestEntry) {
		out.ents = append(out.ents, e)
		out.units += e.toks.Input + e.toks.Output + e.toks.CacheRead + e.toks.CacheWrite
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var rec struct {
			Type      string      `json:"type"`
			ID        string      `json:"id"`
			Timestamp string      `json:"timestamp"`
			Cwd       string      `json:"cwd"`
			Title     string      `json:"title"`
			Model     string      `json:"model"`
			Message   *ompMessage `json:"message"`
		}
		if json.Unmarshal(line, &rec) != nil {
			continue
		}
		switch rec.Type {
		case "session":
			if rec.Cwd != "" {
				cwd = rec.Cwd
			}
		case "title":
			if t := strings.TrimSpace(rec.Title); t != "" {
				name = t
			}
		case "model_change":
			// omp qualifies the fallback ("zai/glm-5.3-flash"); the
			// message's own provider/model win whenever present.
			if p, mm, ok := strings.Cut(rec.Model, "/"); ok {
				if provider == "" {
					provider = p
				}
				model = mm
			} else if rec.Model != "" {
				model = rec.Model
			}
		case "compaction", "compaction_summary":
			out.compactions = append(out.compactions, compaction{at: ompTime(rec.Timestamp, mtime), cwd: cwd})
		case "message":
			if rec.Message == nil {
				continue
			}
			m := rec.Message
			t := mtime
			if ms, err := m.Timestamp.Int64(); err == nil && ms > 0 {
				t = time.UnixMilli(ms)
			}
			prov := m.Provider
			if prov == "" {
				prov = provider
			}
			mm := m.Model
			if mm != "" {
				model = mm
			} else {
				mm = model
			}
			var toks session.TokenTotals
			var split session.CostSplit
			var cost float64
			if m.Usage != nil {
				u := m.Usage
				toks = session.TokenTotals{
					Input: u.Input, Output: u.Output,
					CacheRead: u.CacheRead, CacheWrite: u.CacheWrite,
					Reasoning: u.Reasoning,
				}
				if u.Cost != nil {
					c := u.Cost
					split = session.CostSplit{
						Input: c.Input, Output: c.Output,
						CacheRead: c.CacheRead, CacheWrite: c.CacheWrite,
					}
					cost = c.Total
				}
			}
			var tools []string
			for _, b := range m.Content {
				// Same block vocabulary as pi; anything else is skipped,
				// never merged into an equivalence we cannot back.
				if b.Type == "toolCall" && b.Name != "" {
					tools = append(tools, b.Name)
				}
			}
			e := guestEntry{
				at: t, key: path, cwd: cwd, name: name,
				role: m.Role, model: mm, prov: prov,
				cost: cost, split: split, toks: toks, tools: tools,
				abort: m.StopReason == "aborted",
				// omp stamps a stopReason on every assistant turn, so each
				// one is an inspected outcome whether or not it failed.
				results: boolToInt(m.Role == "assistant" && m.StopReason != ""),
				errs:    boolToInt(m.StopReason == "error"),
			}
			add(e)
			if m.Duration > 0 {
				out.timing.SessionMs += int64(m.Duration)
			}
		}
	}
	// The cwd only becomes known at the session record, which is near the
	// top; entries built before it borrow the latest truth, the way the
	// codex adapter backfills its own.
	for i := range out.ents {
		if out.ents[i].cwd == "" {
			out.ents[i].cwd = cwd
		}
	}
	return out
}

// ompTime reads an RFC3339 timestamp with an mtime fallback, for the
// session-level records that carry text rather than millis.
func ompTime(raw string, fallback time.Time) time.Time {
	if raw == "" {
		return fallback
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	return fallback
}
