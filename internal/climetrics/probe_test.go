package climetrics

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/pricing"
)

// The two tests here are harnesses, not gates. They read the developer's own
// session stores, so they are skipped unless CLIMETRICS_PROBE=1 — but they
// are committed because every measurement quoted in ADR-0097 and the
// benchmark study came out of them, and a claim nobody can reproduce is
// just an assertion.

// TestProbeRealTree is a manual harness, not a gate: it prints what the
// meters find on the developer's own machine. Skipped unless CLIMETRICS_PROBE=1.
func TestProbeRealTree(t *testing.T) {
	if os.Getenv("CLIMETRICS_PROBE") != "1" {
		t.Skip("set CLIMETRICS_PROBE=1")
	}
	now := time.Now()
	loc := time.Local
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	days := 7 // CLIMETRICS_PROBE_DAYS widens it, to match a dashboard range
	if n, err := strconv.Atoi(os.Getenv("CLIMETRICS_PROBE_DAYS")); err == nil && n > 0 {
		days = n
	}
	from := to.AddDate(0, 0, -days)
	req := Request{From: from, To: to, PriorFrom: from.AddDate(0, 0, -days), Loc: loc}
	// CLIMETRICS_PROBE_PRICES names a LiteLLM table on disk, to see the
	// list-price estimates (ADR-0185) the server would add.
	if path := os.Getenv("CLIMETRICS_PROBE_PRICES"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if req.Prices, err = pricing.Parse(raw); err != nil {
			t.Fatal(err)
		}
	}
	out := Aggregate(req, Meters())
	b, _ := json.MarshalIndent(struct {
		Current  any `json:"current"`
		ByCLI    any `json:"byCli"`
		Coverage any `json:"coverage"`
		Impact   any `json:"impact"`
		Timing   any `json:"timing"`
		Tools    any `json:"tools"`
		Limits   any `json:"limits"`
		Models   any `json:"byModel"`
	}{out.Current, out.ByCLI, out.Coverage, out.Impact, out.Timing, out.Tools, out.Limits, out.ByModel}, "", " ")
	t.Log("\n" + string(b))
}

func TestProbePerf(t *testing.T) {
	if os.Getenv("CLIMETRICS_PROBE") != "1" {
		t.Skip("set CLIMETRICS_PROBE=1")
	}
	ms := Meters()

	t0 := time.Now()
	fp := Fingerprint(ms)
	t.Logf("fingerprint (stat sweep, all six): %v  len=%d", time.Since(t0), len(fp))

	now := time.Now()
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

	warm := func(name string, req Request) {
		t0 := time.Now()
		out := Aggregate(req, ms)
		t.Logf("WARM %-6s %8v  cost=$%.2f", name, time.Since(t0), out.Current.Cost)
		for _, one := range ms {
			t1 := time.Now()
			_, _ = one.Meter(req)
			t.Logf("      %-12s %8v", one.CLI(), time.Since(t1))
		}
	}
	for _, c := range []struct {
		name string
		req  Request
	}{
		{"today", Request{From: to.AddDate(0, 0, -1), To: to, PriorFrom: to.AddDate(0, 0, -2), Loc: time.Local}},
		{"7d", Request{From: to.AddDate(0, 0, -7), To: to, PriorFrom: to.AddDate(0, 0, -14), Loc: time.Local}},
		{"all", Request{To: to, Loc: time.Local}},
	} {
		t0 := time.Now()
		out := Aggregate(c.req, ms)
		t.Logf("cold %-6s %8v  cost=$%.2f msgs=%d sessions=%d", c.name, time.Since(t0), out.Current.Cost, out.Current.Messages, out.Current.Sessions)
		warm(c.name, c.req)
	}
}
