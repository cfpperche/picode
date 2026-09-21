package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/delivery"
	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/version"
)

// D2 publication observation (ADR-0170). The environment is the PiCode
// instance answering this read, configured explicitly on a workspace and
// bound to the repository key resolved when it was configured. Nothing here
// deploys, retries or orders anything: it records and compares facts.

// instanceIdentity is the running artifact read directly, without an HTTP
// round trip — the same two facts /api/health (bootId) and /api/version
// (revision) expose. Responding is true by construction: this process is
// serving the request that asks. It is a var so tests can name a revision.
var instanceIdentity = func() delivery.Running {
	return delivery.Running{Display: version.Build(), Revision: version.Revision(), Boot: BootID(), Responding: true}
}

// deliveryWorkspaceID finds the workspace a Git owner belongs to. "" means
// the owner has no workspace of its own (the free workspace): there is then
// nothing configured to read.
func deliveryWorkspaceID(deps Deps, kind, id string) string {
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

// deliveryBinding resolves the viewed owner's own binding, and never another
// workspace's. Four outcomes, in the order they can be established:
//
//   - the owner is bound to this repository: observed;
//   - the owner is bound to a *different* repository while another workspace
//     claims this one: a contradiction, reported as a conflict rather than
//     resolved by picking a side (ADR-0170);
//   - the owner is bound to a different repository: the folder moved or the
//     project was replaced, so the binding names another repository;
//   - nobody is bound: unconfigured. A sibling workspace's binding for this
//     repository is never substituted.
func deliveryBinding(deps Deps, workspaceID, repo string) (*delivery.Binding, bool, string) {
	if deps.Store == nil || workspaceID == "" || repo == "" {
		return nil, false, ""
	}
	own, bound, err := deps.Store.GetDeliveryObserver(workspaceID)
	if err != nil {
		return nil, false, "Environment configuration unavailable"
	}
	if bound && own.Repo == repo {
		return &delivery.Binding{WorkspaceID: own.WorkspaceID, Repo: own.Repo, Observer: own.Observer}, false, ""
	}
	rows, err := deps.Store.ListDeliveryObservers(repo)
	if err != nil {
		return nil, false, "Environment configuration unavailable"
	}
	claimed := false
	for _, row := range rows {
		if row.WorkspaceID != workspaceID {
			claimed = true
		}
	}
	if bound {
		// The binding travels either way: the read reports which repository
		// this project is connected to, and the contradiction if there is one.
		ownBinding := &delivery.Binding{WorkspaceID: own.WorkspaceID, Repo: own.Repo, Observer: own.Observer}
		return ownBinding, claimed, ""
	}
	return nil, false, ""
}

// deliveryEnvironment is the whole D2 half of a read: the binding, the
// instance's identity sampled around the evidence, and the publication state
// it implies for every change.
func deliveryEnvironment(deps Deps, r *http.Request, kind, id, cwd, repo, targetOID string, changes []delivery.Change) (delivery.Environment, []delivery.Change) {
	binding, conflict, issue := deliveryBinding(deps, deliveryWorkspaceID(deps, kind, id), repo)
	// The guard probe walks every agent and terminal (ADR-0078's interlocks),
	// so it runs only for a project that is actually connected: an
	// unconfigured project has no environment to report on.
	var busy *int
	if binding != nil && !conflict {
		if owners, complete := deployBusyCoverage(r.Context(), deps); complete {
			n := len(owners)
			busy = &n
		}
	}
	// The change scan has its own ten-second budget (Observe); this half is
	// bounded here, so a slow Git cannot hold the read open.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	env, marked := delivery.ObserveEnvironment(ctx, delivery.EnvironmentInput{
		DataDir: deps.DataDir, Cwd: cwd, Repo: repo, TargetOID: targetOID,
		Binding: binding, Conflict: conflict, Busy: busy, Sample: instanceIdentity,
	}, changes)
	if issue != "" {
		env.Issues = append(env.Issues, issue)
	}
	return env, marked
}

type deliveryObserverRequest struct {
	Observer string `json:"observer"`
}

// The environment selector's only write: bind the owner's workspace, or
// unbind it. Owner-scoped like the read route, so an agent or terminal tab
// configures the project it belongs to without the client knowing a workspace
// id. The repository key is resolved here, so a later folder move reads as a
// mismatch rather than silently observing the new folder (ADR-0170).
func registerDeliveryObserverRoutes(mux Registrar, deps Deps) {
	for _, route := range []struct{ ns, kind string }{{"workspaces", "workspace"}, {"agents", "agent"}, {"terminals", "term"}} {
		mux.HandleFunc("POST /api/"+route.ns+"/{id}/delivery/observer", func(w http.ResponseWriter, r *http.Request) {
			var req deliveryObserverRequest
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid observer request")
				return
			}
			if err := dec.Decode(&struct{}{}); err != io.EOF {
				writeErr(w, http.StatusBadRequest, "expected one JSON object")
				return
			}
			if req.Observer != "none" && req.Observer != "picode-self" {
				writeErr(w, http.StatusBadRequest, "observer must be picode-self or none")
				return
			}
			id := r.PathValue("id")
			cwd, ok := previewOwnerCwd(deps, w, r, route.kind, id)
			if !ok {
				return
			}
			workspace := deliveryWorkspaceID(deps, route.kind, id)
			if workspace == "" || workspace == store.FreeWorkspaceID {
				writeErr(w, http.StatusNotFound, "This owner has no project folder to connect.")
				return
			}
			repo := gitgraph.Key(cwd)
			if repo == "" {
				writeErr(w, http.StatusNotFound, "This folder is not a Git repository.")
				return
			}
			if err := deps.Store.SetDeliveryObserver(workspace, repo, req.Observer); err != nil {
				writeStoreErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"schemaVersion": 1, "observer": req.Observer, "repositoryKey": repo})
		})
	}
}
