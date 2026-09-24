package server

import (
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/climetrics"
	"github.com/cfpperche/picode/internal/pricing"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// workspaceForCwd attributes a recorded folder to its deepest registered
// workspace. Equal canonical roots are ambiguous and belong to neither.
// Empty or missing cwd is never assigned to the workspace being viewed.
func workspaceForCwd(cwd string, workspaces []store.Workspace) string {
	path := canonDir(cwd)
	if path == "" {
		return ""
	}
	owner, longest, ambiguous := "", -1, false
	for _, ws := range workspaces {
		root := canonDir(ws.Path)
		if root == "" {
			continue
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || rel != "." && (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
			continue
		}
		if len(root) > longest {
			owner, longest, ambiguous = ws.ID, len(root), false
		} else if len(root) == longest {
			ambiguous = true
		}
	}
	if ambiguous {
		return ""
	}
	return owner
}

type workspaceStatsView struct {
	Range    string                   `json:"range"`
	From     string                   `json:"from"`
	To       string                   `json:"to"`
	Current  session.PeriodTotals     `json:"current"`
	Prior    *session.PeriodTotals    `json:"prior,omitempty"`
	Series   []session.DayBucket      `json:"series"`
	Coverage []climetrics.CoverageRow `json:"coverage"`
}

func handleWorkspaceStats(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetWorkspace(id); err != nil {
			writeStoreErr(w, err)
			return
		}
		workspaces, err := deps.Store.ListWorkspaces()
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		rng := normalizeRange(r.URL.Query().Get("range"))
		from, to, priorFrom := statsWindow(rng, time.Now(), time.Local)
		// The meters share this predicate across concurrent CLI scans. A
		// recorded cwd repeats for most entries; canonicalising it once keeps
		// the scoped read close to the machine-wide scan's cost.
		var owners sync.Map
		keep := func(cwd string) bool {
			if cached, ok := owners.Load(cwd); ok {
				return cached.(string) == id
			}
			owner := workspaceForCwd(cwd, workspaces)
			owners.Store(cwd, owner)
			return owner == id
		}
		st := climetrics.Aggregate(climetrics.Request{
			From: from, To: to, PriorFrom: priorFrom, Loc: time.Local,
			Hourly: rng == "today", Prices: pricing.Current(),
			KeepCwd: keep,
		}, climetrics.Meters())
		writeJSON(w, http.StatusOK, workspaceStatsView{
			Range: rng, From: st.From, To: st.To, Current: st.Current,
			Prior: st.Prior, Series: st.Series, Coverage: st.Coverage,
		})
	}
}
