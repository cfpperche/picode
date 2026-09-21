package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/delivery"
	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
)

// A vendor-independent declaration door, under the normal daemon auth gate.
// It never invokes a project script, merges, validates tests, or deploys.
func registerDeliveryRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/delivery/tool", handleDeliveryTool(deps))
}

type deliveryToolRequest struct {
	Agent string `json:"agent,omitempty"`
	Term  string `json:"term,omitempty"`
	store.DeliveryMutation
	Before int64 `json:"before,omitempty"`
}

func deliveryGit(ctx context.Context, cwd string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func handleDeliveryTool(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req deliveryToolRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeErr(w, 400, "invalid delivery request")
			return
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			writeErr(w, 400, "expected one JSON object")
			return
		}
		// Identity is an existing launch principal, not a conversation credential.
		var cwd, principal string
		if req.Agent != "" {
			a, err := deps.Store.GetAgent(req.Agent)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			if req.Term != "" && (a.TerminalID == nil || *a.TerminalID != req.Term) {
				writeErr(w, 403, "agent and terminal identities disagree")
				return
			}
			cwd, err = agentCwd(deps, req.Agent)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			principal = a.ID
		} else if req.Term != "" {
			t, err := deps.Store.GetTerminal(req.Term)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			cwd = t.Cwd
			principal = "term:" + t.ID
			if a, err := deps.Store.AgentByTerminal(t.ID); err == nil {
				principal = a.ID
				cwd, err = agentCwd(deps, a.ID)
				if err != nil {
					writeStoreErr(w, err)
					return
				}
			}
		} else {
			writeErr(w, 403, "no identity: open this CLI through PiCode")
			return
		}
		repo := gitgraph.Key(cwd)
		if repo == "" {
			writeErr(w, 409, "the launch folder is not an available Git repository")
			return
		}
		if req.Action == "capabilities" {
			writeJSON(w, 200, map[string]any{"schemaVersion": 1, "principal": principal, "identityScope": "launch", "actions": []string{"register", "update", "request-review", "withdraw-review", "show", "list"}, "integrationQueue": false, "deployment": false})
			return
		}
		if req.Action == "show" {
			d, err := deps.Store.GetDelivery(repo, req.ID)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			writeJSON(w, 200, deliveryView(r.Context(), cwd, d))
			return
		}
		if req.Action == "list" {
			ds, err := deps.Store.ListDeliveries(repo, req.Before)
			if err != nil {
				writeErr(w, 400, err.Error())
				return
			}
			more := len(ds) > 100
			if more {
				ds = ds[:100]
			}
			var next int64
			if more {
				next = ds[len(ds)-1].Sequence
			}
			writeJSON(w, 200, map[string]any{"schemaVersion": 1, "deliveries": ds, "nextBefore": next, "evidence": "not-evaluated"})
			return
		}
		if req.Action == "request-integration" || req.Action == "request-deployment" {
			writeErr(w, 409, "capability unavailable: delivery queues and execution are not implemented")
			return
		}
		if err := store.ValidateDeliveryMutation(req.DeliveryMutation); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if d, ok, err := deps.Store.ReplayDelivery(repo, principal, req.DeliveryMutation); ok || err != nil {
			if err != nil {
				writeErr(w, 409, err.Error())
				return
			}
			writeJSON(w, 200, map[string]any{"schemaVersion": 1, "delivery": d, "replayed": true})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		branch, revision, target := req.Branch, req.Revision, req.Target
		if req.Action == "request-review" {
			d, err := deps.Store.GetDelivery(repo, req.ID)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			branch, revision, target = d.Branch, d.Revision, d.Target
		}
		if req.Action != "withdraw-review" {
			for _, ref := range []string{branch, target} {
				if _, err := deliveryGit(ctx, cwd, "check-ref-format", "refs/heads/"+ref); err != nil {
					writeErr(w, 400, "branch and target must name local branches")
					return
				}
			}
			head, err := deliveryGit(ctx, cwd, "rev-parse", "--verify", "refs/heads/"+branch+"^{commit}")
			if err != nil || head != revision {
				writeErr(w, 409, "source branch does not match revision; inspect it and use a new request ID for changed content")
				return
			}
			if _, err = deliveryGit(ctx, cwd, "rev-parse", "--verify", "refs/heads/"+target+"^{commit}"); err != nil {
				writeErr(w, 409, "target branch is unavailable")
				return
			}
		}
		d, err := deps.Store.ApplyDelivery(repo, principal, req.DeliveryMutation)
		if err != nil {
			if errors.Is(err, store.ErrDeliveryConflict) {
				writeErr(w, 409, err.Error())
			} else if errors.Is(err, store.ErrDeliveryCapacity) {
				writeErr(w, 409, err.Error())
			} else {
				writeStoreErr(w, err)
			}
			return
		}
		writeJSON(w, 200, map[string]any{"schemaVersion": 1, "delivery": d, "replayed": false})
	}
}

func deliveryView(parent context.Context, cwd string, d store.Delivery) map[string]any {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	current, err := deliveryGit(ctx, cwd, "rev-parse", "--verify", "refs/heads/"+d.Branch+"^{commit}")
	status := "unknown"
	if err == nil {
		status = "unchanged"
		if current != d.Revision {
			status = "changed"
		}
	}
	snapshot := delivery.Observe(ctx, cwd, gitgraph.Key(cwd), d.Target, []delivery.Declaration{{ID: d.ID, Title: d.Title, Branch: d.Branch, Revision: d.Revision, Review: d.Review, Target: d.Target}})
	result := map[string]any{"schemaVersion": 1, "delivery": d, "source": map[string]any{"status": status, "observedRevision": current, "observedAt": time.Now().UTC().Format(time.RFC3339Nano)}, "validation": "unknown", "integration": "unknown", "publication": "unknown", "reviewMeaning": "requested is an agent declaration, not human approval", "coverage": map[string]any{"complete": snapshot.Complete, "issues": snapshot.Issues}}
	for _, change := range snapshot.Changes {
		if change.ID == d.ID {
			result["validation"] = change.Validation
			result["integration"] = change.Integration
			result["evidence"] = change.Evidence
			break
		}
	}
	return result
}
