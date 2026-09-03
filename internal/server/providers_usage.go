package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/usage"
)

// handleUsageSummary answers the roster from the usage cache only. A page
// load must never fan out to eight undocumented vendor endpoints, so a row
// we have not fetched yet says so ("unknown") and offers Refresh — the
// ADR-0031 rule that a guessed bar is worse than no bar, applied to the
// roster instead of the dialog.
func handleUsageSummary(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep, err := catalog.Load(deps.AgentCmd)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": usage.Summary(rep.Providers, time.Now()),
		})
	}
}

// providerIDOK keeps a path value from becoming a flag when it is handed to
// the pi CLI. Ids are catalog slugs; anything else is refused.
func providerIDOK(id string) bool {
	if id == "" || len(id) > 64 || strings.HasPrefix(id, "-") {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// handleProviderVerify asks pi whether it could actually use this provider
// right now. Cursor and Raycast both validate a key at entry; pi ships the
// primitive (`pi auth check`), so PiCode does not invent its own probe and
// does not spend a token on a test completion. --no-refresh keeps a health
// check from burning a refresh, and --credentials is never passed: the
// answer must not carry the secret.
func handleProviderVerify(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !providerIDOK(id) {
			writeErr(w, http.StatusBadRequest, "invalid provider id")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, deps.AgentCmd, "auth", "check", "--provider", id, "--json", "--no-refresh")
		out, err := cmd.Output()
		res := map[string]any{"provider": id, "ok": false}
		var parsed struct {
			Status   string `json:"status"`
			Provider string `json:"provider"`
			AuthType string `json:"authType"`
			Reason   string `json:"reason"`
		}
		if json.Unmarshal(out, &parsed) == nil && parsed.Status != "" {
			res["status"] = parsed.Status
			res["authType"] = parsed.AuthType
			res["reason"] = parsed.Reason
			res["ok"] = parsed.Status == "ready"
			writeJSON(w, http.StatusOK, res)
			return
		}
		// pi missing, or an answer we cannot read: say which, do not guess.
		msg := strings.TrimSpace(string(out))
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "pi did not answer"
		}
		res["status"] = "unknown"
		res["reason"] = msg
		writeJSON(w, http.StatusOK, res)
	}
}

// handleAccountPause keeps a credential but takes the row out of play.
// Sign out is permanent and Anthropic, OpenRouter and Raycast all keep a
// reversible disable next to it; this is that verb.
func handleAccountPause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	var req struct {
		Paused bool `json:"paused"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := catalog.PauseAccount(id, aid, req.Paused); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "account": aid, "paused": req.Paused})
}

// StartUsageRefresh keeps the active slot of each meterable provider warm so
// the roster has something true to show on the next page load. Only the
// active row refreshes on a timer (cc-switch's rule): the vendor endpoints
// are undocumented and rate-limited, and polling every account of every
// provider is how a vault gets throttled. every <= 0 disables the loop.
func StartUsageRefresh(ctx context.Context, deps Deps, every time.Duration) {
	if every <= 0 {
		return
	}
	run := func() {
		rep, err := catalog.Load(deps.AgentCmd)
		if err != nil {
			return // pi missing or offline: keep whatever the cache holds
		}
		targets := usage.ActiveTargets(rep.Providers)
		if len(targets) == 0 {
			return
		}
		rctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		if n := usage.Default.Refresh(rctx, targets); n > 0 {
			log.Printf("usage: refreshed %d provider account(s)", n)
		}
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
