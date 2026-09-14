package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/browser"
)

// The work-browser command channel (ADR-0132). The desktop shell's page opens
// the stream — the same authenticated session every other /api/* call uses, no
// new port and no new credential — and the daemon pushes one command per line.
// The shell re-checks the method against its tier catalog and the domains
// against navigation (ADR-0128), runs it, and posts the result back.
//
// GET  /api/browser/stream   event: hello | command
// POST /api/browser/result   {id, output|error} → 204, 404 when the command is gone
//
// Payloads are large by API standards (a CDP result can carry a screenshot or
// a page body), hence resultMaxBytes.
const resultMaxBytes = 32 << 20

func registerBrowserRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/browser/stream", handleBrowserStream(deps))
	mux.HandleFunc("POST /api/browser/result", handleBrowserResult(deps))
	mux.HandleFunc("POST /api/browser/tool", handleBrowserTool(deps))
}

// handleBrowserTool is what a Pi tool calls: a verb, not a CDP method. The
// daemon resolves the agent's policy (ADR-0134: read on the tab on screen
// unless a grant says more), refuses a verb the tier does not reach, and waits
// for the shell's answer through the hub.
func handleBrowserTool(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Browser == nil {
			writeErr(w, http.StatusServiceUnavailable, "browser channel is not running")
			return
		}
		var req struct {
			Agent  string          `json:"agent"`
			Verb   string          `json:"verb"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		verb, ok := browser.VerbFor(req.Verb)
		if !ok {
			names := make([]string, 0, 4)
			for name := range browser.Verbs() {
				names = append(names, name)
			}
			sort.Strings(names)
			writeErr(w, http.StatusBadRequest, "unknown verb "+strconv.Quote(req.Verb)+" — use "+strings.Join(names, ", "))
			return
		}
		var params map[string]any
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				writeErr(w, http.StatusBadRequest, "params must be an object: "+err.Error())
				return
			}
		}
		policy := browser.Resolve(deps.Store, req.Agent)
		if !policy.Allows(verb) {
			writeErr(w, http.StatusForbidden, fmt.Sprintf(
				"%s needs the %s tier; this agent has %s — grant it in Settings ▸ Browser",
				req.Verb, verb.Tier, policy.Tier))
			return
		}
		// A verb with a destination is checked here too: the shell's
		// navigation gate is the other half of the same rule (AllowsOrigin).
		if verb.NeedsURL && !policy.AllowsVerb(verb, params) {
			raw, _ := params["url"].(string)
			writeErr(w, http.StatusForbidden, fmt.Sprintf(
				"%s to %q is outside this agent's grant (domains: %s) — add it in Settings ▸ Browser",
				req.Verb, raw, strings.Join(policy.Domains, ", ")))
			return
		}
		output, err := deps.Browser.Dispatch(r.Context(), browser.Command{
			Method:  verb.Method,
			Params:  req.Params,
			Tier:    policy.Tier,
			Domains: policy.Domains,
		})
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"verb": req.Verb, "output": output})
	}
}

func handleBrowserStream(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Browser == nil {
			writeErr(w, http.StatusServiceUnavailable, "browser channel is not running")
			return
		}
		fl, ok := w.(http.Flusher)
		if !ok {
			writeErr(w, http.StatusInternalServerError, "streaming unsupported")
			return
		}
		events, detach := deps.Browser.Attach()
		defer detach()

		h := w.Header()
		h.Set("Content-Type", "text/event-stream; charset=utf-8")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		write := func(event string, data any) bool {
			body, err := json.Marshal(data)
			if err != nil {
				return true
			}
			if _, err := w.Write([]byte("event: " + event + "\ndata: " + string(body) + "\n\n")); err != nil {
				return false
			}
			fl.Flush()
			return true
		}
		// hello tells the shell it owns the line. A second window attaching
		// makes itself the active one; the older stream stays open and quiet.
		if !write("hello", map[string]any{"connected": true}) {
			return
		}
		heartbeat := time.NewTicker(eventsHeartbeat)
		defer heartbeat.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-heartbeat.C:
				if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
					return
				}
				fl.Flush()
			case cmd := <-events:
				if !write("command", cmd) {
					return
				}
			}
		}
	}
}

func handleBrowserResult(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Browser == nil {
			writeErr(w, http.StatusServiceUnavailable, "browser channel is not running")
			return
		}
		var res browser.Result
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, resultMaxBytes)).Decode(&res); err != nil {
			writeErr(w, http.StatusBadRequest, "bad result: "+err.Error())
			return
		}
		if res.ID == "" {
			writeErr(w, http.StatusBadRequest, "result needs an id")
			return
		}
		switch err := deps.Browser.Complete(res); {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, browser.ErrUnknownResult):
			writeErr(w, http.StatusNotFound, "no such command")
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
	}
}
