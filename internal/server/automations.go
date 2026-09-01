package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/cron"
	"github.com/cfpperche/picode/internal/store"
)

// Automations routes (ADR-0044). Same localhost trust model as the rest
// of the API (ADR-0007) — except /fire, which is meant to be called by
// other tools and therefore carries its own per-automation secret.
func registerAutomationRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/automations", handleListAutomations(deps))
	mux.HandleFunc("POST /api/automations", handleCreateAutomation(deps))
	mux.HandleFunc("GET /api/automations/{id}", handleGetAutomation(deps))
	mux.HandleFunc("PATCH /api/automations/{id}", handlePatchAutomation(deps))
	mux.HandleFunc("DELETE /api/automations/{id}", handleDeleteAutomation(deps))
	mux.HandleFunc("POST /api/automations/{id}/secret", handleAutomationSecret(deps))
	mux.HandleFunc("POST /api/automations/{id}/run", handleRunAutomation(deps))
	mux.HandleFunc("POST /api/automations/{id}/fire", handleFireAutomation(deps))
	mux.HandleFunc("GET /api/automations/{id}/runs", handleListAutomationRuns(deps))
}

// maxWebhookPayload bounds what a webhook may append to the prompt.
const maxWebhookPayload = 64 << 10

type automationView struct {
	store.Automation
	LastRun    *store.Run `json:"lastRun,omitempty"`
	Running    bool       `json:"running"`
	NextFireAt *string    `json:"nextFireAt,omitempty"`
	Sparkline  []int      `json:"sparkline"`
	AgentName  string     `json:"agentName,omitempty"`
}

func (deps Deps) automationView(a store.Automation, now time.Time) automationView {
	v := automationView{Automation: a, Sparkline: []int{}}
	if last, err := deps.Store.LastRun(a.ID); err == nil && last != nil {
		v.LastRun = last
		v.Running = last.Status == store.RunRunning
	}
	if counts, err := deps.Store.RunCountsByDay(a.ID, 30, now); err == nil {
		v.Sparkline = counts
	}
	if a.Enabled && a.Cron != nil {
		if sched, err := cron.Parse(*a.Cron); err == nil {
			if next, ok := sched.Next(now.Truncate(time.Minute)); ok {
				s := next.Add(automate.Jitter(a.ID, sched.Interval(now))).Format(time.RFC3339)
				v.NextFireAt = &s
			}
		}
	}
	if a.AgentID != nil {
		if ag, err := deps.Store.GetAgent(*a.AgentID); err == nil {
			v.AgentName = ag.Name
		}
	}
	return v
}

func writeAutomationErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "automation not found")
		return
	}
	writeErr(w, http.StatusBadRequest, err.Error())
}

func handleListAutomations(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := deps.Store.ListAutomations()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		now := time.Now()
		out := make([]automationView, 0, len(items))
		for _, a := range items {
			out = append(out, deps.automationView(a, now))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

type automationBody struct {
	Name             *string  `json:"name"`
	Enabled          *bool    `json:"enabled"`
	WorkspaceID      *string  `json:"workspaceId"`
	Action           *string  `json:"action"`
	TargetAgentID    *string  `json:"targetAgentId"`
	Prompt           *string  `json:"prompt"`
	Provider         *string  `json:"provider"`
	Model            *string  `json:"model"`
	Thinking         *string  `json:"thinking"`
	Cron             *string  `json:"cron"`
	Webhook          *bool    `json:"webhook"`
	MaxCostUSD       *float64 `json:"maxCostUsd"`
	MaxRuns          *int     `json:"maxRuns"`
	MaxRunsWindowMin *int     `json:"maxRunsWindowMin"`
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func handleCreateAutomation(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b automationBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		a, secret, err := deps.Store.CreateAutomation(store.AutomationParams{
			Name: deref(b.Name), WorkspaceID: deref(b.WorkspaceID), Action: deref(b.Action),
			TargetAgentID: deref(b.TargetAgentID), Prompt: deref(b.Prompt),
			Provider: deref(b.Provider), Model: deref(b.Model), Thinking: deref(b.Thinking),
			Cron: deref(b.Cron), Webhook: deref(b.Webhook), MaxCostUSD: deref(b.MaxCostUSD),
			MaxRuns: deref(b.MaxRuns), MaxRunsWindowMin: deref(b.MaxRunsWindowMin),
		})
		if err != nil {
			writeAutomationErr(w, err)
			return
		}
		out := map[string]any{"automation": deps.automationView(a, time.Now())}
		if secret != "" {
			out["webhookSecret"] = secret
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func handleGetAutomation(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := deps.Store.GetAutomation(r.PathValue("id"))
		if err != nil {
			writeAutomationErr(w, err)
			return
		}
		runs, _ := deps.Store.ListRuns(a.ID, 50)
		writeJSON(w, http.StatusOK, map[string]any{"automation": deps.automationView(a, time.Now()), "runs": runs})
	}
}

func handlePatchAutomation(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var b automationBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		secret := ""
		// Webhook on before the patch so "turn webhook on, drop the cron"
		// validates; off after, so "add a cron, drop the webhook" does too.
		if b.Webhook != nil && *b.Webhook {
			cur, err := deps.Store.GetAutomation(id)
			if err != nil {
				writeAutomationErr(w, err)
				return
			}
			if !cur.Webhook {
				if secret, err = deps.Store.SetAutomationWebhook(id, true); err != nil {
					writeAutomationErr(w, err)
					return
				}
			}
		}
		a, err := deps.Store.UpdateAutomation(id, store.AutomationPatch{
			Name: b.Name, Enabled: b.Enabled, WorkspaceID: b.WorkspaceID, Action: b.Action,
			TargetAgentID: b.TargetAgentID, Prompt: b.Prompt, Provider: b.Provider, Model: b.Model,
			Thinking: b.Thinking, Cron: b.Cron, MaxCostUSD: b.MaxCostUSD, MaxRuns: b.MaxRuns,
			MaxRunsWindowMin: b.MaxRunsWindowMin,
		})
		if err != nil {
			writeAutomationErr(w, err)
			return
		}
		if b.Webhook != nil && !*b.Webhook && a.Webhook {
			if _, err := deps.Store.SetAutomationWebhook(id, false); err != nil {
				writeAutomationErr(w, err)
				return
			}
			a, _ = deps.Store.GetAutomation(id)
		}
		out := map[string]any{"automation": deps.automationView(a, time.Now())}
		if secret != "" {
			out["webhookSecret"] = secret
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleDeleteAutomation(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteAutomation(r.PathValue("id")); err != nil {
			writeAutomationErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAutomationSecret rotates the webhook secret (turning the webhook
// on if it was off). The plaintext is shown once.
func handleAutomationSecret(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret, err := deps.Store.SetAutomationWebhook(r.PathValue("id"), true)
		if err != nil {
			writeAutomationErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"webhookSecret": secret})
	}
}

// handleRunAutomation is "Run now": explicit, so it ignores the enabled
// toggle. A busy automation answers 409 and still records the skip.
func handleRunAutomation(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := deps.Store.GetAutomation(r.PathValue("id"))
		if err != nil {
			writeAutomationErr(w, err)
			return
		}
		run, err := AutomationRunner(deps).Fire(a, store.TriggerManual, "")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if run.Status == store.RunSkipped && run.Reason == reasonBusy {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "busy", "run": run})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"run": run})
	}
}

func webhookSecretFrom(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return strings.TrimSpace(r.Header.Get("X-Webhook-Secret"))
}

// handleFireAutomation is the webhook. 404 when no webhook is configured
// (indistinguishable from a missing id, on purpose), 401 on a bad secret,
// 413 on an oversized body. The body is appended to the prompt as text —
// it is never parsed, so any content type works.
func handleFireAutomation(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		a, ok, err := deps.Store.VerifyWebhookSecret(id, webhookSecretFrom(r))
		if err != nil || !a.Webhook {
			writeErr(w, http.StatusNotFound, "automation not found")
			return
		}
		if !ok {
			writeErr(w, http.StatusUnauthorized, "bad webhook secret")
			return
		}
		body, err := readLimited(r, maxWebhookPayload)
		if err != nil {
			writeErr(w, http.StatusRequestEntityTooLarge, "payload is larger than "+strconv.Itoa(maxWebhookPayload/1024)+" KB")
			return
		}
		run, err := AutomationRunner(deps).Fire(a, store.TriggerWebhook, string(body))
		if errors.Is(err, errAutomationDisabled) {
			writeErr(w, http.StatusConflict, "automation is disabled")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"run": run})
	}
}

func readLimited(r *http.Request, limit int64) ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(nil, r.Body, limit))
}

func handleListAutomationRuns(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetAutomation(id); err != nil {
			writeAutomationErr(w, err)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		runs, err := deps.Store.ListRuns(id, limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": runs})
	}
}
