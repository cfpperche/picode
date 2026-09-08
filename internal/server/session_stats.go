package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/climetrics"
	"github.com/cfpperche/picode/internal/session"
)

type sessionStatsView struct {
	climetrics.FleetStats
	Range string `json:"range"` // the clamped value actually used, not the raw query
}

// statsCache memoises one WindowStats per (root, range). An entry is served
// only while the sessions tree's Fingerprint (count/size/newest mtime — a
// stat sweep, no file opened) and the window boundaries still match, so a
// freshly appended message invalidates it on the very next request and a
// dashboard sitting on its 60s poll never re-parses an unchanged tree.
// Bounded by the four range values × one root; no eviction needed.
type statsCache struct {
	mu      sync.Mutex
	entries map[string]statsEntry
}

type statsEntry struct {
	fp       string
	from, to time.Time
	stats    climetrics.FleetStats
}

var sessionStats statsCache

func (c *statsCache) get(key, fp string, from, to time.Time) (climetrics.FleetStats, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	// An empty fingerprint means a meter could not describe its own state;
	// serving that from cache would pin a stale window forever.
	if !ok || fp == "" || e.fp != fp || !e.from.Equal(from) || !e.to.Equal(to) {
		return climetrics.FleetStats{}, false
	}
	return e.stats, true
}

func (c *statsCache) put(key, fp string, from, to time.Time, st climetrics.FleetStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]statsEntry{}
	}
	c.entries[key] = statsEntry{fp: fp, from: from, to: to, stats: st}
}

func (c *statsCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = nil
}

func handleSessionStats(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rng := normalizeRange(r.URL.Query().Get("range"))
		scope := normalizeScope(r.URL.Query().Get("scope"))
		from, to, priorFrom := statsWindow(rng, time.Now(), time.Local)
		meters := climetrics.Meters()
		claimed := claimedDirs(deps)

		// The claimed folders are part of the cache identity, not just the
		// request: a picode-scoped window answered from cache after a new
		// workspace was added would keep excluding that folder until some
		// unrelated session file happened to move the fingerprint.
		fp := climetrics.Fingerprint(meters) + "|ws=" + strings.Join(claimed, ",")
		key := session.Root() + "|" + rng + "|" + string(scope)
		st, hit := sessionStats.get(key, fp, from, to)
		if !hit {
			st = climetrics.Aggregate(climetrics.Request{
				From: from, To: to, PriorFrom: priorFrom,
				Loc:   time.Local,
				Scope: scope,
				// climetrics never sees the store: the handler is the only
				// layer that knows which folders PiCode claims, the same
				// boundary ADR-0042 drew for workspace labels.
				Claimed: claimed,
			}, meters)
			sessionStats.put(key, fp, from, to, st)
		}
		st.WindowStats = labelWorkspaces(deps, st.WindowStats)
		writeJSON(w, http.StatusOK, sessionStatsView{FleetStats: st, Range: rng})
	}
}

// claimedDirs is every workspace folder, canonicalised. A picode-scoped
// window counts a session only when its cwd is one of these or sits beneath
// one, so a worktree rolls up into the workspace that owns it instead of
// splitting the breakdown in two.
func claimedDirs(deps Deps) []string {
	if deps.Store == nil {
		return nil
	}
	wss, err := deps.Store.ListWorkspaces()
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(wss))
	for _, w := range wss {
		if d := canonDir(w.Path); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// normalizeScope clamps to the two supported values, defaulting rather than
// 400ing on drift — the same spirit as normalizeRange.
//
// The default is the whole machine, not PiCode's own folders. The v1
// dashboard counted every pi session wherever it ran, and a narrower
// default would silently drop rows the surface has always shown; the
// picode scope is the opt-in narrowing, not the baseline.
func normalizeScope(raw string) climetrics.Scope {
	switch strings.TrimSpace(raw) {
	case string(climetrics.ScopePiCode):
		return climetrics.ScopePiCode
	default:
		return climetrics.ScopeMachine
	}
}

// labelWorkspaces resolves each cwd bucket to the PiCode workspace that owns
// that folder, the same canonDir match handleAllSessions uses. Folders no
// workspace claims keep only their cwd. The session package never sees the
// store, so this stays a server concern; the slices are copied so the
// cached WindowStats is never mutated.
func labelWorkspaces(deps Deps, st session.WindowStats) session.WindowStats {
	if deps.Store == nil {
		return st
	}
	wss, err := deps.Store.ListWorkspaces()
	if err != nil || len(wss) == 0 {
		return st
	}
	byDir := map[string]struct{ id, name string }{}
	for _, wk := range wss {
		byDir[canonDir(wk.Path)] = struct{ id, name string }{wk.ID, wk.Name}
	}
	ws := make([]session.WorkspaceBucket, len(st.ByWorkspace))
	for i, b := range st.ByWorkspace {
		if w, ok := byDir[canonDir(b.Cwd)]; ok {
			b.WorkspaceID, b.Workspace = w.id, w.name
		}
		ws[i] = b
	}
	top := make([]session.SessionSpend, len(st.TopSessions))
	for i, s := range st.TopSessions {
		if w, ok := byDir[canonDir(s.Cwd)]; ok {
			s.WorkspaceID, s.Workspace = w.id, w.name
		}
		top[i] = s
	}
	st.ByWorkspace = ws
	st.TopSessions = top
	return st
}

// normalizeRange clamps to the 4 supported values, defaulting rather than
// 400ing on drift — same spirit as gitgraph.go's graphLimit/graphRemotes.
func normalizeRange(raw string) string {
	switch strings.TrimSpace(raw) {
	case "today", "7d", "30d", "all":
		return strings.TrimSpace(raw)
	default:
		return "7d"
	}
}

// statsWindow computes [from,to) for the current period and priorFrom for
// the equal-length preceding period, in loc. to is always the start of
// tomorrow (today's activity-in-progress counts). priorFrom is the zero
// time for "all" — the caller's signal to skip the prior comparison
// entirely, since there is no well-defined "period before all time".
//
// Day boundaries use AddDate (calendar-day arithmetic), not a fixed
// Duration subtraction, so a DST transition never produces a 23h/25h window.
func statsWindow(rng string, now time.Time, loc *time.Location) (from, to, priorFrom time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)
	switch rng {
	case "today":
		return today, tomorrow, today.AddDate(0, 0, -1)
	case "30d":
		from = today.AddDate(0, 0, -29)
		return from, tomorrow, from.AddDate(0, 0, -30)
	case "all":
		return time.Time{}, tomorrow, time.Time{}
	default: // "7d" and the normalizeRange fallback
		from = today.AddDate(0, 0, -6)
		return from, tomorrow, from.AddDate(0, 0, -7)
	}
}
