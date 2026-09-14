package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/modellist"
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
// handleProviderVerify asks whether the provider works. For a built-in that
// means pi's own answer (`pi auth check`), which is the code path that will run
// the agent. A custom endpoint is different: pi only reports the credential's
// presence, so a wrong key on a gateway reads green. Those are verified with
// one real, minimal request to the endpoint itself (the owner's call; open
// topic P4) — a fraction of a cent, and the answer is the endpoint's.
func handleProviderVerify(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !providerIDOK(id) {
			writeErr(w, http.StatusBadRequest, "invalid provider id")
			return
		}
		// Dispatch: a body carrying a base URL is the dialog verifying what the
		// form holds (the definition may not be saved yet), and a saved custom
		// definition is the roster's row. Everything else is a built-in, which
		// only pi can answer for.
		var req struct {
			BaseURL string `json:"baseUrl"`
			API     string `json:"api"`
			Key     string `json:"key"`
			Model   string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		def, isCustom := catalog.LoadCustomDefinitions()[id]
		if isCustom || strings.TrimSpace(req.BaseURL) != "" {
			verifyCustomEndpoint(w, r.Context(), id, def, req.BaseURL, req.API, req.Key, req.Model)
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

// verifyCustomEndpoint spends one minimal real request on a custom endpoint and
// reports what the endpoint said. It answers with the same shape the roster's
// verify uses ({ok, status, reason}) plus what was actually spent, so the row
// renders it without knowing which path ran.
func verifyCustomEndpoint(w http.ResponseWriter, ctx context.Context, id string, def catalog.CustomDefinition, bodyBaseURL, bodyAPI, bodyKey, bodyModel string) {
	baseURL := strings.TrimSpace(bodyBaseURL)
	if baseURL == "" {
		baseURL = def.BaseURL
	}
	api := strings.TrimSpace(bodyAPI)
	if api == "" {
		api = def.API
	}
	key := strings.TrimSpace(bodyKey)
	if key == "" {
		if saved, ok := catalog.ActiveAPIKey(id); ok {
			key = saved
		}
	}
	model := strings.TrimSpace(bodyModel)
	if model == "" && len(def.Models) > 0 {
		model = def.Models[0].ID
	}

	probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	res, err := modellist.Probe(probeCtx, baseURL, api, key, model)
	if err != nil {
		var typed *modellist.Error
		if !errors.As(err, &typed) {
			writeJSON(w, http.StatusOK, map[string]any{
				"provider": id, "ok": false, "status": "unknown", "spent": true,
				"reason": "The endpoint could not be verified.",
			})
			return
		}
		status, body := probeFailureReply(typed, model, key != "")
		body["provider"] = id
		body["spent"] = typed.Status != 0
		writeJSON(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider":     id,
		"ok":           true,
		"status":       "ready",
		"authType":     "api_key",
		"verified":     true,
		"spent":        true,
		"model":        res.Model,
		"ms":           res.MS,
		"label":        probeSuccessLabel(res),
		"inputTokens":  res.InputTokens,
		"outputTokens": res.OutputTokens,
	})
}

// probeSuccessLabel is what the row shows after a real request: the model that
// answered, how long it took and what it cost in tokens (when reported).
func probeSuccessLabel(res modellist.ProbeResult) string {
	label := res.Model + " answered in " + strconv.Itoa(res.MS) + "ms"
	if res.InputTokens > 0 || res.OutputTokens > 0 {
		label += " (" + strconv.Itoa(res.InputTokens) + " in, " + strconv.Itoa(res.OutputTokens) + " out)"
	}
	return label
}

// probeFailureReply is the one line a failed probe shows. It names the model
// where that is the answer, and never repeats the key.
func probeFailureReply(e *modellist.Error, model string, keyed bool) (int, map[string]any) {
	label := ""
	switch e.Kind {
	case modellist.KindAuth:
		if keyed {
			return http.StatusOK, map[string]any{"ok": false, "status": "refused", "verified": false,
				"reason": "The endpoint refused the key (" + strconv.Itoa(e.Status) + ")" + detailTail(e.Detail) + "."}
		}
		return http.StatusOK, map[string]any{"ok": false, "status": "refused", "verified": false,
			"reason": "The endpoint wants a key (" + strconv.Itoa(e.Status) + "). Add one and try again."}
	case modellist.KindQuota:
		label = "The account cannot make a request (" + strconv.Itoa(e.Status) + ")" + detailTail(e.Detail) + ". Billing or quota needs attention first."
	case modellist.KindModel:
		label = "The endpoint does not know " + model + " (" + strconv.Itoa(e.Status) + ")" + detailTail(e.Detail) + ". Check the model id, or the base URL's route."
	case modellist.KindInput:
		// A refusal from the endpoint carries its status; a request PiCode could
		// not even build has none, and printing "(0)" would be nonsense. The
		// supported list is named because that is the one input case a person
		// can act on (a hand-edited `api` value).
		if e.Status == 0 {
			label = capitalize(e.Detail) + "." + hintTail(e.Hint)
			break
		}
		if !keyed {
			label = "The endpoint answered " + strconv.Itoa(e.Status) + detailTail(e.Detail) +
				". Some gateways need the API key in the request."
		} else {
			label = "The endpoint rejected the request (" + strconv.Itoa(e.Status) + ")" + detailTail(e.Detail) + "."
		}
	case modellist.KindTransport:
		label = "Could not reach " + e.Host + ": " + e.Detail + "."
	default:
		label = "The endpoint answered " + strconv.Itoa(e.Status) + detailTail(e.Detail) + "."
	}
	return http.StatusOK, map[string]any{"ok": false, "status": "unknown", "verified": false, "reason": label}
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
