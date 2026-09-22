package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/cfpperche/picode/internal/delivery"
	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
)

// The integration queue's owner doors (ADR-0182). The queue is *read* inside
// the Delivery read every surface already fetches; the owner acts on it here.
// The agent's own door (request/withdraw) rides the delivery tool contract,
// which the store already serves.
//
// Ordering and authorizing are the owner's alone; the store enforces that with
// store.OwnerActor, so a route that forgot to check cannot grant them.

func registerDeliveryQueueRoutes(mux Registrar, deps Deps) {
	// The queue's recovery runs from the daemon's own start path
	// (ResumeDeliveryQueue), never from mounting these routes: a process that
	// only builds a handler — a test — has no queue to report on.
	runs := newQueueRuns(deps)
	for _, route := range []struct{ ns, kind string }{{"workspaces", "workspace"}, {"agents", "agent"}, {"terminals", "term"}} {
		mux.HandleFunc("POST /api/"+route.ns+"/{id}/delivery/queue", func(w http.ResponseWriter, r *http.Request) {
			cwd, repo, ok := ownerDeliveryRepo(deps, w, r, route.kind, r.PathValue("id"))
			if !ok {
				return
			}
			var m store.QueueMutation
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&m); err != nil {
				writeErr(w, 400, "invalid queue request")
				return
			}
			if err := dec.Decode(&struct{}{}); err != io.EOF {
				writeErr(w, 400, "expected one JSON object")
				return
			}
			if err := store.ValidateQueueMutation(m); err != nil {
				writeErr(w, 400, err.Error())
				return
			}
			entry, err := deps.Store.ApplyQueueMutation(repo, store.OwnerActor, m)
			if err != nil {
				writeQueueErr(w, err)
				return
			}
			// Authorizing is execution authority (ADR-0182): the entry the owner
			// just authorized — and anything authorized behind it — runs now.
			if entry.State == store.QueueAuthorized {
				runs.launch(cwd, repo, ownerWorkspaceID(deps, route.kind, r.PathValue("id")))
			}
			writeJSON(w, 200, map[string]any{"schemaVersion": 1, "entry": entry})
		})
	}
	mux.HandleFunc("GET /api/delivery/integration", handleGetIntegrationSettings(deps))
	mux.HandleFunc("PUT /api/delivery/integration", handlePutIntegrationSettings(deps))
	mux.HandleFunc("DELETE /api/delivery/integration", handleDeleteIntegrationSettings(deps))
}

// deliveryQueueRead is the Delivery read with the queue layered on (ADR-0182):
// the Git snapshot stays the observer's, the queue and its declaration are
// PiCode state read fresh, and the declaration is already resolved through the
// workspace-first fallback the executor will use.
type deliveryQueueRead struct {
	delivery.Snapshot
	Queue       []store.QueueEntry        `json:"queue"`
	Integration store.IntegrationSettings `json:"integration"`
}

func queueLayer(deps Deps, kind, id, repo string, view delivery.Snapshot) (deliveryQueueRead, error) {
	entries, err := deps.Store.ListQueueEntries(repo, 0)
	if err != nil {
		return deliveryQueueRead{}, err
	}
	settings, err := deps.Store.EffectiveIntegrationSettings(ownerWorkspaceID(deps, kind, id))
	if err != nil {
		return deliveryQueueRead{}, err
	}
	return deliveryQueueRead{Snapshot: view, Queue: entries, Integration: settings}, nil
}

// ownerDeliveryRepo resolves an owner (workspace, agent or terminal) to the
// repository its queue belongs to — and to the folder a run works in — or
// answers the request itself.
func ownerDeliveryRepo(deps Deps, w http.ResponseWriter, r *http.Request, kind, id string) (string, string, bool) {
	cwd, ok := previewOwnerCwd(deps, w, r, kind, id)
	if !ok {
		return "", "", false
	}
	cwd = canonDir(cwd)
	if !checkFileRoot(w, r, cwd) {
		return "", "", false
	}
	repo := gitgraph.Key(cwd)
	if repo == "" {
		writeErr(w, 404, "This folder is not a Git repository.")
		return "", "", false
	}
	return cwd, repo, true
}

// ownerWorkspaceID names the workspace an owner belongs to, for the
// workspace-first declaration fallback.
func ownerWorkspaceID(deps Deps, kind, id string) string {
	switch kind {
	case "workspace":
		return id
	case "agent":
		if a, err := deps.Store.GetAgent(id); err == nil {
			return a.WorkspaceID
		}
	case "term":
		if t, err := deps.Store.GetTerminal(id); err == nil {
			return t.WorkspaceID
		}
	}
	return ""
}

// writeQueueErr keeps the queue's refusals honest: a missing entry is a 404, and
// everything else the store refuses — a stale version, capacity, a state this
// action cannot move — is a 409 carrying the store's own words.
func writeQueueErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, 404, "that queue entry is gone")
		return
	}
	writeErr(w, 409, err.Error())
}

func handleGetIntegrationSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("workspace")
		if len(scope) > 128 {
			writeErr(w, 400, "Invalid workspace")
			return
		}
		declared, ok, err := deps.Store.IntegrationSettingsFor(scope)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		effective, err := deps.Store.EffectiveIntegrationSettings(scope)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"schemaVersion": 1, "workspace": scope, "declared": ok, "settings": declared, "effective": effective})
	}
}

func handlePutIntegrationSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("workspace")
		if scope != "" {
			if _, err := deps.Store.GetWorkspace(scope); err != nil {
				writeErr(w, 404, "workspace not found")
				return
			}
		}
		var m store.IntegrationSettingsMutation
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			writeErr(w, 400, "invalid integration settings")
			return
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			writeErr(w, 400, "expected one JSON object")
			return
		}
		if err := store.ValidateIntegrationSettings(m); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		settings, err := deps.Store.PutIntegrationSettings(scope, m)
		if errors.Is(err, store.ErrIntegrationConflict) {
			writeErr(w, 409, "These settings changed since you opened them. Reopen to see the current ones.")
			return
		}
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"schemaVersion": 1, "settings": settings})
	}
}

// handleDeleteIntegrationSettings drops a workspace's own declaration so it
// inherits the machine's again. It names a workspace: the machine layer is
// written, never deleted through here, so no request can wipe the fallback.
func handleDeleteIntegrationSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("workspace")
		if scope == "" {
			writeErr(w, 400, "name the workspace whose declaration to drop")
			return
		}
		if _, err := deps.Store.GetWorkspace(scope); err != nil {
			writeErr(w, 404, "workspace not found")
			return
		}
		if err := deps.Store.DeleteIntegrationSettings(scope); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
