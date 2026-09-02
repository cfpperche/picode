package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/cron"
	"github.com/cfpperche/picode/internal/store"
)

// Automations routes (ADR-0045). Same localhost trust model as the rest
// of the API (ADR-0007) — except /fire, which is meant to be called by
// other tools and therefore carries its own per-automation secret.
func registerAutomationRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/automations", handleListAutomations(deps))
	mux.HandleFunc("GET /api/automations/templates", handleAutomationTemplates)
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

// handleAutomationTemplates serves the built-in suggestions (ADR-0045 v2).
func handleAutomationTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": automate.Templates()})
}

type automationView struct {
	store.Automation
	LastRun    *store.Run `json:"lastRun,omitempty"`
	Running    bool       `json:"running"`
	NextFireAt *string    `json:"nextFireAt,omitempty"`
	Sparkline  []int      `json:"sparkline"`
	AgentName  string     `json:"agentName,omitempty"`
	WebhookURL string     `json:"webhookUrl,omitempty"` // where a caller reaches /fire from where it is (ADR-0045 amendment)
}

func (deps Deps) automationView(a store.Automation, now time.Time) automationView {
	v := automationView{Automation: a, Sparkline: []int{}}
	if a.Webhook {
		v.WebhookURL = deps.webhookURL(a.ID)
	}
	if last, err := deps.Store.LastRun(a.ID); err == nil && last != nil {
		v.LastRun = last
		v.Running = last.Status == store.RunRunning
	}
	if counts, err := deps.Store.RunCountsByDay(a.ID, 30, now); err == nil {
		v.Sparkline = counts
	}
	if a.Enabled && a.Cron != nil {
		if sched, err := cron.Parse(*a.Cron); err == nil {
			if next, ok := automate.NextFire(sched, a.ID, now); ok {
				s := next.Format(time.RFC3339)
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
	NotifyURL        *string  `json:"notifyUrl"`
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
			Cron: deref(b.Cron), Webhook: deref(b.Webhook), NotifyURL: deref(b.NotifyURL), MaxCostUSD: deref(b.MaxCostUSD),
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
			Thinking: b.Thinking, Cron: b.Cron, NotifyURL: b.NotifyURL, MaxCostUSD: b.MaxCostUSD, MaxRuns: b.MaxRuns,
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

// linuxUser is this daemon's account name, the member key a gateway
// routes by (ADR-0051); "" when unknown.
var linuxUser = func() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}()

// webhookURL is the /fire address a caller can use from where it is:
// the gateway's hook route when this daemon sits behind one (plain HTTP
// with a public URL), the public URL when one is set, else "" and the UI
// falls back to the browser's own origin.
func (deps Deps) webhookURL(id string) string {
	pub := strings.TrimRight(deps.publicURL(), "/")
	if pub == "" {
		return ""
	}
	if deps.Insecure && linuxUser != "" {
		return pub + "/-/hook/" + url.PathEscape(linuxUser) + "/" + url.PathEscape(id)
	}
	return pub + "/api/automations/" + url.PathEscape(id) + "/fire"
}
