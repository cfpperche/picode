package climetrics

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestProbeRealTree is a manual harness, not a gate: it prints what the
// meters find on the developer's own machine. Skipped unless CLIMETRICS_PROBE=1.
func TestProbeRealTree(t *testing.T) {
	if os.Getenv("CLIMETRICS_PROBE") != "1" {
		t.Skip("set CLIMETRICS_PROBE=1")
	}
	now := time.Now()
	loc := time.Local
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	from := to.AddDate(0, 0, -7)
	req := Request{From: from, To: to, PriorFrom: from.AddDate(0, 0, -7), Loc: loc, Scope: ScopeMachine}
	out := Aggregate(req, []Meter{PiMeter{}, ClaudeCodeMeter{}})
	b, _ := json.MarshalIndent(struct {
		Current  any `json:"current"`
		ByCLI    any `json:"byCli"`
		Coverage any `json:"coverage"`
		Impact   any `json:"impact"`
		Timing   any `json:"timing"`
		Tools    any `json:"tools"`
		Models   any `json:"byModel"`
	}{out.Current, out.ByCLI, out.Coverage, out.Impact, out.Timing, out.Tools, out.ByModel}, "", " ")
	t.Log("\n" + string(b))
}
