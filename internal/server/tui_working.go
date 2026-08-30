package server

import (
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/tmux"
)

// handleTuiWorking reports which agents' interactive pi TUIs are currently
// busy. The TUI has no event channel (ADR-0006: one process, no RPC while
// interactive), so the only honest signal is the pane itself: capture the
// tail and look for pi's "Working…" status line.
func handleTuiWorking(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeJSON(w, http.StatusOK, map[string]any{"working": []string{}})
			return
		}
		q := r.URL.Query().Get("ids")
		ids := strings.Split(q, ",")
		if len(ids) > 24 {
			ids = ids[:24]
		}
		out := []string{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			name := tmux.SessionName(id)
			has, err := deps.Tmux.HasSession(r.Context(), name)
			if err != nil || !has {
				continue
			}
			tail, err := deps.Tmux.CaptureTail(r.Context(), name, 8)
			if err != nil {
				continue
			}
			if tmux.LooksWorking(tail) {
				out = append(out, id)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"working": out})
	}
}
