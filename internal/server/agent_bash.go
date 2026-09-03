package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// registerAgentBash wires the composer `!cmd` flow (roadmap A3):
// RPC bash in the agent cwd. Not a task kind — the tasks table CHECK
// stays prompt/steer/follow_up.
func registerAgentBash(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/bash", handleAgentBash(deps))
	mux.HandleFunc("POST /api/agents/{id}/bash/abort", handleAgentBashAbort(deps))
}

func handleAgentBash(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetAgent(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var req struct {
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		req.Command = strings.TrimSpace(req.Command)
		if req.Command == "" {
			writeErr(w, http.StatusBadRequest, "command is required")
			return
		}
		if strings.HasPrefix(req.Command, "!") {
			// `!!` hidden bash stays a TUI feature (roadmap A3 refuse row).
			writeErr(w, http.StatusBadRequest, "!! runs without sending output — use the TUI for that")
			return
		}
		ma := deps.Runtime.Get(id)
		if ma == nil {
			writeErr(w, http.StatusConflict, "agent is not running")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		res, err := ma.SendBash(ctx, req.Command)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if !res.Success {
			writeErr(w, http.StatusBadRequest, res.Error)
			return
		}
		var out struct {
			Output    string `json:"output"`
			ExitCode  int    `json:"exitCode"`
			Cancelled bool   `json:"cancelled"`
			Truncated bool   `json:"truncated"`
			FullPath  string `json:"fullOutputPath"`
		}
		_ = json.Unmarshal(res.Data, &out)
		writeJSON(w, http.StatusOK, map[string]any{
			"output": out.Output, "exitCode": out.ExitCode,
			"cancelled": out.Cancelled, "truncated": out.Truncated,
			"fullOutputPath": out.FullPath,
		})
	}
}

func handleAgentBashAbort(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ma := deps.Runtime.Get(id)
		if ma == nil {
			writeErr(w, http.StatusConflict, "agent is not running")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if err := ma.AbortBash(ctx); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
