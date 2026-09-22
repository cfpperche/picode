package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/delivery"
	"github.com/cfpperche/picode/internal/gitgraph"
)

type deliveryReadEntry struct {
	done chan struct{}
	at   time.Time
	view delivery.Snapshot
}

func registerDeliveryObservationRoutes(mux Registrar, deps Deps) {
	var mu sync.Mutex
	cache := map[string]*deliveryReadEntry{}
	for _, route := range []struct{ ns, kind string }{{"workspaces", "workspace"}, {"agents", "agent"}, {"terminals", "term"}} {
		mux.HandleFunc("GET /api/"+route.ns+"/{id}/delivery", func(w http.ResponseWriter, r *http.Request) {
			cwd, ok := previewOwnerCwd(deps, w, r, route.kind, r.PathValue("id"))
			if !ok {
				return
			}
			cwd = canonDir(cwd)
			if !checkFileRoot(w, r, cwd) {
				return
			}
			repo := gitgraph.Key(cwd)
			if repo == "" {
				writeErr(w, 404, "This folder is not a Git repository.")
				return
			}
			target := r.URL.Query().Get("target")
			if len(target) > 512 {
				writeErr(w, 400, "Invalid target branch")
				return
			}
			// Root participates so no owner can receive another checkout's pinned root.
			key := repo + "\x00" + cwd + "\x00" + target
			mu.Lock()
			entry := cache[key]
			if entry != nil {
				select {
				case <-entry.done:
					if time.Since(entry.at) > 5*time.Second || r.URL.Query().Get("fresh") == "1" {
						entry = nil
					}
				default:
				}
			}
			if entry != nil {
				mu.Unlock()
				select {
				case <-entry.done:
					out, err := queueLayer(deps, route.kind, r.PathValue("id"), repo, entry.view)
					if err != nil {
						writeErr(w, 500, err.Error())
						return
					}
					writeJSON(w, 200, out)
				case <-r.Context().Done():
				}
				return
			}
			if len(cache) >= 64 {
				for k, v := range cache {
					select {
					case <-v.done:
						delete(cache, k)
					default:
					}
				}
			}
			if len(cache) >= 64 {
				mu.Unlock()
				writeErr(w, 503, "Delivery observations are busy. Try again.")
				return
			}
			entry = &deliveryReadEntry{done: make(chan struct{})}
			cache[key] = entry
			mu.Unlock()
			declarations := []delivery.Declaration{}
			var cursor int64
			readFailed := false
			for {
				page, err := deps.Store.ListDeliveries(repo, cursor)
				if err != nil {
					readFailed = true
					break
				}
				if len(page) == 0 {
					break
				}
				for _, d := range page {
					declarations = append(declarations, delivery.Declaration{ID: d.ID, Title: d.Title, Branch: d.Branch, Revision: d.Revision, Review: d.Review, Target: d.Target})
				}
				cursor = page[len(page)-1].Sequence
				if len(page) < 101 {
					break
				}
			}
			view := delivery.Observe(r.Context(), cwd, repo, target, declarations)
			if readFailed {
				view.Complete = false
				view.Issues = append(view.Issues, "Registered deliveries unavailable")
			}
			// Association is occupancy, never authorship or native conversation ownership.
			agents, err := deps.Store.ListAllAgents()
			if err != nil {
				view.Complete = false
				view.Issues = append(view.Issues, "Agent associations unavailable")
			} else {
				for _, a := range agents {
					path, e := agentCwd(deps, a.ID)
					if e != nil {
						continue
					}
					path = canonDir(path)
					for i := range view.Changes {
						if path == canonDir(view.Changes[i].Worktree) && view.Changes[i].Worktree != "" {
							view.Changes[i].Agents = append(view.Changes[i].Agents, a.Name)
						}
					}
				}
			}
			mu.Lock()
			entry.view = view
			entry.at = time.Now()
			close(entry.done)
			mu.Unlock()
			out, err := queueLayer(deps, route.kind, r.PathValue("id"), repo, view)
			if err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			writeJSON(w, 200, out)
		})
	}
}
