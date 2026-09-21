package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/usage"
)

func registerSlashOps(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/trust", handleAgentTrust(deps))
	mux.HandleFunc("PUT /api/providers/custom/{id}", handleCustomProviderPut)
	mux.HandleFunc("DELETE /api/providers/custom/{id}", handleCustomProviderDelete)
	// Load models: the literal path wins over the {id} pattern, so this route
	// can sit beside them (Go 1.22 mux prefers the more specific pattern).
	mux.HandleFunc("POST /api/providers/custom/models", handleCustomProviderModels)
	mux.HandleFunc("PUT /api/providers/{id}", handleProviderLogin)
	mux.HandleFunc("DELETE /api/providers/{id}", handleProviderLogout(deps))
	mux.HandleFunc("POST /api/providers/{id}/accounts/{aid}/activate", handleAccountActivate)
	mux.HandleFunc("PATCH /api/providers/{id}/accounts/{aid}", handleAccountRename)
	mux.HandleFunc("DELETE /api/providers/{id}/accounts/{aid}", handleAccountDelete)
	mux.HandleFunc("GET /api/providers/usage", handleUsageSummary(deps))
	mux.HandleFunc("POST /api/providers/{id}/verify", handleProviderVerify(deps))
	mux.HandleFunc("POST /api/providers/{id}/accounts/{aid}/pause", handleAccountPause)
	mux.HandleFunc("GET /api/providers/{id}/usage", handleProviderUsage)
	mux.HandleFunc("POST /api/providers/{id}/usage/reset", handleProviderUsageReset)
	mux.HandleFunc("GET /api/providers/{id}/accounts/{aid}/usage", handleAccountUsage)
	mux.HandleFunc("POST /api/providers/{id}/accounts/{aid}/usage/reset", handleAccountUsageReset)
}

func handleProviderUsage(w http.ResponseWriter, r *http.Request) {
	rep := usage.Fetch(r.Context(), r.PathValue("id"))
	writeJSON(w, http.StatusOK, rep)
}

func handleProviderUsageReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	rep := usage.Redeem(r.Context(), r.PathValue("id"), req.ID)
	writeJSON(w, http.StatusOK, rep)
}

func handleAccountUsage(w http.ResponseWriter, r *http.Request) {
	rep := usage.FetchAccount(r.Context(), r.PathValue("id"), r.PathValue("aid"))
	writeJSON(w, http.StatusOK, rep)
}

func handleAccountUsageReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	rep := usage.RedeemAccount(r.Context(), r.PathValue("id"), r.PathValue("aid"), req.ID)
	writeJSON(w, http.StatusOK, rep)
}

func handleAgentTrust(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd := store.AgentCwd(wk, agent)
		if pisettings.Trusted(cwd) {
			writeJSON(w, http.StatusOK, map[string]any{"trusted": true, "cwd": cwd, "already": true})
			return
		}
		if err := pisettings.Set(cwd, true); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"trusted": true, "cwd": cwd, "already": false})
	}
}

func handleProviderLogin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Key string `json:"key"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var err error
	if id == "llama.cpp" {
		err = catalog.PutLlama(req.URL, req.Key)
	} else {
		err = catalog.PutAPIKey(id, req.Key)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "signedIn": true})
}

// handleCustomProviderPut creates or updates one custom provider definition
// (ADR-0129). For pi the definition merges into models.json and the key,
// when present, goes to auth.json like any native sign-in — never into
// models.json. For omp (the owner's amendment to ADR-0169) the definition
// and its key merge into omp's models.yml, the one channel a custom id has
// there; the key is never returned by any read.
func handleCustomProviderPut(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		BaseURL            string                `json:"baseUrl"`
		API                string                `json:"api"`
		Compat             map[string]bool       `json:"compat"`
		ThinkingFormat     string                `json:"thinkingFormat"`
		ChatTemplateKwargs map[string]any        `json:"chatTemplateKwargs"`
		ChatTemplateArgs   map[string]any        `json:"chatTemplateArgs"`
		Models             []catalog.CustomModel `json:"models"`
		Key                string                `json:"key"`
		CLI                string                `json:"cli"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	def := catalog.CustomDefinition{
		BaseURL: req.BaseURL, API: req.API, Models: req.Models, ThinkingFormat: req.ThinkingFormat,
		ChatTemplateKwargs: req.ChatTemplateKwargs, ChatTemplateArgs: req.ChatTemplateArgs,
	}
	if def.API == "" {
		def.API = catalog.APIOpenAICompletions
	}
	if req.Compat != nil {
		// Only the keys the form manages are writable; unknown hand-set keys
		// survive. A missing object leaves an existing compat block untouched.
		def.Compat = map[string]bool{}
		for _, k := range []string{"supportsDeveloperRole", "supportsReasoningEffort", "supportsUsageInStreaming"} {
			if v, ok := req.Compat[k]; ok {
				def.Compat[k] = v
			}
		}
	}
	switch strings.TrimSpace(req.CLI) {
	case "omp":
		if err := catalog.OMPUpsertCustomProvider(id, def, req.Key); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "custom": true})
	case "", "pi":
		if err := catalog.UpsertCustomProvider(id, def); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if key := strings.TrimSpace(req.Key); key != "" {
			if err := catalog.PutAPIKey(id, key); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "custom": true})
	default:
		writeErr(w, http.StatusBadRequest, "This CLI has no custom provider surface.")
	}
}

// handleCustomProviderDelete removes the definition, the active auth.json
// credential and any vault rows. For omp it removes the definition and its
// models.yml key; there is no vault row to forget. Built-in overrides are
// refused, not deleted.
func handleCustomProviderDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cli := strings.TrimSpace(r.URL.Query().Get("cli"))
	if cli == "omp" {
		if err := catalog.OMPRemoveCustomProvider(id); err != nil {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "removed": true})
		return
	}
	if err := catalog.RemoveCustomProvider(id); err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if err := catalog.RemoveAuth(id); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	usage.ForgetProvider(id)
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "removed": true})
}

func handleAccountActivate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	if err := catalog.ActivateAccount(id, aid); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "active": aid})
}

func handleAccountRename(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := catalog.RenameAccount(id, aid, req.Label); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "account": aid, "label": strings.TrimSpace(req.Label)})
}

func handleAccountDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	if err := catalog.RemoveAccount(id, aid); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	usage.Forget(id, aid)
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "removed": aid})
}

func handleProviderLogout(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := catalog.RemoveAuth(id); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		usage.ForgetProvider(id)
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "signedIn": false})
	}
}
