package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
)

// The act watch registry is per-process state: which agents have a turn
// in flight that may end in a picode-act block. The panel polls while
// this is true and stops when it flips false.
var (
	actMu       sync.Mutex
	actWatchers = map[string]int{}
)

func addActWatch(agentID string) {
	actMu.Lock()
	actWatchers[agentID]++
	actMu.Unlock()
}

func releaseActWatch(agentID string) {
	actMu.Lock()
	if n := actWatchers[agentID]; n <= 1 {
		delete(actWatchers, agentID)
	} else {
		actWatchers[agentID] = n - 1
	}
	actMu.Unlock()
}

func watchingAct(agentID string) bool {
	actMu.Lock()
	defer actMu.Unlock()
	return actWatchers[agentID] > 0
}

func registerActRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/extension/act/next", handleActNext(deps))
	mux.HandleFunc("POST /api/extension/act/{id}/result", handleActResult(deps))
}

// startActWatch arms the settle observer for one act-enabled send. It
// refuses when another run already observes the agent (the automations'
// observer owns the settle event).
func startActWatch(deps Deps, agentID, origin string) error {
	ma := deps.Runtime.Get(agentID)
	if ma == nil {
		return errors.New("agent is not running")
	}
	if ma.Observed() {
		return errors.New("the agent is busy with another run")
	}
	addActWatch(agentID)
	w := actWatch{deps: deps, agentID: agentID, origin: origin}
	ma.Observe(&rpc.RunObserver{OnSettled: w.settled, OnExit: func(bool) { w.exited() }})
	return nil
}

// actWatch follows one turn that may end in a picode-act block. The
// observer is one-shot: settle or exit clears it, so the person's own
// later turns are never intercepted.
type actWatch struct {
	deps    Deps
	agentID string
	origin  string
}

func (w actWatch) settled(final string) {
	ma := w.deps.Runtime.Get(w.agentID)
	if ma != nil {
		ma.Observe(nil)
	}
	releaseActWatch(w.agentID)
	batch, ok, err := browserhost.ParseActBlock(final)
	if err != nil || !ok {
		return // plain answer — the loop ends; nothing to execute
	}
	raw, err := json.Marshal(batch.Actions)
	if err != nil {
		return
	}
	_, _ = w.deps.Store.CreateActBatch(w.agentID, w.origin, string(raw), 1)
}

func (w actWatch) exited() {
	ma := w.deps.Runtime.Get(w.agentID)
	if ma != nil {
		ma.Observe(nil)
	}
	releaseActWatch(w.agentID)
}

// handleActNext claims the agent's next batch for the asking tab. The
// tab's origin must match the batch's — a batch for another tab waits
// without being claimed (blocked=origin).
func handleActNext(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.URL.Query().Get("agent")
		if strings.TrimSpace(agentID) == "" {
			writeErr(w, http.StatusBadRequest, "agent is required")
			return
		}
		if _, err := deps.Store.GetAgent(agentID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		b, ok, err := deps.Store.PendingActBatch(agentID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"watching": watchingAct(agentID)})
			return
		}
		tabOrigin := r.URL.Query().Get("tab")
		if b.State == store.ActPending && sameOrigin(b.Origin, tabOrigin) {
			if b, _, err = deps.Store.ClaimActBatch(agentID); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if b.State == store.ActPending {
			writeJSON(w, http.StatusOK, map[string]any{"watching": true, "blocked": "origin"})
			return
		}
		var acts []browserhost.Act
		if err := json.Unmarshal([]byte(b.Actions), &acts); err != nil {
			_ = deps.Store.FinishActBatch(b.ID)
			writeJSON(w, http.StatusOK, map[string]any{"watching": watchingAct(agentID)})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"watching": true,
			"batch": browserhost.ActBatchWire{
				ID: b.ID, AgentID: b.AgentID, Origin: b.Origin,
				Round: b.Round, Rounds: browserhost.ActMaxRounds, Actions: acts,
			},
		})
	}
}

// handleActResult records outcomes and, unless stopped or out of rounds,
// feeds them back to the agent as one more watched turn.
func handleActResult(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var req struct {
			Outcomes []browserhost.ActOutcome `json:"outcomes"`
			Stopped  bool                     `json:"stopped"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		b, ok, err := deps.Store.GetActBatchRow(id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, "batch not found")
			return
		}
		if err := deps.Store.FinishActBatch(b.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if req.Stopped || b.Round >= browserhost.ActMaxRounds {
			releaseActWatch(b.AgentID)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "watching": false})
			return
		}
		ma := deps.Runtime.Get(b.AgentID)
		if ma == nil {
			releaseActWatch(b.AgentID)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "watching": false})
			return
		}
		addActWatch(b.AgentID)
		aw := actWatch{deps: deps, agentID: b.AgentID, origin: b.Origin}
		ma.Observe(&rpc.RunObserver{OnSettled: aw.settled, OnExit: func(bool) { aw.exited() }})
		if err := ma.SendTurn(store.TaskPrompt,
			browserhost.ComposeActResult(b.Round, req.Outcomes, false), nil); err != nil {
			ma.Observe(nil)
			releaseActWatch(b.AgentID)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "watching": false})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "watching": true})
	}
}

// sameOrigin compares two URL origins ("https://x.com") case-insensitively
// on host+scheme; an empty tab origin never matches.
func sameOrigin(a, b string) bool {
	if strings.TrimSpace(b) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimRight(a, "/"), strings.TrimRight(b, "/"))
}
