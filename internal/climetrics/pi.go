package climetrics

import (
	"os"
	"path/filepath"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

// PiMeter reports pi's own sessions.
//
// It adds no dialect knowledge: internal/session already parses pi's JSONL
// for the v1 dashboard and carries its own tests, so this adapter drives
// session.ParseFile and files the result through the same cache and
// accumulator every guest CLI uses.
//
// Routing pi through the cache is not a tidiness point. Measured on this
// machine once the five guests were cached, pi was 2.32 s of a 2.37 s warm
// refresh — 98% of the cost, because it alone re-read 437 MB on every poll.
//
// pi is the one CLI that records cost, tokens, tools and turns natively and
// the one CLI that records nothing about lines changed, request duration or
// quota windows — the exact inverse of Claude Code and Codex.
type PiMeter struct{}

func (PiMeter) CLI() string   { return "pi" }
func (PiMeter) Label() string { return "Pi" }

func (PiMeter) Fingerprint() string { return session.Fingerprint(session.Root()) }

func (m PiMeter) Meter(req Request) (Window, error) {
	root := session.Root()
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
			replay(cachedParse(path, func(p string) *parsed { return piParse(p, mtime) }), acc, req)
		}
	}

	billing := req.BillingFor(m.CLI())
	st := acc.result()
	// pi is the one CLI whose credentials PiCode holds, so it is the one
	// that belongs in the Providers view's per-key spend.
	st.ByProvider = piProviders(acc, billing)
	return Window{
		CLI:      m.CLI(),
		Stats:    st,
		Coverage: piCoverage(m, billing),
	}, nil
}

// piParse adapts session.ParseFile to the shared cache shape.
func piParse(path string, mtime time.Time) *parsed {
	fs := session.ParseFile(path, mtime)
	out := &parsed{ents: make([]guestEntry, 0, len(fs.Entries))}
	for _, c := range fs.Compactions {
		out.compactions = append(out.compactions, compaction{at: c.At, cwd: c.Cwd})
	}
	for _, e := range fs.Entries {
		out.ents = append(out.ents, guestEntry{
			at: e.At, key: path, cwd: e.Cwd, name: e.Name,
			role: e.Role, model: e.Model, prov: e.Provider,
			cost: e.Cost, split: e.Split, toks: e.Usage, tools: e.Tools,
			abort: e.StopReason == "aborted",
			errs:  boolToInt(e.StopReason == "error"),
		})
		out.units += e.Usage.Input + e.Usage.Output + e.Usage.CacheRead + e.Usage.CacheWrite
	}
	return out
}

// piProviders rebuilds the per-provider rows from the model breakdown. Only
// pi contributes them: the Providers view joins these onto the credentials
// PiCode holds, and a guest CLI signs in with its own account.
func piProviders(acc *guestAcc, b Billing) []session.ProviderBucket {
	byProv := map[string]*session.ProviderBucket{}
	for _, m := range acc.byModel {
		p := byProv[m.Provider]
		if p == nil {
			p = &session.ProviderBucket{Provider: m.Provider, Billing: string(b)}
			byProv[m.Provider] = p
		}
		p.Cost += m.Cost
		p.Messages += m.Messages
	}
	out := make([]session.ProviderBucket, 0, len(byProv))
	for _, p := range byProv {
		out = append(out, *p)
	}
	return out
}

func piCoverage(m PiMeter, b Billing) CoverageRow {
	return CoverageRow{
		CLI:     m.CLI(),
		Label:   m.Label(),
		Billing: b,
		Signals: map[Signal]State{
			SigCost:     StateReported,
			SigTokens:   StateReported,
			SigModel:    StateReported,
			SigMessages: StateReported,
			SigTurns:    StateReported,
			SigTools:    StateReported,
			SigErrors:   StateReported,
			SigImpact:   StateNotReported,
			SigTiming:   StateNotReported,
			SigLimits:   StateNotReported,
		},
		Note: "pi records no edit counts, request durations or quota windows.",
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
