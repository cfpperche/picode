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

// ClaudeCodeMeter reports Claude Code sessions from the transcripts under
// ~/.claude/projects, the same tree clisession.ClaudeCodeSource lists.
//
// Cost needs care. Claude Code writes no price on a message: the only money
// on disk is the "cost-state" record, which is a *cumulative* snapshot of
// the whole session (totalCostUSD plus per-model costUSD and token counts).
// Two facts follow, both measured on this machine on 2026-09-07:
//
//   - Taking the last snapshot as the window's cost would charge a
//     month-old session's whole bill to the day it was last touched — the
//     fabricated-spike failure ADR-0041 refused for exactly this reason.
//   - Only 46 of 340 transcripts carry a cost-state at all, and none of the
//     six most recently written ones did. A live session has no snapshot.
//
// So this adapter takes the per-model cost from the session's own final
// snapshot — the vendor's own arithmetic, never a price list PiCode would
// have to maintain — and spreads it across that model's messages in
// proportion to their tokens. A window covering the whole session reports
// exactly what Claude Code said it cost; a shorter window reports its share.
// The rate divides by the tokens actually observed in the transcript rather
// than by the snapshot's own totals, because the two disagree (sidechains
// and compacted-away turns are billed but no longer on disk) and dividing
// by the larger number would quietly under-report every session.
// Sessions with no snapshot contribute tokens and activity but no cost, and
// say so: coverage reports cost as partial with both counts, never as a zero
// that would read as "Claude Code was free".
type ClaudeCodeMeter struct{}

func (ClaudeCodeMeter) CLI() string   { return "claude-code" }
func (ClaudeCodeMeter) Label() string { return "Claude Code" }

// Fingerprint reuses pi's sweep: ~/.claude/projects nests transcripts one
// directory deep, the same shape session.Fingerprint already walks.
func (ClaudeCodeMeter) Fingerprint() string {
	return session.Fingerprint(clisession.ClaudeProjectsRoot())
}

// ccUnits is the denominator the per-model rate is expressed in. Any
// consistent definition makes a session's total reconcile exactly, because
// the rate is that session's own cost divided by that session's own units.
// Raw token count is the one definition that invents nothing: weighting
// output above cache reads would be truer to how vendors bill and would be
// a number PiCode made up. The consequence is bounded and worth naming: the
// daily *shape* of a session's spend is approximate, its total is exact.
type ccUnits = int64

type ccMsg struct {
	at     time.Time
	role   string
	model  string
	units  ccUnits
	tokens session.TokenTotals
	stop   string
	errs   int
	tools  []string
}

type ccState struct {
	modelCost map[string]float64 // model -> USD for the whole session
	linesAdd  int64
	linesDel  int64
	apiMs     int64
	toolMs    int64
	wallMs    int64
	hasCost   bool
}

func (m ClaudeCodeMeter) Meter(req Request) (Window, error) {
	root := clisession.ClaudeProjectsRoot()
	acc := newCCAcc(req)
	priced, unpriced := 0, 0

	for _, dir := range ccProjectDirs(root) {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			// A file untouched since before the widened window cannot hold
			// an in-window message; the same cheap pre-filter pi's scan uses.
			if !req.PriorFrom.IsZero() && info.ModTime().Before(req.PriorFrom) {
				continue
			}
			ok, hadCost := acc.scanFile(filepath.Join(dir, e.Name()))
			if !ok {
				continue
			}
			if hadCost {
				priced++
			} else {
				unpriced++
			}
		}
	}

	st := acc.result()
	billing := req.BillingFor(m.CLI())
	for i := range st.ByProvider {
		st.ByProvider[i].Billing = string(billing)
	}
	w := Window{
		CLI:      m.CLI(),
		Stats:    st,
		Coverage: ccCoverage(m, billing, priced, unpriced),
	}
	if acc.impact.LinesAdded != 0 || acc.impact.LinesRemoved != 0 {
		i := acc.impact
		w.Impact = &i
	}
	if acc.timing != (Timing{}) {
		t := acc.timing
		w.Timing = &t
	}
	return w, nil
}

func ccCoverage(m ClaudeCodeMeter, b Billing, priced, unpriced int) CoverageRow {
	cost := StateReported
	note := ""
	switch {
	case priced == 0 && unpriced == 0:
		cost = StateReported // nothing in window; nothing to qualify
	case priced == 0:
		cost = StateNotReported
		note = "No session in this window recorded a cost snapshot; tokens and activity are complete."
	case unpriced > 0:
		cost = StatePartial
		note = "Priced from " + itoa(priced) + " of " + itoa(priced+unpriced) +
			" sessions — Claude Code writes cost only on a session snapshot, and a live session has none."
	}
	return CoverageRow{
		CLI:     m.CLI(),
		Label:   m.Label(),
		Billing: b,
		Signals: map[Signal]State{
			SigCost:     cost,
			SigTokens:   StateReported,
			SigModel:    StateReported,
			SigMessages: StateReported,
			SigTurns:    StateReported,
			SigTools:    StateReported,
			SigErrors:   StateReported,
			SigImpact:   StateReported,
			SigTiming:   StateReported,
			SigLimits:   StateNotReported,
		},
		Note: note,
	}
}

func ccProjectDirs(root string) []string {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, filepath.Join(root, e.Name()))
		}
	}
	return out
}

// ccAcc accumulates across files. It mirrors what session.statsAcc holds for
// pi, so merge() can treat both the same way.
type ccAcc struct {
	req        Request
	current    session.PeriodTotals
	prior      session.PeriodTotals
	byModel    map[string]*session.ModelBucket
	byFolder   map[string]*session.WorkspaceBucket
	byDay      map[string]*session.DayBucket
	tools      map[string]int
	tokens     session.TokenTotals
	turns      session.TurnStats
	sessions   []session.SessionSpend
	impact     Impact
	timing     Timing
	priorPaths map[string]bool
	pathSeen   map[string]bool
	loc        *time.Location
}

func newCCAcc(req Request) *ccAcc {
	loc := req.Loc
	if loc == nil {
		loc = time.Local
	}
	return &ccAcc{
		req:        req,
		byModel:    map[string]*session.ModelBucket{},
		byFolder:   map[string]*session.WorkspaceBucket{},
		byDay:      map[string]*session.DayBucket{},
		tools:      map[string]int{},
		priorPaths: map[string]bool{},
		pathSeen:   map[string]bool{},
		loc:        loc,
	}
}

// scanFile reads one transcript. It returns whether the file contributed to
// the current window at all, and whether it carried a cost snapshot.
func (a *ccAcc) scanFile(path string) (contributed, hadCost bool) {
	f, err := os.Open(path)
	if err != nil {
		return false, false
	}
	defer f.Close()

	var msgs []ccMsg
	var st ccState
	cwd := ""
	name := ""

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		if c, _ := raw["cwd"].(string); c != "" && cwd == "" {
			cwd = c
			// Claude Code names the cwd on ordinary entries rather than a
			// header, so the verdict cannot be taken before the first one.
			if a.req.Scope == ScopePiCode && !a.req.InScope(cwd) {
				return false, false
			}
		}
		switch raw["type"] {
		case "summary":
			if s, _ := raw["summary"].(string); s != "" && name == "" {
				name = s
			}
		case "cost-state":
			readCostState(raw, &st)
		case "user", "assistant":
			if m, ok := ccMessage(raw); ok {
				msgs = append(msgs, m)
			}
		}
	}
	if len(msgs) == 0 {
		return false, st.hasCost
	}
	if cwd == "" {
		cwd = filepath.Base(filepath.Dir(path))
		if a.req.Scope == ScopePiCode && !a.req.InScope(cwd) {
			return false, st.hasCost
		}
	}
	return a.fold(path, cwd, name, msgs, st), st.hasCost
}

// fold prices the file's messages and files them into the window.
func (a *ccAcc) fold(path, cwd, name string, msgs []ccMsg, st ccState) bool {
	var inWindow, total int64
	observed := map[string]int64{}
	for _, m := range msgs {
		total += m.units
		observed[ccModelKey(m.model)] += m.units
		if a.inCurrent(m.at) {
			inWindow += m.units
		}
	}
	rates, flat := ccRates(st.modelCost, observed, total)

	var fileCost float64
	var fileMsgs int
	var last time.Time
	contributed := false

	for _, m := range msgs {
		cost := flat * float64(m.units)
		if r, ok := rates[ccModelKey(m.model)]; ok {
			cost += r * float64(m.units)
		}
		switch {
		case !a.req.PriorFrom.IsZero() && m.at.Before(a.req.PriorFrom):
			continue
		case !m.at.Before(a.req.To):
			continue
		case !a.req.From.IsZero() && m.at.Before(a.req.From):
			a.prior.Cost += cost
			a.prior.Messages++
			a.priorPaths[path] = true
			continue
		}

		contributed = true
		a.current.Cost += cost
		a.current.Messages++
		fileCost += cost
		fileMsgs++
		if m.at.After(last) {
			last = m.at
		}

		day := m.at.In(a.loc).Format("2006-01-02")
		db := a.byDay[day]
		if db == nil {
			db = &session.DayBucket{Date: day}
			a.byDay[day] = db
		}
		db.Cost += cost
		db.Messages++

		wb := a.byFolder[cwd]
		if wb == nil {
			wb = &session.WorkspaceBucket{Cwd: cwd}
			a.byFolder[cwd] = wb
		}
		wb.Cost += cost
		wb.Messages++

		if m.role != "assistant" {
			a.turns.User++
			continue
		}
		a.turns.Assistant++
		db.Turns++
		a.turns.Errors += m.errs
		if m.stop == "aborted" || m.stop == "stop_sequence" {
			a.turns.Aborted++
		}
		model := m.model
		if model == "" {
			model = "unknown"
		}
		mb := a.byModel[model]
		if mb == nil {
			mb = &session.ModelBucket{Provider: ccProvider(model), Model: model, CLI: "claude-code"}
			a.byModel[model] = mb
		}
		mb.Cost += cost
		mb.Messages++
		a.tokens.Input += m.tokens.Input
		a.tokens.Output += m.tokens.Output
		a.tokens.CacheRead += m.tokens.CacheRead
		a.tokens.CacheWrite += m.tokens.CacheWrite
		a.tokens.Reasoning += m.tokens.Reasoning
		for _, t := range m.tools {
			a.tools[t]++
		}
	}

	if !contributed {
		return false
	}
	if !a.pathSeen[path] {
		a.pathSeen[path] = true
		a.byFolder[cwd].Sessions++
		a.current.Sessions++
	}
	a.sessions = append(a.sessions, session.SessionSpend{
		Path: path, CLI: "claude-code", Name: name, Cwd: cwd,
		Cost: fileCost, Messages: fileMsgs,
		LastAt: last.In(a.loc).Format(time.RFC3339),
	})

	// Lines changed and durations are session-lifetime totals on the
	// snapshot. Prorate them by the share of the session's tokens that fall
	// inside the window — the same proportional rule the cost rate uses, and
	// the only one that does not charge a whole session's edits to whichever
	// day the window happens to end on.
	if total > 0 && inWindow > 0 {
		share := float64(inWindow) / float64(total)
		a.impact.LinesAdded += scale(st.linesAdd, share)
		a.impact.LinesRemoved += scale(st.linesDel, share)
		a.timing.APIMs += scale(st.apiMs, share)
		a.timing.ToolMs += scale(st.toolMs, share)
		a.timing.WallMs += scale(st.wallMs, share)
	}
	return true
}

// ccRates turns a session's per-model snapshot costs into per-token rates.
//
// Two mismatches between the snapshot and the transcript make the naive
// division wrong, both measured on real files (2026-09-07):
//
//   - The snapshot names models by their billing id ("claude-opus-5[1m]")
//     while messages name them plainly ("claude-opus-5"). Matching on the
//     raw string silently dropped $540 of a $1,001 session — the largest
//     model in it — so the key is normalised on both sides.
//   - A model can be billed with no message of its own left on disk (a
//     subagent's haiku turns live in another transcript). Its cost cannot be
//     placed on any one model, so rather than lose it, it is spread flat
//     across the session's remaining tokens: the model breakdown stays
//     approximate for that slice, the session total stays exact.
//
// Dividing by observed rather than snapshot tokens also absorbs the 1.6x
// gap between them, whatever its cause, because each rate is that model's
// own cost over its own surviving tokens.
func ccRates(modelCost map[string]float64, observed map[string]int64, total int64) (rates map[string]float64, flat float64) {
	rates = make(map[string]float64, len(modelCost))
	var unplaced float64
	for model, cost := range modelCost {
		key := ccModelKey(model)
		if u := observed[key]; u > 0 {
			rates[key] += cost / float64(u)
			continue
		}
		unplaced += cost
	}
	if unplaced > 0 && total > 0 {
		flat = unplaced / float64(total)
	}
	return rates, flat
}

// ccModelKey strips the bracketed billing variant a snapshot appends
// ("claude-opus-5[1m]" -> "claude-opus-5") so both sides of the match speak
// the same name.
func ccModelKey(model string) string {
	if i := strings.IndexByte(model, '['); i > 0 {
		return model[:i]
	}
	return model
}

func scale(v int64, share float64) int64 { return int64(float64(v) * share) }

func (a *ccAcc) inCurrent(t time.Time) bool {
	if !t.Before(a.req.To) {
		return false
	}
	return a.req.From.IsZero() || !t.Before(a.req.From)
}

func (a *ccAcc) result() session.WindowStats {
	st := session.WindowStats{Current: a.current, Tokens: a.tokens, Turns: a.turns}
	st.Current.Sessions = len(a.pathSeen)
	if !a.req.PriorFrom.IsZero() {
		p := a.prior
		p.Sessions = len(a.priorPaths)
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
		st.Tools = append(st.Tools, session.ToolBucket{Name: n, CLI: "claude-code", Calls: c})
	}
	st.TopSessions = a.sessions
	// No ByProvider rows, deliberately. That breakdown feeds the Providers
	// view, which answers "what did the credential I hold cost" by joining
	// provider ids onto PiCode's own credential roster. Claude Code
	// authenticates itself against an account PiCode does not hold, so
	// adding an "anthropic" row here would bill a subscription's list-price
	// equivalent to an API key that never paid it. The cross-CLI spend
	// story lives in ByCLI and ByModel, where the CLI is named.
	return st
}

// ccProvider is the vendor behind a Claude Code model id. Claude Code only
// ever talks to Anthropic, so this is a constant rather than a lookup — but
// it is written as a function so a future Bedrock/Vertex id has one place
// to land.
func ccProvider(model string) string {
	switch {
	case strings.HasPrefix(model, "<"): // "<synthetic>" — Claude Code's own placeholder
		return "unknown"
	default:
		return "anthropic"
	}
}

// readCostState turns one cumulative snapshot into per-model unit rates.
func readCostState(raw map[string]any, st *ccState) {
	st.hasCost = true
	st.linesAdd = int64(num(raw["totalLinesAdded"]))
	st.linesDel = int64(num(raw["totalLinesRemoved"]))
	st.apiMs = int64(num(raw["totalAPIDuration"]))
	st.toolMs = int64(num(raw["totalToolDuration"]))
	st.wallMs = int64(num(raw["totalDuration"]))

	mu, _ := raw["modelUsage"].(map[string]any)
	if mu == nil {
		return
	}
	costs := make(map[string]float64, len(mu))
	for model, v := range mu {
		u, _ := v.(map[string]any)
		if u == nil {
			continue
		}
		if c := num(u["costUSD"]); c > 0 {
			costs[model] = c
		}
	}
	if len(costs) > 0 {
		st.modelCost = costs
	}
}

// ccMessage projects one transcript entry. Content is read for tool names
// and error flags only — never text, which never leaves this package.
func ccMessage(raw map[string]any) (ccMsg, bool) {
	m, _ := raw["message"].(map[string]any)
	if m == nil {
		return ccMsg{}, false
	}
	ts, _ := raw["timestamp"].(string)
	at, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ccMsg{}, false
	}
	out := ccMsg{at: at}
	out.role, _ = m["role"].(string)
	out.model, _ = m["model"].(string)
	out.stop, _ = m["stop_reason"].(string)

	if u, _ := m["usage"].(map[string]any); u != nil {
		out.tokens = session.TokenTotals{
			Input:      int64(num(u["input_tokens"])),
			Output:     int64(num(u["output_tokens"])),
			CacheRead:  int64(num(u["cache_read_input_tokens"])),
			CacheWrite: int64(num(u["cache_creation_input_tokens"])),
		}
		if d, _ := u["output_tokens_details"].(map[string]any); d != nil {
			out.tokens.Reasoning = int64(num(d["thinking_tokens"]))
		}
		out.units = out.tokens.Input + out.tokens.Output + out.tokens.CacheRead + out.tokens.CacheWrite
	}
	if blocks, _ := m["content"].([]any); blocks != nil {
		for _, b := range blocks {
			bm, _ := b.(map[string]any)
			if bm == nil {
				continue
			}
			switch bm["type"] {
			case "tool_use":
				if n, _ := bm["name"].(string); n != "" {
					out.tools = append(out.tools, n)
				}
			case "tool_result":
				if e, _ := bm["is_error"].(bool); e {
					out.errs++
				}
			}
		}
	}
	return out, true
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}
