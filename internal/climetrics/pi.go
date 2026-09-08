package climetrics

import (
	"github.com/cfpperche/picode/internal/session"
)

// PiMeter reports pi's own sessions. It adds no parser: internal/session
// already scans pi's JSONL for the v1 dashboard and carries its own tests,
// so this adapter runs that scan uncapped and labels the result.
//
// pi is the one CLI that records cost, tokens, tools and turns natively and
// the one CLI that records nothing about lines changed, request duration or
// quota windows — the exact inverse of Claude Code and Codex.
type PiMeter struct{}

func (PiMeter) CLI() string   { return "pi" }
func (PiMeter) Label() string { return "Pi" }

func (PiMeter) Fingerprint() string { return session.Fingerprint(session.Root()) }

func (m PiMeter) Meter(req Request) (Window, error) {
	var keep func(string) bool
	if req.Scope == ScopePiCode {
		keep = req.InScope
	}
	st, err := session.StatsRootFiltered(session.Root(), req.From, req.To, req.PriorFrom, req.Loc, keep)
	if err != nil {
		return Window{}, err
	}
	billing := req.BillingFor(m.CLI())
	for i := range st.ByProvider {
		st.ByProvider[i].Billing = string(billing)
	}
	return Window{
		CLI:      m.CLI(),
		Stats:    st,
		Coverage: piCoverage(m, billing),
	}, nil
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
