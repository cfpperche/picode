package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/modellist"
)

// Load models for a custom endpoint (the open topic's P1): the server asks the
// endpoint what it serves so nobody has to copy ids by hand. The key never
// travels back to the browser — the request may carry it once (the same key the
// form would save), or the server reads the saved one from pi's auth.json.
// This is a listing call only: PiCode never proxies model traffic (ADR-0003,
// ADR-0129).
func handleCustomProviderModels(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL string `json:"baseUrl"`
		API     string `json:"api"`
		Key     string `json:"key"`
		ID      string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	baseURL := strings.TrimSpace(req.BaseURL)
	key := strings.TrimSpace(req.Key)
	if key == "" && strings.TrimSpace(req.ID) != "" {
		// Editing: the stored credential is what the endpoint expects.
		if saved, ok := catalog.ActiveAPIKey(strings.TrimSpace(req.ID)); ok {
			key = saved
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	res, err := modellist.List(ctx, baseURL, strings.TrimSpace(req.API), key)
	if err != nil {
		var typed *modellist.Error
		if !errors.As(err, &typed) {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": "Could not read the endpoint's model list.", "kind": "upstream",
			})
			return
		}
		status, body := listFailure(typed, key != "")
		writeJSON(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"models": res.Models,
		"url":    res.URL,
		"count":  len(res.Models),
	})
}

// listFailure turns a classified error into one line a person can act on,
// plus the kind the dialog styles by. The copy lives here, not in the browser:
// both apps show the same words, and the tests pin them.
func listFailure(e *modellist.Error, keyed bool) (int, map[string]any) {
	host := e.Host
	if host == "" {
		host = "the endpoint"
	}
	switch e.Kind {
	case modellist.KindInput:
		return http.StatusBadRequest, map[string]any{"error": capitalize(e.Detail) + ".", "kind": "input"}
	case modellist.KindAuth:
		lead, fix := "The endpoint refused the key", "Check the API key and try again."
		if !keyed {
			lead, fix = "The endpoint wants a key", "Type the API key above and try again."
		}
		return http.StatusBadGateway, map[string]any{
			"error": lead + " (" + itoa(e.Status) + ")" + detailTail(e.Detail) + ". " + fix,
			"kind":  "auth",
		}
	case modellist.KindBilling:
		return http.StatusBadGateway, map[string]any{
			"error": "The account cannot list models (" + itoa(e.Status) + ")" + detailTail(e.Detail) +
				". Its billing or quota needs attention first.",
			"kind": "auth",
		}
	case modellist.KindMissing:
		return http.StatusBadGateway, map[string]any{
			"error": "No model list at that address (" + itoa(e.Status) + "). " +
				"Some gateways answer one level down — try the base URL with /v1, or type the ids by hand.",
			"kind": "missing",
		}
	case modellist.KindTransport:
		return http.StatusBadGateway, map[string]any{
			"error": "Could not reach " + host + ": " + e.Detail + ".",
			"kind":  "transport",
		}
	}
	return http.StatusBadGateway, map[string]any{
		"error": "The endpoint answered " + itoa(e.Status) + detailTail(e.Detail) + ".",
		"kind":  "upstream",
	}
}

// detailTail adds the provider's own words when it gave any, so "invalid api
// key" reaches the person instead of a bare status.
func detailTail(detail string) string {
	if detail == "" || detail == "no details" || detail == "no model list" {
		return ""
	}
	return ": " + detail
}

func itoa(n int) string {
	if n == 0 {
		return "no answer"
	}
	return strconv.Itoa(n)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
