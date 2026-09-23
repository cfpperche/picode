package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// Exit records (ADR-0194): a person's removal of an agent writes one, the
// catalog lists and counts them, and the person can label, relabel, delete
// or export them. Nothing here leaves the machine.

func registerAgentExitRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/agent-exits", handleListAgentExits(deps))
	mux.HandleFunc("GET /api/agent-exits/summary", handleAgentExitSummary(deps))
	mux.HandleFunc("GET /api/agent-exits/export", handleExportAgentExits(deps))
	mux.HandleFunc("GET /api/agent-exits/prefs", handleAgentExitPrefs(deps))
	mux.HandleFunc("PUT /api/agent-exits/prefs", handleSetAgentExitPrefs(deps))
	mux.HandleFunc("GET /api/agent-exits/{id}", handleGetAgentExit(deps))
	mux.HandleFunc("PATCH /api/agent-exits/{id}", handleLabelAgentExit(deps))
	mux.HandleFunc("DELETE /api/agent-exits/{id}", handleDeleteAgentExit(deps))
	mux.HandleFunc("POST /api/agent-exits/{id}/undo", handleUndoAgentExit(deps))
}

// exitBody is the optional body of DELETE /api/agents/{id}: which face
// removed the agent, whether it showed the question, and the answer.
type exitBody struct {
	Origin  string   `json:"origin"`
	Asked   bool     `json:"asked"`
	Outcome string   `json:"outcome"`
	Reasons []string `json:"reasons"`
	Note    string   `json:"note"`
}

type exitRequest struct {
	Exit *exitBody `json:"exit"`
}

const exitBodyCap = 16 << 10

// readExitRequest reads the optional removal body; an empty body is a
// removal without an answer (scripts, older clients).
func readExitRequest(w http.ResponseWriter, r *http.Request) (exitRequest, error) {
	var req exitRequest
	if r.Body == nil || r.Body == http.NoBody {
		return req, nil
	}
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, exitBodyCap)).Decode(&req)
	if errors.Is(err, io.EOF) {
		return exitRequest{}, nil
	}
	if err != nil {
		return exitRequest{}, fmt.Errorf("invalid request body")
	}
	return req, nil
}

// exitOriginOf is the face a removal body names ("" = the API).
func exitOriginOf(req exitRequest) string {
	if req.Exit == nil {
		return ""
	}
	return req.Exit.Origin
}

// endTerminalAgent writes the exit of the agent a terminal removal ends: a
// CLI agent's TUI is its terminal (ADR-0160), so removing the terminal is
// removing the agent, and the person decided it (ADR-0194). The caller
// deletes the terminal itself afterwards; a plain shell has no agent.
func (deps Deps) endTerminalAgent(termID string, req exitRequest) error {
	a, err := deps.Store.AgentByTerminal(termID)
	if err != nil {
		return nil
	}
	in := deps.exitInput(a, req, false, false, time.Now())
	in.LeaveTerminal = true
	if _, err := deps.Store.RemoveAgentWithExit(a.ID, in); err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	return nil
}

// exitPreview is the exit block of GET /api/agents/{id}/cleanup: whether
// the removal dialog asks, why not, and the words it shows.
type exitPreview struct {
	Ask      bool               `json:"ask"`
	Skip     string             `json:"skip,omitempty"`
	Taxonomy store.ExitTaxonomy `json:"taxonomy"`
}

// exitAsk applies ADR-0194's decision table to this agent now. A store
// that cannot answer asks: a missed answer costs more than a question.
func (deps Deps) exitAsk(agent store.Agent, now time.Time) (bool, string) {
	on, err := deps.Store.ExitAskOn()
	if err != nil {
		on = true
	}
	act, err := deps.Store.AgentActivityOf(agent.ID)
	if err != nil {
		act = store.AgentActivity{}
	}
	return store.ExitAskDecision(on, act, agent.CreatedAt, now)
}

func (deps Deps) exitPreviewFor(agent store.Agent, now time.Time) *exitPreview {
	ask, skip := deps.exitAsk(agent, now)
	return &exitPreview{Ask: ask, Skip: skip, Taxonomy: store.ExitTaxonomyV1()}
}

// exitInput turns the removal request into what the store records. When
// the client did not show the question, the skip says why: the server's
// own reason, or "client" when the server would have asked.
func (deps Deps) exitInput(agent store.Agent, req exitRequest, purgeSessions, purgeWork bool, now time.Time) store.ExitInput {
	in := store.ExitInput{Origin: store.ExitFromAPI, SessionsPurged: purgeSessions, WorkPurged: purgeWork}
	if b := req.Exit; b != nil {
		in.Origin = b.Origin
		in.Asked = b.Asked
		in.Label = store.ExitLabel{Outcome: b.Outcome, Reasons: b.Reasons, Note: b.Note}
	}
	if !in.Asked {
		if ask, skip := deps.exitAsk(agent, now); ask {
			in.AskSkip = store.ExitSkipClient
		} else {
			in.AskSkip = skip
		}
	}
	return in
}

// exitSince is the lower bound of a dashboard range ("" = all time).
func exitSince(raw string, now time.Time) string {
	if strings.TrimSpace(raw) == "" || raw == "all" {
		return ""
	}
	from, _, _ := statsWindow(normalizeRange(raw), now, time.Local)
	if from.IsZero() {
		return ""
	}
	return from.UTC().Format(time.RFC3339Nano)
}

func exitFilterOf(r *http.Request, now time.Time) store.ExitFilter {
	q := r.URL.Query()
	f := store.ExitFilter{
		WorkspaceID: strings.TrimSpace(q.Get("workspace")),
		CLI:         strings.TrimSpace(q.Get("cli")),
		Outcome:     strings.TrimSpace(q.Get("outcome")),
		Since:       exitSince(q.Get("range"), now),
		Before:      strings.TrimSpace(q.Get("before")),
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil {
		f.Limit = n
	}
	return f
}

func handleListAgentExits(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := exitFilterOf(r, time.Now())
		if f.Limit <= 0 || f.Limit > 200 {
			f.Limit = 50
		}
		exits, err := deps.Store.ListAgentExits(f)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		ask, err := deps.Store.ExitAskOn()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		next := ""
		if len(exits) == f.Limit {
			next = exits[len(exits)-1].RemovedAt
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"exits": exits, "next": next, "ask": ask, "taxonomy": store.ExitTaxonomyV1(),
		})
	}
}

func handleAgentExitSummary(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := exitFilterOf(r, time.Now())
		sum, err := deps.Store.AgentExitSummary(f)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		rng := strings.TrimSpace(r.URL.Query().Get("range"))
		if rng == "" {
			rng = "all"
		} else {
			rng = normalizeRange(rng)
		}
		writeJSON(w, http.StatusOK, map[string]any{"range": rng, "summary": sum, "taxonomy": store.ExitTaxonomyV1()})
	}
}

// handleExportAgentExits hands the person the whole catalog, undone exits
// included, as JSON lines.
func handleExportAgentExits(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", `attachment; filename="picode-agent-exits-`+time.Now().Format("20060102")+`.jsonl"`)
		enc := json.NewEncoder(w)
		if err := deps.Store.EachAgentExit(func(ex store.AgentExit) error { return enc.Encode(ex) }); err != nil {
			// Headers are gone once a line is written; a failure before
			// the first one still reaches the client as an error.
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
	}
}

func handleAgentExitPrefs(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ask, err := deps.Store.ExitAskOn()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ask": ask})
	}
}

func handleSetAgentExitPrefs(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Ask *bool `json:"ask"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil || req.Ask == nil {
			writeErr(w, http.StatusBadRequest, "ask (true or false) is required")
			return
		}
		if err := deps.Store.SetExitAskOn(*req.Ask); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ask": *req.Ask})
	}
}

func writeExitErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "exit not found")
	case errors.Is(err, store.ErrInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func handleGetAgentExit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ex, err := deps.Store.GetAgentExit(r.PathValue("id"))
		if err != nil {
			writeExitErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ex)
	}
}

func handleLabelAgentExit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req store.ExitLabel
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, exitBodyCap)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		ex, err := deps.Store.LabelAgentExit(r.PathValue("id"), req)
		if err != nil {
			writeExitErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ex)
	}
}

func handleDeleteAgentExit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteAgentExit(r.PathValue("id")); err != nil {
			writeExitErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleUndoAgentExit links the exit to the agent an Undo brought back; the
// exit stays for the record and leaves every count.
func handleUndoAgentExit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AgentID string `json:"agentId"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		ex, err := deps.Store.MarkAgentExitUndone(r.PathValue("id"), req.AgentID)
		if err != nil {
			writeExitErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ex)
	}
}
