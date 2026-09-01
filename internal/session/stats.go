package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// DayBucket is one calendar day's totals in the current period's series.
type DayBucket struct {
	Date     string  `json:"date"` // YYYY-MM-DD, in the caller's Location
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
	Turns    int     `json:"turns"` // assistant messages (one per model call)
}

// ProviderBucket is one provider's totals in the current period.
type ProviderBucket struct {
	Provider string  `json:"provider"` // "unknown" when neither the message nor a model_change line named one
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
}

// ModelBucket is one provider/model pair's totals in the current period.
type ModelBucket struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
}

// WorkspaceBucket is one session folder's totals in the current period.
// Cwd is what pi recorded on the session line; WorkspaceID/Workspace are
// filled in by the server, which is the only layer that knows the store.
type WorkspaceBucket struct {
	Cwd         string  `json:"cwd"`
	WorkspaceID string  `json:"workspaceId,omitempty"`
	Workspace   string  `json:"workspace,omitempty"`
	Cost        float64 `json:"cost"`
	Messages    int     `json:"messages"`
	Sessions    int     `json:"sessions"`
}

// TokenTotals sums every assistant message's usage in the current period.
// CacheHit is cacheRead/(input+cacheRead) in percent — the same formula the
// composer status bar uses (status.go) — nil when there was no prompt input.
type TokenTotals struct {
	Input      int64    `json:"input"`
	Output     int64    `json:"output"`
	CacheRead  int64    `json:"cacheRead"`
	CacheWrite int64    `json:"cacheWrite"`
	Reasoning  int64    `json:"reasoning"`
	CacheHit   *float64 `json:"cacheHit,omitempty"`
}

// ToolBucket is how many times one tool was called in the current period.
type ToolBucket struct {
	Name  string `json:"name"`
	Calls int    `json:"calls"`
}

// TurnStats counts conversation beats in the current period. Errors and
// Aborted come from the assistant message's stopReason; Compactions from
// pi's compaction marker lines.
type TurnStats struct {
	Assistant   int `json:"assistant"`
	User        int `json:"user"`
	Errors      int `json:"errors"`
	Aborted     int `json:"aborted"`
	Compactions int `json:"compactions"`
}

// SessionSpend is one session's in-period totals for the top-N list. Name
// and Cwd only — never Preview (the dashboard is an aggregate surface).
type SessionSpend struct {
	Path        string  `json:"path"`
	Name        string  `json:"name,omitempty"`
	Cwd         string  `json:"cwd"`
	WorkspaceID string  `json:"workspaceId,omitempty"`
	Workspace   string  `json:"workspace,omitempty"`
	Cost        float64 `json:"cost"`
	Messages    int     `json:"messages"`
	LastAt      string  `json:"lastAt"` // RFC3339, newest in-window message
}

// PeriodTotals is the aggregate over one window.
type PeriodTotals struct {
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
	Sessions int     `json:"sessions"` // distinct session files with >=1 in-window message
}

// WindowStats is the response shape for a spend/activity aggregation window.
// Everything but Prior describes the current period only.
type WindowStats struct {
	From        string            `json:"from"` // RFC3339, in loc
	To          string            `json:"to"`   // RFC3339, exclusive, in loc
	Current     PeriodTotals      `json:"current"`
	Prior       *PeriodTotals     `json:"prior,omitempty"` // nil when priorFrom is the zero time (range=all)
	ByProvider  []ProviderBucket  `json:"byProvider"`      // sorted desc by Cost
	ByModel     []ModelBucket     `json:"byModel"`         // sorted desc by Cost
	ByWorkspace []WorkspaceBucket `json:"byWorkspace"`     // sorted desc by Cost
	Tokens      TokenTotals       `json:"tokens"`
	Tools       []ToolBucket      `json:"tools"` // sorted desc by Calls, top TopTools
	Turns       TurnStats         `json:"turns"`
	TopSessions []SessionSpend    `json:"topSessions"` // sorted desc by Cost, top TopSessions
	Series      []DayBucket       `json:"series"`      // one entry per calendar day, zero-filled, ascending
}

// TopTools / TopSessionsN cap the ranked lists — a dashboard row, not a
// report. The full ranking is one scan away if a future surface wants it.
const (
	TopTools     = 8
	TopSessionsN = 5
)

type modelKey struct{ provider, model string }

// fileAgg is one session file's in-window totals plus the identity facts
// (cwd, name) pi writes on its own header lines.
type fileAgg struct {
	path     string
	cwd      string
	name     string
	cost     float64
	messages int
	last     time.Time
}

type statsAcc struct {
	from, to, priorFrom time.Time
	loc                 *time.Location
	current             PeriodTotals
	prior               PeriodTotals
	curSessions         map[string]*fileAgg
	priorSessions       map[string]struct{}
	byProvider          map[string]*ProviderBucket
	byModel             map[modelKey]*ModelBucket
	byDay               map[string]*DayBucket
	tools               map[string]int
	tokens              TokenTotals
	turns               TurnStats
}

func newStatsAcc(from, to, priorFrom time.Time, loc *time.Location) *statsAcc {
	return &statsAcc{
		from: from, to: to, priorFrom: priorFrom, loc: loc,
		curSessions:   map[string]*fileAgg{},
		priorSessions: map[string]struct{}{},
		byProvider:    map[string]*ProviderBucket{},
		byModel:       map[modelKey]*ModelBucket{},
		byDay:         map[string]*DayBucket{},
		tools:         map[string]int{},
	}
}

// StatsForRange aggregates every session under the real sessions root.
func StatsForRange(from, to, priorFrom time.Time, loc *time.Location) (WindowStats, error) {
	return StatsRoot(Root(), from, to, priorFrom, loc)
}

// StatsRoot aggregates every session under root for [from,to), plus an
// equal-length prior window [priorFrom,from) for the delta comparison.
// priorFrom.IsZero() skips the prior computation (range=all).
func StatsRoot(root string, from, to, priorFrom time.Time, loc *time.Location) (WindowStats, error) {
	acc := newStatsAcc(from, to, priorFrom, loc)
	if root != "" {
		ents, err := os.ReadDir(root)
		if err != nil && !os.IsNotExist(err) {
			return WindowStats{}, err
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			acc.scanDir(filepath.Join(root, e.Name()))
		}
	}
	return acc.result(), nil
}

// Fingerprint is a cheap change detector for a sessions root: the count,
// total size and newest mtime of every JSONL under it. Appending to a
// session changes its size and mtime; creating or deleting one changes the
// count. Stat-only — never opens a file — so a caller can check it on every
// request and only re-run the (expensive) scan when it differs.
func Fingerprint(root string) string {
	if root == "" {
		return ""
	}
	var n int
	var size int64
	var newest int64
	ents, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, e.Name()))
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
			n++
			size += info.Size()
			if m := info.ModTime().UnixNano(); m > newest {
				newest = m
			}
		}
	}
	return strconv.Itoa(n) + ":" + strconv.FormatInt(size, 10) + ":" + strconv.FormatInt(newest, 10)
}

// scanDir walks one workspace's session directory, skipping files whose
// mtime cannot possibly hold an in-window message (mirrors the pre-filter
// sweepOrphanSessions already uses for the same reason: cheap to check,
// expensive to open and parse).
func (a *statsAcc) scanDir(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !a.priorFrom.IsZero() && info.ModTime().Before(a.priorFrom) {
			continue // cannot contain any message in [priorFrom, to)
		}
		a.scanFile(filepath.Join(dir, e.Name()), info.ModTime())
	}
}

// msgFacts is what one message line contributes beyond cost/count.
type msgFacts struct {
	role       string
	provider   string
	model      string
	stopReason string
	usage      TokenTotals
	tools      []string
}

func (a *statsAcc) scanFile(path string, mtime time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	provider := ""
	model := ""
	cwd := filepath.Base(filepath.Dir(path)) // fallback: pi's encoded folder name
	name := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		switch raw["type"] {
		case "session":
			if c, _ := raw["cwd"].(string); c != "" {
				cwd = c
			}
		case "session_info":
			if n, _ := raw["name"].(string); n != "" {
				name = n
			}
		case "model_change":
			if p, _ := raw["provider"].(string); p != "" {
				provider = p
			}
			if m, _ := raw["modelId"].(string); m != "" {
				model = m
			}
		case "compaction", "compaction_summary":
			// pi writes no timestamp on the marker; the file's mtime is the
			// best available "when" and only decides which window it counts in.
			if a.inCurrent(mtime) {
				a.turns.Compactions++
			}
		case "message":
			ts := entryTS(raw)
			t := mtime
			if ts > 0 {
				// pi writes message.timestamp in epoch milliseconds
				// (JS Date.now() convention), not seconds.
				t = time.UnixMilli(ts)
			}
			facts := factsFrom(raw["message"])
			// An assistant message names its own provider/model and
			// becomes the running truth for the user/tool lines that
			// follow it; a model_change line is only the fallback for
			// messages that carry neither (older pi versions).
			if facts.provider != "" {
				provider = facts.provider
			} else {
				facts.provider = provider
			}
			if facts.model != "" {
				model = facts.model
			} else {
				facts.model = model
			}
			a.add(path, cwd, name, t, facts, costFrom(raw["message"]))
		}
	}
}

func (a *statsAcc) inCurrent(t time.Time) bool {
	if !t.Before(a.to) {
		return false
	}
	return a.from.IsZero() || !t.Before(a.from)
}

func factsFrom(msg any) msgFacts {
	var out msgFacts
	m, _ := msg.(map[string]any)
	if m == nil {
		return out
	}
	out.role, _ = m["role"].(string)
	out.provider, _ = m["provider"].(string)
	out.model, _ = m["model"].(string)
	out.stopReason, _ = m["stopReason"].(string)
	if u, _ := m["usage"].(map[string]any); u != nil {
		get := func(key string) int64 {
			if v, ok := u[key].(float64); ok {
				return int64(v)
			}
			return 0
		}
		out.usage = TokenTotals{
			Input:      get("input"),
			Output:     get("output"),
			CacheRead:  get("cacheRead"),
			CacheWrite: get("cacheWrite"),
			Reasoning:  get("reasoning"),
		}
	}
	if blocks, _ := m["content"].([]any); blocks != nil {
		for _, b := range blocks {
			bm, _ := b.(map[string]any)
			if bm == nil || bm["type"] != "toolCall" {
				continue
			}
			if n, _ := bm["name"].(string); n != "" {
				out.tools = append(out.tools, n)
			}
		}
	}
	return out
}

// add files one message's cost/count into whichever window t falls in.
// Only the current window gets the breakdowns; the prior window exists
// solely for the headline delta.
func (a *statsAcc) add(path, cwd, name string, t time.Time, f msgFacts, cost float64) {
	prov := f.provider
	if prov == "" {
		prov = "unknown"
	}
	switch {
	case !a.priorFrom.IsZero() && t.Before(a.priorFrom):
		return // older than the widened (current+prior) window
	case !t.Before(a.to):
		return // in the future relative to the window (defensive; should not happen)
	case !a.from.IsZero() && t.Before(a.from):
		a.prior.Cost += cost
		a.prior.Messages++
		a.priorSessions[path] = struct{}{}
	default:
		a.current.Cost += cost
		a.current.Messages++

		fa := a.curSessions[path]
		if fa == nil {
			fa = &fileAgg{path: path}
			a.curSessions[path] = fa
		}
		fa.cwd, fa.name = cwd, name
		fa.cost += cost
		fa.messages++
		if t.After(fa.last) {
			fa.last = t
		}

		pb := a.byProvider[prov]
		if pb == nil {
			pb = &ProviderBucket{Provider: prov}
			a.byProvider[prov] = pb
		}
		pb.Cost += cost
		pb.Messages++

		day := t.In(a.loc).Format("2006-01-02")
		db := a.byDay[day]
		if db == nil {
			db = &DayBucket{Date: day}
			a.byDay[day] = db
		}
		db.Cost += cost
		db.Messages++

		switch f.role {
		case "user":
			a.turns.User++
		case "assistant":
			a.turns.Assistant++
			db.Turns++
			switch f.stopReason {
			case "error":
				a.turns.Errors++
			case "aborted":
				a.turns.Aborted++
			}
			mk := modelKey{prov, f.model}
			if mk.model == "" {
				mk.model = "unknown"
			}
			mb := a.byModel[mk]
			if mb == nil {
				mb = &ModelBucket{Provider: mk.provider, Model: mk.model}
				a.byModel[mk] = mb
			}
			mb.Cost += cost
			mb.Messages++
			a.tokens.Input += f.usage.Input
			a.tokens.Output += f.usage.Output
			a.tokens.CacheRead += f.usage.CacheRead
			a.tokens.CacheWrite += f.usage.CacheWrite
			a.tokens.Reasoning += f.usage.Reasoning
			for _, n := range f.tools {
				a.tools[n]++
			}
		}
	}
}

func (a *statsAcc) result() WindowStats {
	a.current.Sessions = len(a.curSessions)
	a.prior.Sessions = len(a.priorSessions)

	providers := make([]ProviderBucket, 0, len(a.byProvider))
	for _, pb := range a.byProvider {
		providers = append(providers, *pb)
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].Cost > providers[j].Cost })

	models := make([]ModelBucket, 0, len(a.byModel))
	for _, mb := range a.byModel {
		models = append(models, *mb)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Cost != models[j].Cost {
			return models[i].Cost > models[j].Cost
		}
		return models[i].Model < models[j].Model
	})

	byWs := map[string]*WorkspaceBucket{}
	sessions := make([]SessionSpend, 0, len(a.curSessions))
	for _, fa := range a.curSessions {
		wb := byWs[fa.cwd]
		if wb == nil {
			wb = &WorkspaceBucket{Cwd: fa.cwd}
			byWs[fa.cwd] = wb
		}
		wb.Cost += fa.cost
		wb.Messages += fa.messages
		wb.Sessions++
		sessions = append(sessions, SessionSpend{Path: fa.path, Name: fa.name, Cwd: fa.cwd, Cost: fa.cost, Messages: fa.messages, LastAt: fa.last.In(a.loc).Format(time.RFC3339)})
	}
	workspaces := make([]WorkspaceBucket, 0, len(byWs))
	for _, wb := range byWs {
		workspaces = append(workspaces, *wb)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		if workspaces[i].Cost != workspaces[j].Cost {
			return workspaces[i].Cost > workspaces[j].Cost
		}
		return workspaces[i].Cwd < workspaces[j].Cwd
	})
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Cost != sessions[j].Cost {
			return sessions[i].Cost > sessions[j].Cost
		}
		return sessions[i].Path < sessions[j].Path
	})
	if len(sessions) > TopSessionsN {
		sessions = sessions[:TopSessionsN]
	}

	tools := make([]ToolBucket, 0, len(a.tools))
	for n, c := range a.tools {
		tools = append(tools, ToolBucket{Name: n, Calls: c})
	}
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Calls != tools[j].Calls {
			return tools[i].Calls > tools[j].Calls
		}
		return tools[i].Name < tools[j].Name
	})
	if len(tools) > TopTools {
		tools = tools[:TopTools]
	}

	tok := a.tokens
	if prompt := tok.Input + tok.CacheRead; prompt > 0 {
		h := 100 * float64(tok.CacheRead) / float64(prompt)
		tok.CacheHit = &h
	}

	st := WindowStats{
		From:        a.from.Format(time.RFC3339),
		To:          a.to.Format(time.RFC3339),
		Current:     a.current,
		ByProvider:  providers,
		ByModel:     models,
		ByWorkspace: workspaces,
		Tokens:      tok,
		Tools:       tools,
		Turns:       a.turns,
		TopSessions: sessions,
		Series:      a.series(),
	}
	if !a.priorFrom.IsZero() {
		p := a.prior
		st.Prior = &p
	}
	return st
}

// series zero-fills every calendar day in the current window. For range=all
// (a.from is the zero time) the walk starts at the earliest day that
// actually has data instead of the zero time — iterating from year 1 would
// be unbounded and pointless. No data at all yields an empty series, which
// is the honest "nothing happened" signal rather than a fabricated range.
func (a *statsAcc) series() []DayBucket {
	from := a.from
	if from.IsZero() {
		if len(a.byDay) == 0 {
			return []DayBucket{}
		}
		days := make([]string, 0, len(a.byDay))
		for d := range a.byDay {
			days = append(days, d)
		}
		sort.Strings(days)
		earliest, err := time.ParseInLocation("2006-01-02", days[0], a.loc)
		if err != nil {
			return []DayBucket{}
		}
		from = earliest
	}
	out := []DayBucket{}
	for d := from; d.Before(a.to); d = d.AddDate(0, 0, 1) {
		key := d.In(a.loc).Format("2006-01-02")
		if db := a.byDay[key]; db != nil {
			out = append(out, *db)
		} else {
			out = append(out, DayBucket{Date: key})
		}
	}
	return out
}
