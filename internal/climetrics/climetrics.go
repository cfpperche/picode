// Package climetrics aggregates what every agent CLI on this machine
// recorded about its own sessions into one dashboard window (ADR-0097).
//
// It exists because the dashboard measured pi and only pi while PiCode ran
// nine CLIs, so the headline number was a fraction of the real one. Each
// adapter reads the same session files internal/clisession already lists,
// and reports two things: what it found, and what that CLI does not record
// at all. A signal a CLI never writes comes back as StateNotReported, never
// as a zero — a dashboard that prints $0.00 for Codex is lying; one that
// prints "—" beside a reason is not.
//
// Adapters return uncapped windows. Ranking and the top-N cuts happen once,
// over the union, in Aggregate: capping per CLI would drop pi's ninth tool
// before it was ever compared with Claude Code's first.
package climetrics

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/pricing"
	"github.com/cfpperche/picode/internal/session"
)

// Signal names one metric family a CLI may or may not record.
type Signal string

const (
	SigCost     Signal = "cost"
	SigTokens   Signal = "tokens"
	SigModel    Signal = "model"
	SigMessages Signal = "messages"
	SigTurns    Signal = "turns"
	SigTools    Signal = "tools"
	SigErrors   Signal = "errors"
	SigImpact   Signal = "impact"
	SigTiming   Signal = "timing"
	SigLimits   Signal = "limits"
)

// Signals is the fixed column order every coverage row answers, so the
// coverage panel renders a matrix instead of a ragged list.
var Signals = []Signal{
	SigCost, SigTokens, SigModel, SigMessages,
	SigTurns, SigTools, SigErrors,
	SigImpact, SigTiming, SigLimits,
}

// State is what a CLI can say about one signal in this window.
type State string

const (
	// StateReported: the CLI writes this signal and the adapter read it.
	StateReported State = "reported"
	// StateNotReported: the CLI does not record this signal at all. The
	// surface must render "—" plus the reason, never 0.
	StateNotReported State = "not-reported"
	// StatePartial: the CLI records this signal, but not for every session
	// in the window. Claude Code writes cost only on a session snapshot and
	// a live session has none, so its spend is a floor, not a total. The
	// surface must show the number *and* say what it is missing.
	StatePartial State = "partial"
	// StateUnavailable: the CLI records it, but this machine could not be
	// read (missing root, unreadable database). Distinct from
	// not-reported so the fix is discoverable.
	StateUnavailable State = "unavailable"
	// StateEstimated: the CLI does not price this, and PiCode did — at list
	// rate from LiteLLM's table (ADR-0185). Cost only. The surface shows
	// the number and marks it as an estimate.
	StateEstimated State = "estimated"
)

// Billing is how a CLI's spend is actually paid for. It changes what the
// cost number means, so it is never inferred — Meters report what the CLI
// itself states, and the operator sets the rest.
type Billing string

const (
	BillingAPI          Billing = "api"          // metered against a key; real money
	BillingSubscription Billing = "subscription" // covered by a plan; list-price equivalent
	BillingUnknown      Billing = "unknown"
)

// Impact is what a window changed on disk. Only CLIs that count their own
// edits report it; the rest leave it nil.
//
// There is no file count here, on purpose. OpenCode records
// summary_files and Claude Code does not, so a merged total would cover a
// sliver of the lines beside it — on this machine, a "0 files" sitting next
// to 21,346 changed lines, because the one CLI that counts files happened
// to touch none. A number that describes 4 messages out of 59,043 is not a
// fleet metric. The capability stays visible in the coverage matrix, and
// the field comes back the day a second CLI reports it.
type Impact struct {
	LinesAdded   int64 `json:"linesAdded"`
	LinesRemoved int64 `json:"linesRemoved"`
}

// Timing separates the time a window spent waiting on a model from the time
// it spent running tools, and both from the sessions' own elapsed time.
// ADR-0042 refused latency because pi's JSONL carries no duration; Claude
// Code's does.
//
// SessionMs is *agent* time, not clock time: sessions run concurrently, so
// on this machine a 7-day window summed to 226 hours — 32 per day. A
// surface that labels it "elapsed" is lying; it is how long the fleet was
// busy, added up.
type Timing struct {
	APIMs     int64 `json:"apiMs"`
	ToolMs    int64 `json:"toolMs"`
	SessionMs int64 `json:"sessionMs"`
}

// Add sums another Timing into this one.
func (t *Timing) Add(o Timing) {
	t.APIMs += o.APIMs
	t.ToolMs += o.ToolMs
	t.SessionMs += o.SessionMs
}

// Add sums another Impact into this one.
func (i *Impact) Add(o Impact) {
	i.LinesAdded += o.LinesAdded
	i.LinesRemoved += o.LinesRemoved
}

// LimitWindow is one quota window a CLI reported against its own plan. This
// is what stands in for cost on a CLI that never prices a token: the binding
// constraint on a subscription is the window, not the bill.
type LimitWindow struct {
	CLI         string  `json:"cli"`
	Label       string  `json:"label"` // "5h" / "weekly", derived from WindowMinutes
	UsedPercent float64 `json:"usedPercent"`
	WindowMin   int     `json:"windowMinutes"`
	ResetsAt    string  `json:"resetsAt,omitempty"` // RFC3339
	Plan        string  `json:"plan,omitempty"`
	ObservedAt  string  `json:"observedAt"` // RFC3339; a quota reading is a snapshot, not a sum
}

// CoverageRow is one CLI's answer for every signal, always populated —
// including for a CLI that reported nothing this window. Silence with a
// reason is information; a missing row is not.
type CoverageRow struct {
	CLI     string           `json:"cli"`
	Label   string           `json:"label"`
	Billing Billing          `json:"billing"`
	Signals map[Signal]State `json:"signals"`
	Note    string           `json:"note,omitempty"`
}

// CLIBucket is one CLI's totals — the pivot dimension the fleet dashboard
// adds. CostState carries whether Cost means anything for this CLI, so the
// row can render "not priced" instead of a zero that reads as "free".
type CLIBucket struct {
	CLI       string  `json:"cli"`
	Label     string  `json:"label"`
	Billing   Billing `json:"billing"`
	Cost      float64 `json:"cost"`
	Estimated float64 `json:"estimated,omitempty"` // list-price part of Cost (ADR-0185)
	CostState State   `json:"costState"`
	Messages  int     `json:"messages"`
	Sessions  int     `json:"sessions"`
	Tokens    int64   `json:"tokens"`
}

// Window is one CLI's uncapped contribution to a window. Stats holds the
// facts that already have a home in session.WindowStats; the rest are the
// families only CLI agents record.
type Window struct {
	CLI      string
	Stats    session.WindowStats
	Impact   *Impact
	Timing   *Timing
	Limits   []LimitWindow
	Coverage CoverageRow
}

// Request is one aggregation window. The dashboard measures the whole
// machine (ADR-0127): which folders a product workspace claims is labelling
// in the server layer, never a filter here.
type Request struct {
	From, To, PriorFrom time.Time
	Loc                 *time.Location
	Billing             map[string]Billing // per-CLI, from the operator's cli_configs
	// KeepCwd scopes a separate consumer to session entries whose recorded
	// folder belongs to it. Nil preserves the machine-wide dashboard.
	KeepCwd func(string) bool

	// Hourly buckets the series by clock hour instead of by calendar day.
	// range=today is why it exists: one bar per day over a one-day window is a
	// single full-width block under a range picker that promises a chart
	// (the owner's 2026-09-13 screenshot). A pure function of the range, so
	// the server's root|range cache stays correct.
	Hourly bool

	// Prices estimates turns a CLI left unpriced (ADR-0185). Nil prices
	// nothing, and every such turn stays unpriced.
	Prices *pricing.Table
}

// LocOf is the request's location, defaulting to the process's, which is the
// same default every meter applies.
func (r Request) LocOf() *time.Location {
	if r.Loc == nil {
		return time.Local
	}
	return r.Loc
}

// SeriesKey formats one instant as a series key: a calendar day, or an hour
// of one when the request asked for hourly buckets. The key states its own
// granularity ("2026-09-13" versus "2026-09-13T14"), so a consumer labels it
// without a second flag and an older client degrades to the raw key.
func (r Request) SeriesKey(t time.Time) string {
	if r.Hourly {
		return t.In(r.LocOf()).Format("2006-01-02T15")
	}
	return t.In(r.LocOf()).Format("2006-01-02")
}

// BillingFor is what the operator recorded for a CLI, defaulting to unknown
// rather than guessing. A guess here silently changes what the headline
// number means.
func (r Request) BillingFor(cli string) Billing {
	if b, ok := r.Billing[cli]; ok && b != "" {
		return b
	}
	return BillingUnknown
}

// Meter aggregates one CLI's own session files over a window. Every Meter
// answers Coverage even when it found nothing, so the dashboard can say
// what it did not measure.
//
// Fingerprint is a stat-only change detector over that CLI's own store. It
// must never open a file: the dashboard polls on a timer, and the whole
// point is that an unchanged tree costs a directory sweep rather than a
// re-parse. An empty string means "cannot tell", which the caller must read
// as "always changed" rather than "never changed".
type Meter interface {
	CLI() string
	Label() string
	Fingerprint() string
	Meter(req Request) (Window, error)
}

// Meters is the registry the server aggregates over. Adding a CLI here is
// the whole wiring: coverage, the by-CLI pivot and every breakdown pick it
// up from the Window it returns.
func Meters() []Meter {
	return []Meter{
		PiMeter{}, ClaudeCodeMeter{}, CodexMeter{},
		OpenCodeMeter{}, HermesMeter{}, GrokMeter{},
		OmpMeter{}, MuseMeter{}, AgyMeter{},
	}
}

// Fingerprint joins every meter's own change detector. An empty component
// forces a miss, because a meter that cannot describe its state must not be
// served from a cache that assumes it can.
func Fingerprint(meters []Meter) string {
	parts := make([]string, 0, len(meters))
	for _, m := range meters {
		fp := m.Fingerprint()
		if fp == "" {
			return ""
		}
		parts = append(parts, m.CLI()+"="+fp)
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

// FleetStats is the merged answer: session.WindowStats as the dashboard
// already knows it, now summed across every CLI, plus the families that
// only exist because CLI agents record more than pi does.
type FleetStats struct {
	session.WindowStats
	ByCLI    []CLIBucket   `json:"byCli"`
	Coverage []CoverageRow `json:"coverage"`
	Impact   *Impact       `json:"impact,omitempty"`
	Timing   *Timing       `json:"timing,omitempty"`
	Limits   []LimitWindow `json:"limits,omitempty"`
}

// Aggregate runs every registered Meter and merges the results. A Meter
// that fails degrades to a coverage row explaining itself — one unreadable
// CLI must not take the dashboard down with it.
//
// Meters run concurrently: each one parses its own store (JSONL, SQLite)
// and the cold 7d window was 8.2s sequential (pi 4.5s + codex 2.7s +
// claude-code 1.25s on this machine). The shared parseCache is mutex-guarded,
// each cliAcc is per-meter, and merge is single-threaded over the finished
// windows — so the wall time drops to the slowest meter, not the sum.
func Aggregate(req Request, meters []Meter) FleetStats {
	windows := make([]Window, len(meters))
	var wg sync.WaitGroup
	for i, m := range meters {
		wg.Add(1)
		go func(i int, m Meter) {
			defer wg.Done()
			w, err := m.Meter(req)
			if err != nil {
				windows[i] = Window{
					CLI:      m.CLI(),
					Coverage: unavailableRow(m, req.BillingFor(m.CLI()), err.Error()),
				}
				return
			}
			w.CLI = m.CLI()
			windows[i] = w
		}(i, m)
	}
	wg.Wait()
	return merge(req, windows)
}

func unavailableRow(m Meter, b Billing, note string) CoverageRow {
	sig := map[Signal]State{}
	for _, s := range Signals {
		sig[s] = StateUnavailable
	}
	return CoverageRow{CLI: m.CLI(), Label: m.Label(), Billing: b, Signals: sig, Note: note}
}

// merge sums every window and ranks once over the union.
func merge(req Request, windows []Window) FleetStats {
	out := FleetStats{}
	out.From = req.From.Format(time.RFC3339)
	out.To = req.To.Format(time.RFC3339)

	var prior session.PeriodTotals
	byProvider := map[string]*session.ProviderBucket{}
	byModel := map[[3]string]*session.ModelBucket{}
	byWorkspace := map[string]*session.WorkspaceBucket{}
	byDay := map[string]*session.DayBucket{}
	byTool := map[[2]string]*session.ToolBucket{}
	var sessions []session.SessionSpend
	var impact Impact
	var timing Timing
	haveImpact, haveTiming := false, false

	for _, w := range windows {
		st := w.Stats
		out.Current.Cost += st.Current.Cost
		out.Current.Estimated += st.Current.Estimated
		out.Current.Messages += st.Current.Messages
		out.Current.Sessions += st.Current.Sessions
		if st.Prior != nil {
			prior.Cost += st.Prior.Cost
			prior.Estimated += st.Prior.Estimated
			prior.Messages += st.Prior.Messages
			prior.Sessions += st.Prior.Sessions
		}

		out.Tokens.Input += st.Tokens.Input
		out.Tokens.Output += st.Tokens.Output
		out.Tokens.CacheRead += st.Tokens.CacheRead
		out.Tokens.CacheWrite += st.Tokens.CacheWrite
		out.Tokens.Reasoning += st.Tokens.Reasoning
		out.CostSplit.Add(st.CostSplit)

		out.Turns.Assistant += st.Turns.Assistant
		out.Turns.User += st.Turns.User
		out.Turns.Errors += st.Turns.Errors
		out.Turns.Aborted += st.Turns.Aborted
		out.Turns.Refusals += st.Turns.Refusals
		out.Turns.Compactions += st.Turns.Compactions

		for _, p := range st.ByProvider {
			b := byProvider[p.Provider]
			if b == nil {
				b = &session.ProviderBucket{Provider: p.Provider, Billing: p.Billing}
				byProvider[p.Provider] = b
			}
			b.Cost += p.Cost
			b.Messages += p.Messages
		}
		for _, m := range st.ByModel {
			// Keyed by CLI too: the same model reached through two CLIs is
			// two rows, because that is the comparison the surface exists
			// to make.
			k := [3]string{w.CLI, m.Provider, m.Model}
			b := byModel[k]
			if b == nil {
				b = &session.ModelBucket{Provider: m.Provider, Model: m.Model, CLI: w.CLI}
				byModel[k] = b
			}
			b.Cost += m.Cost
			b.Estimated += m.Estimated
			b.Messages += m.Messages
		}
		for _, ws := range st.ByWorkspace {
			b := byWorkspace[ws.Cwd]
			if b == nil {
				b = &session.WorkspaceBucket{Cwd: ws.Cwd}
				byWorkspace[ws.Cwd] = b
			}
			b.Cost += ws.Cost
			b.Messages += ws.Messages
			b.Sessions += ws.Sessions
		}
		for _, d := range st.Series {
			b := byDay[d.Date]
			if b == nil {
				b = &session.DayBucket{Date: d.Date}
				byDay[d.Date] = b
			}
			b.Cost += d.Cost
			b.Messages += d.Messages
			b.Turns += d.Turns
		}
		for _, t := range st.Tools {
			// Tool names are deliberately not normalized across CLIs:
			// "Bash", "bash" and "shell" are three vendors' words, and
			// inventing an equivalence would be a silent claim we cannot
			// back. The CLI mark disambiguates them on the surface.
			k := [2]string{w.CLI, t.Name}
			b := byTool[k]
			if b == nil {
				b = &session.ToolBucket{Name: t.Name, CLI: w.CLI}
				byTool[k] = b
			}
			b.Calls += t.Calls
		}
		for _, s := range st.TopSessions {
			s.CLI = w.CLI
			sessions = append(sessions, s)
		}

		if w.Impact != nil {
			impact.Add(*w.Impact)
			haveImpact = true
		}
		if w.Timing != nil {
			timing.Add(*w.Timing)
			haveTiming = true
		}
		out.Limits = append(out.Limits, w.Limits...)
		if w.Coverage.CLI != "" {
			out.Coverage = append(out.Coverage, w.Coverage)
			out.ByCLI = append(out.ByCLI, cliBucket(w))
		}
	}

	if !req.PriorFrom.IsZero() {
		out.Prior = &prior
	}
	if prompt := out.Tokens.Input + out.Tokens.CacheRead; prompt > 0 {
		h := 100 * float64(out.Tokens.CacheRead) / float64(prompt)
		out.Tokens.CacheHit = &h
	}
	if haveImpact {
		out.Impact = &impact
	}
	if haveTiming {
		out.Timing = &timing
	}

	out.ByProvider = rankProviders(byProvider)
	out.ByModel = rankModels(byModel)
	out.ByWorkspace = rankWorkspaces(byWorkspace)
	out.Series = fillSeries(byDay, req)
	out.Tools = rankTools(byTool)
	out.TopSessions = rankSessions(sessions)
	sortCLIBuckets(out.ByCLI)
	sortCoverage(out.Coverage)
	sortLimits(out.Limits)
	return out
}

func cliBucket(w Window) CLIBucket {
	st := w.Stats
	b := CLIBucket{
		CLI:       w.Coverage.CLI,
		Label:     w.Coverage.Label,
		Billing:   w.Coverage.Billing,
		Cost:      st.Current.Cost,
		Estimated: st.Current.Estimated,
		CostState: w.Coverage.Signals[SigCost],
		Messages:  st.Current.Messages,
		Sessions:  st.Current.Sessions,
		Tokens: st.Tokens.Input + st.Tokens.Output +
			st.Tokens.CacheRead + st.Tokens.CacheWrite,
	}
	return b
}

func rankProviders(m map[string]*session.ProviderBucket) []session.ProviderBucket {
	out := make([]session.ProviderBucket, 0, len(m))
	for _, b := range m {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cost != out[j].Cost {
			return out[i].Cost > out[j].Cost
		}
		return out[i].Provider < out[j].Provider
	})
	return out
}

func rankModels(m map[[3]string]*session.ModelBucket) []session.ModelBucket {
	out := make([]session.ModelBucket, 0, len(m))
	for _, b := range m {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cost != out[j].Cost {
			return out[i].Cost > out[j].Cost
		}
		if out[i].Model != out[j].Model {
			return out[i].Model < out[j].Model
		}
		return out[i].CLI < out[j].CLI
	})
	return out
}

func rankWorkspaces(m map[string]*session.WorkspaceBucket) []session.WorkspaceBucket {
	out := make([]session.WorkspaceBucket, 0, len(m))
	for _, b := range m {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cost != out[j].Cost {
			return out[i].Cost > out[j].Cost
		}
		return out[i].Cwd < out[j].Cwd
	})
	return out
}

func rankTools(m map[[2]string]*session.ToolBucket) []session.ToolBucket {
	out := make([]session.ToolBucket, 0, len(m))
	for _, b := range m {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Calls != out[j].Calls {
			return out[i].Calls > out[j].Calls
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].CLI < out[j].CLI
	})
	if len(out) > session.TopTools {
		out = out[:session.TopTools]
	}
	return out
}

func rankSessions(in []session.SessionSpend) []session.SessionSpend {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Cost != in[j].Cost {
			return in[i].Cost > in[j].Cost
		}
		return in[i].Path < in[j].Path
	})
	if len(in) > session.TopSessionsN {
		in = in[:session.TopSessionsN]
	}
	return in
}

// fillSeries zero-fills every bucket in the window so the bar chart draws
// gaps as gaps. range=all has no fixed start, so it renders only the days
// that carry data, ascending.
func fillSeries(m map[string]*session.DayBucket, req Request) []session.DayBucket {
	if req.From.IsZero() {
		out := make([]session.DayBucket, 0, len(m))
		for _, b := range m {
			out = append(out, *b)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
		return out
	}
	loc := req.LocOf()
	var out []session.DayBucket
	if req.Hourly {
		// Step by real hours, not by position in the day: a DST day is 23 or
		// 25 hours long, and a fall-back day repeats one clock hour by name.
		// That hour is one bucket, so the chart never draws one label twice.
		seen := map[string]bool{}
		for h := req.From.In(loc); h.Before(req.To); h = h.Add(time.Hour) {
			key := h.Format("2006-01-02T15")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, bucketAt(m, key))
		}
		return out
	}
	for d := req.From.In(loc); d.Before(req.To); d = d.AddDate(0, 0, 1) {
		out = append(out, bucketAt(m, d.Format("2006-01-02")))
	}
	return out
}

// bucketAt reads one series bucket, or an empty one: a day (or hour) with no
// data is a zero bar, never a missing one.
func bucketAt(m map[string]*session.DayBucket, key string) session.DayBucket {
	if b := m[key]; b != nil {
		return *b
	}
	return session.DayBucket{Date: key}
}

func sortCLIBuckets(in []CLIBucket) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Cost != in[j].Cost {
			return in[i].Cost > in[j].Cost
		}
		if in[i].Messages != in[j].Messages {
			return in[i].Messages > in[j].Messages
		}
		return in[i].CLI < in[j].CLI
	})
}

func sortCoverage(in []CoverageRow) {
	sort.Slice(in, func(i, j int) bool { return in[i].CLI < in[j].CLI })
}

func sortLimits(in []LimitWindow) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].CLI != in[j].CLI {
			return in[i].CLI < in[j].CLI
		}
		return in[i].WindowMin < in[j].WindowMin
	})
}

// windowLabel names a quota window by its length, which is how vendors talk
// about them ("your weekly limit"), rather than by raw minutes.
func windowLabel(minutes int) string {
	switch {
	case minutes <= 0:
		return ""
	case minutes%(60*24*7) == 0:
		return "weekly"
	case minutes%(60*24) == 0:
		return "daily"
	case minutes%60 == 0:
		return itoa(minutes/60) + "h"
	default:
		return itoa(minutes) + "m"
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// moneyText is a dollar amount for a coverage note.
func moneyText(v float64) string { return "$" + strconv.FormatFloat(v, 'f', 2, 64) }
