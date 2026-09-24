package climetrics

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/cfpperche/picode/internal/session"
)

// countingMeter counts how often it is metered; its fingerprint and cost
// are read through pointers so a test can move them between polls.
type countingMeter struct {
	cli   string
	fp    *string
	cost  *float64
	err   *error
	calls *int32
}

func (m countingMeter) CLI() string         { return m.cli }
func (m countingMeter) Label() string       { return m.cli }
func (m countingMeter) Fingerprint() string { return *m.fp }
func (m countingMeter) Meter(Request) (Window, error) {
	atomic.AddInt32(m.calls, 1)
	if *m.err != nil {
		return Window{}, *m.err
	}
	return Window{
		Stats:    session.WindowStats{Current: session.PeriodTotals{Cost: *m.cost}},
		Coverage: CoverageRow{CLI: m.cli, Label: m.cli, Signals: map[Signal]State{SigCost: StateReported}},
	}, nil
}

func newCounting(cli, fp string, cost float64) (countingMeter, *string, *float64, *error, *int32) {
	f, c, e, n := fp, cost, error(nil), int32(0)
	return countingMeter{cli: cli, fp: &f, cost: &c, err: &e, calls: &n}, &f, &c, &e, &n
}

// One row per way AggregateCached decides to re-meter or reuse.
func TestAggregateCached(t *testing.T) {
	a, aFP, aCost, aErr, aCalls := newCounting("a", "a1", 1)
	b, _, _, _, bCalls := newCounting("b", "b1", 2)
	meters := []Meter{a, b}
	var c WindowCache
	req := testReq()
	calls := func() (int32, int32) { return atomic.LoadInt32(aCalls), atomic.LoadInt32(bCalls) }

	// Cold: both metered.
	if st := AggregateCached(req, meters, &c, "k"); st.Current.Cost != 3 {
		t.Fatalf("cold cost = %v", st.Current.Cost)
	}
	if x, y := calls(); x != 1 || y != 1 {
		t.Fatalf("cold calls = %d %d", x, y)
	}
	// Nothing moved: nothing re-metered, same answer.
	if st := AggregateCached(req, meters, &c, "k"); st.Current.Cost != 3 {
		t.Fatalf("warm cost = %v", st.Current.Cost)
	}
	if x, y := calls(); x != 1 || y != 1 {
		t.Fatalf("warm calls = %d %d", x, y)
	}
	// One CLI wrote a line: only it is re-metered, and its new number shows.
	*aFP, *aCost = "a2", 10
	if st := AggregateCached(req, meters, &c, "k"); st.Current.Cost != 12 {
		t.Fatalf("one dirty cost = %v", st.Current.Cost)
	}
	if x, y := calls(); x != 2 || y != 1 {
		t.Fatalf("one dirty calls = %d %d", x, y)
	}
	// Another window key, or another request, is its own entry.
	AggregateCached(req, meters, &c, "other")
	r2 := req
	r2.Hourly = true
	AggregateCached(r2, meters, &c, "k")
	if x, y := calls(); x != 4 || y != 3 {
		t.Fatalf("new key/request calls = %d %d", x, y)
	}
	// An empty fingerprint never hits. (The key holds one request, so b —
	// last metered for r2 — is metered once more for req here; the server's
	// key is root|range, one request per range.)
	*aFP = ""
	AggregateCached(req, meters, &c, "k")
	AggregateCached(req, meters, &c, "k")
	if x, _ := calls(); x != 6 {
		t.Fatalf("empty fingerprint calls = %d, want every poll", x)
	}
	// A failure is not cached: the next poll tries again.
	*aFP, *aErr = "a3", errors.New("boom")
	st := AggregateCached(req, meters, &c, "k")
	if len(st.Coverage) != 2 || st.Coverage[0].Note == "" && st.Coverage[1].Note == "" {
		t.Fatalf("failed meter row missing: %+v", st.Coverage)
	}
	*aErr = nil
	AggregateCached(req, meters, &c, "k")
	if x, _ := calls(); x != 8 {
		t.Fatalf("after a failure calls = %d, want a retry", x)
	}
	// A folder-filtered request bypasses the cache.
	r3 := req
	r3.KeepCwd = func(string) bool { return true }
	AggregateCached(r3, meters, &c, "k")
	if x, y := calls(); x != 9 || y != 5 {
		t.Fatalf("KeepCwd calls = %d %d", x, y)
	}
}
