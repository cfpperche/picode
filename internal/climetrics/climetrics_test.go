package climetrics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

// fakeMeter is a Meter that returns whatever a test hands it, so merge can
// be exercised without touching any CLI's files.
type fakeMeter struct {
	cli, label string
	fp         string
	win        Window
	err        error
}

func (f fakeMeter) CLI() string         { return f.cli }
func (f fakeMeter) Label() string       { return f.label }
func (f fakeMeter) Fingerprint() string { return f.fp }
func (f fakeMeter) Meter(Request) (Window, error) {
	if f.err != nil {
		return Window{}, f.err
	}
	w := f.win
	w.CLI = f.cli
	if w.Coverage.CLI == "" {
		w.Coverage = CoverageRow{CLI: f.cli, Label: f.label, Signals: map[Signal]State{SigCost: StateReported}}
	}
	return w, nil
}

func testReq() Request {
	to := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	from := to.AddDate(0, 0, -3)
	return Request{From: from, To: to, PriorFrom: from.AddDate(0, 0, -3), Loc: time.UTC}
}

func TestAggregateSumsAcrossCLIs(t *testing.T) {
	a := fakeMeter{cli: "pi", label: "Pi", win: Window{Stats: session.WindowStats{
		Current: session.PeriodTotals{Cost: 10, Messages: 4, Sessions: 2},
		Tokens:  session.TokenTotals{Input: 100, CacheRead: 300},
		Turns:   session.TurnStats{Assistant: 3, Errors: 1},
	}}}
	b := fakeMeter{cli: "claude-code", label: "Claude Code", win: Window{Stats: session.WindowStats{
		Current: session.PeriodTotals{Cost: 90, Messages: 6, Sessions: 1},
		Tokens:  session.TokenTotals{Input: 100, CacheRead: 500},
		Turns:   session.TurnStats{Assistant: 5, Aborted: 2},
	}}}

	out := Aggregate(testReq(), []Meter{a, b})
	if out.Current.Cost != 100 || out.Current.Messages != 10 || out.Current.Sessions != 3 {
		t.Fatalf("current = %+v", out.Current)
	}
	if out.Turns.Assistant != 8 || out.Turns.Errors != 1 || out.Turns.Aborted != 2 {
		t.Fatalf("turns = %+v", out.Turns)
	}
	// Cache hit is recomputed over the union, never averaged from the parts.
	if out.Tokens.CacheHit == nil || !approx(*out.Tokens.CacheHit, 80) {
		t.Fatalf("cacheHit = %v, want 80", out.Tokens.CacheHit)
	}
	if len(out.ByCLI) != 2 || out.ByCLI[0].CLI != "claude-code" {
		t.Fatalf("byCli should rank by cost: %+v", out.ByCLI)
	}
}

// TestAggregateRanksToolsOnceOverTheUnion is why adapters return uncapped
// windows: pi's ninth tool has to meet Claude Code's first before anything
// is cut.
func TestAggregateRanksToolsOnceOverTheUnion(t *testing.T) {
	var piTools []session.ToolBucket
	for i, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"} {
		piTools = append(piTools, session.ToolBucket{Name: n, Calls: 100 - i})
	}
	a := fakeMeter{cli: "pi", win: Window{Stats: session.WindowStats{Tools: piTools}}}
	b := fakeMeter{cli: "claude-code", win: Window{Stats: session.WindowStats{
		Tools: []session.ToolBucket{{Name: "Bash", Calls: 1000}},
	}}}

	out := Aggregate(testReq(), []Meter{a, b})
	if len(out.Tools) != session.TopTools {
		t.Fatalf("tools = %d, want %d", len(out.Tools), session.TopTools)
	}
	if out.Tools[0].Name != "Bash" || out.Tools[0].CLI != "claude-code" {
		t.Fatalf("the biggest tool across both CLIs should lead: %+v", out.Tools[0])
	}
}

// TestAggregateKeepsToolNamesPerCLI locks in a deliberate refusal: "bash"
// and "Bash" are two vendors' words and are never folded together.
func TestAggregateKeepsToolNamesPerCLI(t *testing.T) {
	a := fakeMeter{cli: "pi", win: Window{Stats: session.WindowStats{
		Tools: []session.ToolBucket{{Name: "bash", Calls: 10}},
	}}}
	b := fakeMeter{cli: "claude-code", win: Window{Stats: session.WindowStats{
		Tools: []session.ToolBucket{{Name: "Bash", Calls: 5}},
	}}}
	out := Aggregate(testReq(), []Meter{a, b})
	if len(out.Tools) != 2 {
		t.Fatalf("tools = %+v, want both spellings kept apart", out.Tools)
	}
}

func TestAggregateKeepsSameModelPerCLIApart(t *testing.T) {
	mk := func(cli string, cost float64) fakeMeter {
		return fakeMeter{cli: cli, win: Window{Stats: session.WindowStats{
			ByModel: []session.ModelBucket{{Provider: "anthropic", Model: "claude-opus-5", Cost: cost, Messages: 1}},
		}}}
	}
	out := Aggregate(testReq(), []Meter{mk("pi", 3), mk("claude-code", 7)})
	if len(out.ByModel) != 2 {
		t.Fatalf("byModel = %+v, want one row per CLI", out.ByModel)
	}
	if out.ByModel[0].CLI != "claude-code" || out.ByModel[0].Cost != 7 {
		t.Fatalf("byModel[0] = %+v", out.ByModel[0])
	}
}

func TestAggregateMergesWorkspacesAcrossCLIs(t *testing.T) {
	mk := func(cli string, cost float64) fakeMeter {
		return fakeMeter{cli: cli, win: Window{Stats: session.WindowStats{
			ByWorkspace: []session.WorkspaceBucket{{Cwd: "/repo", Cost: cost, Messages: 1, Sessions: 1}},
		}}}
	}
	out := Aggregate(testReq(), []Meter{mk("pi", 3), mk("claude-code", 7)})
	if len(out.ByWorkspace) != 1 || out.ByWorkspace[0].Cost != 10 || out.ByWorkspace[0].Sessions != 2 {
		t.Fatalf("byWorkspace = %+v, want one folder holding both CLIs", out.ByWorkspace)
	}
}

// TestAggregateAlwaysAnswersCoverage is the anti-silent-zero gate: a CLI
// that reported nothing still owes the surface a row saying so.
func TestAggregateAlwaysAnswersCoverage(t *testing.T) {
	quiet := fakeMeter{cli: "grok", label: "Grok", win: Window{
		Coverage: CoverageRow{CLI: "grok", Label: "Grok", Signals: map[Signal]State{
			SigCost: StateNotReported, SigTokens: StateNotReported,
		}, Note: "Grok records prompt history only."},
	}}
	out := Aggregate(testReq(), []Meter{quiet})
	if len(out.Coverage) != 1 || out.Coverage[0].Note == "" {
		t.Fatalf("coverage = %+v", out.Coverage)
	}
	if len(out.ByCLI) != 1 || out.ByCLI[0].CostState != StateNotReported {
		t.Fatalf("byCli must carry the cost state, not a bare 0: %+v", out.ByCLI)
	}
}

// TestAggregateSurvivesOneBrokenCLI: an unreadable store degrades to a
// coverage row, it does not take the dashboard down.
func TestAggregateSurvivesOneBrokenCLI(t *testing.T) {
	good := fakeMeter{cli: "pi", label: "Pi", win: Window{Stats: session.WindowStats{
		Current: session.PeriodTotals{Cost: 5, Messages: 1},
	}}}
	bad := fakeMeter{cli: "hermes", label: "Hermes Agent", err: errTest{}}

	out := Aggregate(testReq(), []Meter{good, bad})
	if out.Current.Cost != 5 {
		t.Fatalf("a broken CLI must not erase a working one: %+v", out.Current)
	}
	var row *CoverageRow
	for i := range out.Coverage {
		if out.Coverage[i].CLI == "hermes" {
			row = &out.Coverage[i]
		}
	}
	if row == nil || row.Signals[SigCost] != StateUnavailable {
		t.Fatalf("hermes coverage = %+v, want unavailable", row)
	}
}

type errTest struct{}

func (errTest) Error() string { return "database is locked" }

func TestSeriesZeroFillsEveryDay(t *testing.T) {
	m := fakeMeter{cli: "pi", win: Window{Stats: session.WindowStats{
		Series: []session.DayBucket{{Date: "2026-09-06", Cost: 1, Messages: 1}},
	}}}
	out := Aggregate(testReq(), []Meter{m})
	if len(out.Series) != 3 {
		t.Fatalf("series = %+v, want one entry per calendar day", out.Series)
	}
	var filled int
	for _, d := range out.Series {
		if d.Cost > 0 {
			filled++
		}
	}
	if filled != 1 {
		t.Fatalf("only the day with data should carry cost: %+v", out.Series)
	}
}

// The decision table for bucketing: a day window fills days, an hourly window
// fills one bucket per real hour, and a fall-back hour (which repeats a clock
// name) is one bucket rather than two labels on the same bar.
func TestSeriesBucketsByDayOrByHour(t *testing.T) {
	day := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		req    Request
		wins   int
		first  string
		last   string
		filled string // the key the meter reported data for
	}{
		{
			name:   "day window fills one bucket per calendar day",
			req:    Request{From: day, To: day.AddDate(0, 0, 3), Loc: time.UTC},
			wins:   3,
			first:  "2026-09-06",
			last:   "2026-09-08",
			filled: "2026-09-07",
		},
		{
			name:   "hourly window fills one bucket per hour of the day",
			req:    Request{From: day, To: day.AddDate(0, 0, 1), Loc: time.UTC, Hourly: true},
			wins:   24,
			first:  "2026-09-06T00",
			last:   "2026-09-06T23",
			filled: "2026-09-06T14",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fakeMeter{cli: "pi", win: Window{Stats: session.WindowStats{
				Series: []session.DayBucket{{Date: tc.filled, Cost: 2, Messages: 1, Turns: 1}},
			}}}
			out := Aggregate(tc.req, []Meter{m})
			if len(out.Series) != tc.wins {
				t.Fatalf("buckets = %d, want %d: %+v", len(out.Series), tc.wins, out.Series)
			}
			if out.Series[0].Date != tc.first || out.Series[len(out.Series)-1].Date != tc.last {
				t.Fatalf("first/last = %s/%s, want %s/%s", out.Series[0].Date, out.Series[len(out.Series)-1].Date, tc.first, tc.last)
			}
			var filled int
			for _, b := range out.Series {
				if b.Cost > 0 {
					filled++
				}
				if b.Date == "" {
					t.Fatal("a bucket without a key")
				}
			}
			if filled != 1 {
				t.Fatalf("only the reported bucket carries data: %+v", out.Series)
			}
		})
	}
}

func TestSeriesKeysCarryTheirOwnGranularity(t *testing.T) {
	at := time.Date(2026, 9, 6, 14, 30, 0, 0, time.UTC)
	if got := (Request{Loc: time.UTC}).SeriesKey(at); got != "2026-09-06" {
		t.Fatalf("day key = %q", got)
	}
	if got := (Request{Loc: time.UTC, Hourly: true}).SeriesKey(at); got != "2026-09-06T14" {
		t.Fatalf("hour key = %q", got)
	}
	// The key is in the caller's zone, like every other figure in the payload.
	west := time.FixedZone("UTC-3", -3*3600)
	if got := (Request{Loc: west, Hourly: true}).SeriesKey(at); got != "2026-09-06T11" {
		t.Fatalf("hour key in UTC-3 = %q", got)
	}
}

// A fall-back day repeats one clock hour; two buckets named 01:00 would be
// two bars under one label, so the hour merges instead.
func TestHourlySeriesMergesARepeatedDSTHour(t *testing.T) {
	loc := time.FixedZone("fake", -3600)
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, loc)
	series := fillSeries(map[string]*session.DayBucket{}, Request{From: from, To: from.Add(25 * time.Hour), Loc: loc, Hourly: true})
	seen := map[string]bool{}
	for _, b := range series {
		if seen[b.Date] {
			t.Fatalf("duplicate bucket %s in %d buckets", b.Date, len(series))
		}
		seen[b.Date] = true
	}
	if len(series) != 25 {
		t.Fatalf("a 25-hour window is 25 hours under 24 names: got %d buckets", len(series))
	}
}

func TestFingerprintMissesWhenAMeterCannotDescribeItself(t *testing.T) {
	known := fakeMeter{cli: "pi", fp: "1:2:3"}
	unknown := fakeMeter{cli: "codex", fp: ""}
	if got := Fingerprint([]Meter{known}); got == "" {
		t.Fatal("a describable meter must produce a fingerprint")
	}
	if got := Fingerprint([]Meter{known, unknown}); got != "" {
		t.Fatalf("fingerprint = %q, want empty so the cache misses", got)
	}
}

// TestImpactCarriesNoFileCount locks in a deliberate omission. Only
// OpenCode counts changed files; summing it beside Claude Code's line
// counts produced "0 files" next to 21,346 changed lines on the machine
// this was built on, because the one CLI that counts files had touched
// none. A number describing 4 messages out of 59,043 is not a fleet metric.
func TestImpactCarriesNoFileCount(t *testing.T) {
	m := fakeMeter{cli: "claude-code", win: Window{Impact: &Impact{LinesAdded: 10, LinesRemoved: 2}}}
	out := Aggregate(testReq(), []Meter{m})
	blob, err := json.Marshal(out.Impact)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(blob); got != `{"linesAdded":10,"linesRemoved":2}` {
		t.Fatalf("impact = %s, want lines only", got)
	}
}
