package server

import (
	"context"
	"net/http"
	"time"
)

// Deploy readiness (ADR-0086). A restart ends every CLI and agent pane the
// daemon manages (ADR-0084/0085), and deploys used to be one per merged
// branch, tens per day, each one paid by whoever else was mid-turn. The
// route answers the one question `picode deploy` must ask first: is anyone
// working right now? It reuses the Git-actions interlock probes (ADR-0078)
// over every agent and terminal, not one repository. Loopback callers need
// no session (internal/auth exempts it there): the deploy runs on this
// machine, and the answer names terminals, never content.
func registerDeployRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/deploy/readiness", handleDeployReadiness(deps))
}

type deployReadiness struct {
	Ready bool        `json:"ready"`
	Busy  []busyOwner `json:"busy"`
	At    string      `json:"at"`
}

func deployBusy(ctx context.Context, deps Deps) []busyOwner {
	out := []busyOwner{}
	if deps.Store == nil {
		return out
	}
	if agents, err := deps.Store.ListAllAgents(); err == nil {
		for _, a := range agents {
			if busy, why := agentBusyFn(ctx, deps, a); busy {
				out = append(out, busyOwner{Kind: "agent", ID: a.ID, Name: a.Name, Why: why})
			}
		}
	}
	if terms, err := deps.Store.ListTerminals(); err == nil {
		for _, t := range terms {
			if busy, why := terminalBusyFn(ctx, deps, t); busy {
				out = append(out, busyOwner{Kind: "terminal", ID: t.ID, Name: t.Name, Why: why})
			}
		}
	}
	return out
}

func handleDeployReadiness(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		busy := deployBusy(ctx, deps)
		writeJSON(w, http.StatusOK, deployReadiness{
			Ready: len(busy) == 0,
			Busy:  busy,
			At:    time.Now().UTC().Format(time.RFC3339),
		})
	}
}
