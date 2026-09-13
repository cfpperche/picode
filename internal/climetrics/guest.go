package climetrics

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

// guestEntry is one message, normalised out of whatever dialect its CLI
// wrote it in. Every guest adapter's job is to turn its own store into a
// stream of these; the windowing, bucketing and ranking below are then
// identical for all of them, which is what keeps a sixth CLI cheap to add.
//
// Text is absent by construction. An adapter reads a transcript for counts,
// names and identities — never for content — so no field here can carry a
// message body into the payload.
type guestEntry struct {
	at    time.Time
	key   string // session identity: a file path or a row id
	cwd   string
	name  string // the session's own title, if it has one
	role  string // "assistant" counts as a turn; anything else is a prompt
	model string
	prov  string
	cost  float64
	split session.CostSplit
	toks  session.TokenTotals
	tools []string
	// results is how many tool results this entry carried and the parser
	// inspected, errs how many of those were failures. Both are kept for
	// every role: Claude Code returns a tool's result as a *user* turn, and
	// counting errors only on assistant turns reported 0 against 456 on the
	// deployed dashboard.
	results int
	errs    int
	abort   bool
	refusal bool
}

// guestAcc files entries into one CLI's window.
type guestAcc struct {
	req     Request
	cli     string
	loc     *time.Location
	current session.PeriodTotals
	prior   session.PeriodTotals

	byModel  map[[2]string]*session.ModelBucket
	byFolder map[string]*session.WorkspaceBucket
	byDay    map[string]*session.DayBucket
	tools    map[string]int
	toks     session.TokenTotals
	split    session.CostSplit
	turns    session.TurnStats

	files      map[string]*session.SessionSpend
	priorFiles map[string]bool

	// impact and timing are filled by replay, which prorates each file's
	// lifetime figures against this window's share of its tokens.
	impact Impact
	timing Timing

	// seen is what the parser actually observed in this window, per signal.
	// Coverage is derived from it rather than declared: a hand-written
	// "reported" beside a counter nothing ever incremented is how the
	// dashboard shipped a silent zero for Claude Code's errors.
	seen map[Signal]int
}

func newGuestAcc(req Request, cli string) *guestAcc {
	loc := req.Loc
	if loc == nil {
		loc = time.Local
	}
	return &guestAcc{
		req: req, cli: cli, loc: loc,
		byModel:    map[[2]string]*session.ModelBucket{},
		byFolder:   map[string]*session.WorkspaceBucket{},
		byDay:      map[string]*session.DayBucket{},
		tools:      map[string]int{},
		files:      map[string]*session.SessionSpend{},
		priorFiles: map[string]bool{},
		seen:       map[Signal]int{},
	}
}

// add files one entry into whichever window it falls in. Only the current
// window gets breakdowns; the prior one exists solely for the headline
// delta, exactly as pi's own scan does it.
func (a *guestAcc) add(e guestEntry) {
	if e.at.IsZero() {
		return
	}
	switch {
	case !a.req.PriorFrom.IsZero() && e.at.Before(a.req.PriorFrom):
		return
	case !e.at.Before(a.req.To):
		return
	case !a.req.From.IsZero() && e.at.Before(a.req.From):
		a.prior.Cost += e.cost
		a.prior.Messages++
		a.priorFiles[e.key] = true
		return
	}

	a.current.Cost += e.cost
	a.current.Messages++
	a.split.Add(e.split)
	a.seen[SigMessages]++
	if e.cost > 0 {
		a.seen[SigCost]++
	}
	// Tool results ride on whichever role the CLI returns them under, so
	// their evidence and their failures are counted here, before the split.
	a.seen[SigErrors] += e.results
	a.turns.Errors += e.errs

	day := e.at.In(a.loc).Format("2006-01-02")
	db := a.byDay[day]
	if db == nil {
		db = &session.DayBucket{Date: day}
		a.byDay[day] = db
	}
	db.Cost += e.cost
	db.Messages++

	wb := a.byFolder[e.cwd]
	if wb == nil {
		wb = &session.WorkspaceBucket{Cwd: e.cwd}
		a.byFolder[e.cwd] = wb
	}
	wb.Cost += e.cost
	wb.Messages++

	fa := a.files[e.key]
	if fa == nil {
		fa = &session.SessionSpend{Path: e.key, CLI: a.cli, Cwd: e.cwd}
		a.files[e.key] = fa
		wb.Sessions++
	}
	if e.name != "" {
		fa.Name = e.name
	}
	fa.Cost += e.cost
	fa.Messages++
	if last, err := time.Parse(time.RFC3339, fa.LastAt); err != nil || e.at.After(last) {
		fa.LastAt = e.at.In(a.loc).Format(time.RFC3339)
	}

	if e.role != "assistant" {
		a.turns.User++
		return
	}
	a.turns.Assistant++
	db.Turns++
	a.seen[SigTurns]++
	if e.abort {
		a.turns.Aborted++
	}
	if e.refusal {
		a.turns.Refusals++
	}
	if e.model != "" {
		a.seen[SigModel]++
	}
	if e.toks != (session.TokenTotals{}) {
		a.seen[SigTokens]++
	}
	if len(e.tools) > 0 {
		a.seen[SigTools]++
	}

	model := e.model
	if model == "" {
		model = "unknown"
	}
	prov := e.prov
	if prov == "" {
		prov = "unknown"
	}
	mk := [2]string{prov, model}
	mb := a.byModel[mk]
	if mb == nil {
		mb = &session.ModelBucket{Provider: prov, Model: model, CLI: a.cli}
		a.byModel[mk] = mb
	}
	mb.Cost += e.cost
	mb.Messages++

	a.toks.Input += e.toks.Input
	a.toks.Output += e.toks.Output
	a.toks.CacheRead += e.toks.CacheRead
	a.toks.CacheWrite += e.toks.CacheWrite
	a.toks.Reasoning += e.toks.Reasoning
	for _, t := range e.tools {
		a.tools[t]++
	}
}

// result is the uncapped window. Aggregate ranks and cuts once over the
// union of every CLI, so nothing is sorted or truncated here.
func (a *guestAcc) result() session.WindowStats {
	st := session.WindowStats{
		Current:   a.current,
		Tokens:    a.toks,
		CostSplit: a.split,
		Turns:     a.turns,
	}
	st.Current.Sessions = len(a.files)
	if !a.req.PriorFrom.IsZero() {
		p := a.prior
		p.Sessions = len(a.priorFiles)
		st.Prior = &p
	}
	for _, m := range a.byModel {
		st.ByModel = append(st.ByModel, *m)
	}
	for _, w := range a.byFolder {
		st.ByWorkspace = append(st.ByWorkspace, *w)
	}
	for _, d := range a.byDay {
		st.Series = append(st.Series, *d)
	}
	for n, c := range a.tools {
		st.Tools = append(st.Tools, session.ToolBucket{Name: n, CLI: a.cli, Calls: c})
	}
	for _, f := range a.files {
		st.TopSessions = append(st.TopSessions, *f)
	}
	// No ByProvider rows: that breakdown answers "what did the credential
	// PiCode holds cost", and a guest CLI signs in with its own account.
	return st
}

// evidence turns what the accumulator observed into signal states.
//
// can says which signals this CLI is capable of recording at all; a signal
// it cannot record is not-reported regardless of the window. For the rest,
// an active window answers from evidence — reported only when the parser
// actually saw the field — and an empty window answers from capability,
// since there is nothing to have seen. extra carries the adapter-level
// signals (impact, timing, limits) the accumulator never sees.
func (a *guestAcc) evidence(can map[Signal]bool, extra map[Signal]bool) map[Signal]State {
	out := make(map[Signal]State, len(Signals))
	active := a.current.Messages > 0
	for _, sig := range Signals {
		switch {
		case !can[sig]:
			out[sig] = StateNotReported
		case !active:
			out[sig] = StateReported
		case sig == SigImpact || sig == SigTiming || sig == SigLimits:
			if extra[sig] {
				out[sig] = StateReported
			} else {
				out[sig] = StateNotReported
			}
		case a.seen[sig] > 0:
			out[sig] = StateReported
		default:
			out[sig] = StateNotReported
		}
	}
	return out
}

// absentWindow is what a meter returns when its CLI was never installed
// here. It is a coverage row and nothing else — the surface still names the
// CLI and says why it is empty, rather than dropping it silently.
func absentWindow(m Meter, req Request) Window {
	sig := map[Signal]State{}
	for _, s := range Signals {
		sig[s] = StateUnavailable
	}
	return Window{
		CLI: m.CLI(),
		Coverage: CoverageRow{
			CLI: m.CLI(), Label: m.Label(), Billing: req.BillingFor(m.CLI()),
			Signals: sig,
			Note:    "Not installed here — no session store on this machine.",
		},
	}
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

// scanStrings reads one row into a name→text map. Vendor schemas drift and
// their column types are not ours to assume, so every value is taken as
// text and converted by the caller: a column that changes from INTEGER to
// REAL must not turn a whole CLI into an error.
func scanStrings(rows *sql.Rows, sel []string) (map[string]string, bool) {
	holders := make([]sql.RawBytes, len(sel))
	ptrs := make([]any, len(sel))
	for i := range holders {
		ptrs[i] = &holders[i]
	}
	if rows.Scan(ptrs...) != nil {
		return nil, false
	}
	out := make(map[string]string, len(sel))
	for i, name := range sel {
		out[name] = string(holders[i])
	}
	return out, true
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
