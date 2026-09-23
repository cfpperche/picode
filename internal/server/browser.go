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
	mux.HandleFunc("GET /api/browser/developer/audit", handleBrowserDeveloperAudit(deps))
	registerBrowserPolicyRoutes(mux, deps)
	registerBrowserHistoryRoutes(mux, deps)
	registerBrowserAnnotationRoutes(mux, deps)
	registerBrowserDownloadRoutes(mux, deps)
	registerBrowserPermissionRoutes(mux, deps)
	registerBrowserPrefRoutes(mux, deps)
}

// handleBrowserTool is what a Pi tool calls: a verb, not a CDP method. An
// identified caller drives the split bound to its own session (ADR-0172) —
// open, navigate, click, type — without a stored grant. An unidentified
// caller stays read-only on the tab on screen (ADR-0134). Raw CDP still
// needs the full tier and Developer mode.
func handleBrowserTool(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Browser == nil {
			writeErr(w, http.StatusServiceUnavailable, "browser channel is not running")
			return
		}
		var req struct {
			Agent  string          `json:"agent"`
			Term   string          `json:"term,omitempty"`
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
		if !browserAgentAccess(deps.Store) {
			writeErr(w, http.StatusForbidden, "agents may not use the built-in browser right now — turn it back on in Settings ▸ Browser")
			return
		}
		// ADR-0146: the history is its own permission (never by default), not a
		// side effect of a tier — and the daemon answers it from the store, so a
		// read works with no browser tab open and no shell running.
		if req.Verb == "history" {
			prefs, perr := browserPrefsRead(deps.Store)
			if perr != nil {
				writeErr(w, http.StatusInternalServerError, perr.Error())
				return
			}
			if prefs.HistoryAccess != "allow" {
				writeErr(w, http.StatusForbidden, "reading the browsing history is off — turn on Agent history access in Settings ▸ Browser")
				return
			}
			limit := 100
			if n, ok := params["limit"].(float64); ok && n > 0 {
				limit = int(n)
			}
			if limit > 200 {
				limit = 200
			}
			query, _ := params["query"].(string)
			visits, herr := deps.Store.ListBrowserHistory(limit, query)
			if herr != nil {
				writeErr(w, http.StatusInternalServerError, herr.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"verb": "history", "output": map[string]any{"visits": visits, "query": query}})
			return
		}
		// ADR-0143: the caller is a managed agent, a terminal, or neither.
		// ADR-0172: an identified caller driving its own split is act on any
		// http(s) URL. The stored grant is not that gate. Raw CDP still reads
		// the stored tier, so this raise happens only for a session drive.
		policy := browser.ResolveCaller(deps.Store, callerAgentID(deps, req.Agent, req.Term), req.Term)
		key := callerKey(deps, req.Agent, req.Term)
		session := key != "" && browser.IsSessionDrive(req.Verb)
		if session {
			policy = browser.SessionDrive()
		}
		// ADR-0144: the raw verb is the one shape whose method the catalog did
		// not name. Machine opt-in plus the full tier, and either refusal is
		// recorded — the audit is the point of letting it exist at all.
		method := verb.Method
		cdpParams := req.Params
		if verb.Raw {
			devMode := browserDeveloperMode(deps.Store)
			if !devMode {
				browserAuditRaw(deps, callerAgentID(deps, req.Agent, req.Term), req.Term, params, "refused", "developer mode is off")
				writeErr(w, http.StatusForbidden, "raw CDP is off — turn on Developer mode in Settings ▸ Browser (Elevated risk)")
				return
			}
			if !policy.AllowsRaw(devMode) {
				browserAuditRaw(deps, callerAgentID(deps, req.Agent, req.Term), req.Term, params, "refused", "tier is "+policy.Tier)
				writeErr(w, http.StatusForbidden, fmt.Sprintf(
					"raw CDP needs the full tier; this agent has %s — grant it in Settings ▸ Browser",
					policy.Tier))
				return
			}
			rawMethod, err := browser.RawMethod(params)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			method = rawMethod
			// {method, params}: the protocol's own parameters are nested so the
			// verb's bookkeeping can never leak into the call the page runs.
			cdpParams = json.RawMessage("{}")
			if raw, ok := params["params"]; ok {
				body, err := json.Marshal(raw)
				if err != nil {
					writeErr(w, http.StatusBadRequest, "cdp params must be JSON: "+err.Error())
					return
				}
				cdpParams = body
			}
		}
		if !verb.Raw && !policy.Allows(verb) {
			if browser.IsSessionDrive(req.Verb) && key == "" {
				writeErr(w, http.StatusForbidden, req.Verb+" needs a session in PiCode — this caller has no identity, so it cannot drive a browser tab")
				return
			}
			writeErr(w, http.StatusForbidden, fmt.Sprintf(
				"%s needs the %s tier; this agent has %s — grant it in Settings ▸ Browser",
				req.Verb, verb.Tier, policy.Tier))
			return
		}
		// A verb with a destination is checked here too: the shell's
		// navigation gate is the other half of the same rule (AllowsOrigin).
		// A session drive's domains are "*", which is any http(s) host and
		// still refuses file: and javascript:.
		if verb.NeedsURL && !policy.AllowsVerb(verb, params) {
			raw, _ := params["url"].(string)
			writeErr(w, http.StatusForbidden, fmt.Sprintf(
				"%s to %q is outside this agent's grant (domains: %s) — add it in Settings ▸ Browser",
				req.Verb, raw, strings.Join(policy.Domains, ", ")))
			return
		}
		if !verb.Raw {
			prepared, body, err := browser.Prepare(req.Verb, params)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			method = prepared
			cdpParams = body
		}
		output, err := deps.Browser.Dispatch(r.Context(), browser.Command{
			Method:    method,
			Params:    cdpParams,
			Tier:      policy.Tier,
			Domains:   policy.Domains,
			Raw:       verb.Raw,
			Principal: key,
			Agent:     strings.TrimSpace(req.Agent),
			Term:      strings.TrimSpace(req.Term),
			Session:   session,
		})
		if err != nil {
			if verb.Raw {
				browserAuditRaw(deps, callerAgentID(deps, req.Agent, req.Term), req.Term, map[string]any{"method": method}, "failed", err.Error())
			}
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if verb.Raw {
			browserAuditRaw(deps, callerAgentID(deps, req.Agent, req.Term), req.Term, map[string]any{"method": method}, "allowed", "")
		}
		writeJSON(w, http.StatusOK, map[string]any{"verb": req.Verb, "output": output})
	}
}

// browserAuditRaw records one raw CDP call (ADR-0144): who asked, which
// method, and the outcome — allowed, refused, or failed. The event is the
// record the settings card and the agent's conversation are read against;
// a write failure must not turn into an unrecorded call passing silently,
// so it is announced on the feed like any other event (ADR-0048).
func browserAuditRaw(deps Deps, agentID, termID string, params map[string]any, outcome, reason string) {
	if deps.Store == nil {
		return
	}
	method, _ := params["method"].(string)
	var agent *string
	if strings.TrimSpace(agentID) != "" {
		agent = &agentID
	}
	data := map[string]any{"method": method, "outcome": outcome}
	if strings.TrimSpace(termID) != "" {
		data["termId"] = termID
	}
	if reason != "" {
		data["reason"] = reason
	}
	// The caller's own words are not the only thing worth knowing when this
	// goes wrong; the method is the record. A failure here is logged by the
	// store and never blocks the call the owner asked for.
	_ = deps.Store.AppendEvent("browser.cdp", agent, nil, data)
}

// handleBrowserDeveloperAudit is the settings card's reader (ADR-0144): the
// newest raw CDP calls, allowed or refused, so "audited" is something the
// owner can look at rather than a promise in a document.
func handleBrowserDeveloperAudit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		rows, err := deps.Store.EventsOfType("browser.cdp", limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		type call struct {
			ID       int64  `json:"id"`
			Method   string `json:"method"`
			Outcome  string `json:"outcome"`
			Reason   string `json:"reason,omitempty"`
			AgentID  string `json:"agentId,omitempty"`
			TermID   string `json:"termId,omitempty"`
			CalledAt string `json:"calledAt"`
		}
		calls := make([]call, 0, len(rows))
		for _, ev := range rows {
			var data struct {
				Method  string `json:"method"`
				Outcome string `json:"outcome"`
				Reason  string `json:"reason"`
				TermID  string `json:"termId"`
			}
			_ = json.Unmarshal(ev.Data, &data)
			entry := call{ID: ev.ID, Method: data.Method, Outcome: data.Outcome, Reason: data.Reason, TermID: data.TermID, CalledAt: ev.CreatedAt}
			if ev.AgentID != nil {
				entry.AgentID = *ev.AgentID
			}
			calls = append(calls, entry)
		}
		writeJSON(w, http.StatusOK, map[string]any{"calls": calls, "developerMode": browserDeveloperMode(deps.Store)})
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
