package server

import (
	"context"
	"sync"

	"github.com/cfpperche/picode/internal/delivery"
	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
)

// unknownOutcomeNote is what an entry left running by a stopped daemon carries.
// It never becomes a success or a failure on its own.
const unknownOutcomeNote = "PiCode stopped while this entry ran; the outcome is unknown — inspect the repository, then finish, fail or withdraw it"

// The runner's seat in the daemon (ADR-0182). Authorizing is execution
// authority: when the owner authorizes an entry, that repository drains — the
// entry just authorized, then whatever else is authorized behind it, in the
// order the owner set. One repository runs one operation at a time: the durable
// half of that promise is the store's own `start` guard, this is the live half.
type queueRuns struct {
	deps  Deps
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newQueueRuns(deps Deps) *queueRuns {
	return &queueRuns{deps: deps, locks: map[string]*sync.Mutex{}}
}

func (q *queueRuns) repoLock(repo string) *sync.Mutex {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.locks[repo] == nil {
		q.locks[repo] = &sync.Mutex{}
	}
	return q.locks[repo]
}

// launch drains a repository in the background and returns at once: a request
// never waits for an integration. Two launches for the same repository
// serialize, and the second finds the queue as the first left it.
func (q *queueRuns) launch(cwd, repo, workspaceID string) {
	if q.deps.Store == nil || repo == "" || cwd == "" {
		return
	}
	go func() {
		lock := q.repoLock(repo)
		lock.Lock()
		defer lock.Unlock()
		q.drain(context.Background(), cwd, repo, workspaceID)
	}()
}

// drain runs authorized entries until none is left. A run that cannot be
// recorded stops the drain instead of spinning: the entry keeps the state it
// had, and the next launch tries again.
func (q *queueRuns) drain(ctx context.Context, cwd, repo, workspaceID string) {
	for {
		entries, err := q.deps.Store.ListQueueEntries(repo, 0)
		if err != nil {
			return
		}
		var next *store.QueueEntry
		for i := range entries {
			if entries[i].State == store.QueueAuthorized {
				next = &entries[i]
				break
			}
		}
		if next == nil {
			return
		}
		d, err := q.deps.Store.GetDelivery(repo, next.DeliveryID)
		if err != nil {
			return
		}
		settings, err := q.deps.Store.EffectiveIntegrationSettings(workspaceID)
		if err != nil {
			return
		}
		out, err := delivery.RunEntry(ctx, delivery.RunConfig{
			Store: q.deps.Store, Repo: repo, Cwd: cwd, Settings: settings, Delivery: d, Entry: *next})
		if err != nil || out.State == next.State {
			return
		}
	}
}

// ResumeDeliveryQueue reconciles the queue with a daemon start (ADR-0182). The
// daemon calls it once, from the path that actually serves: an entry left
// running has no known outcome, and every authorized entry runs. It is not
// called from route registration, so a process that only mounts the routes
// never touches a queue it does not own.
func ResumeDeliveryQueue(deps Deps) {
	if deps.Store == nil {
		return
	}
	go newQueueRuns(deps).resume()
}

// resume does that reconciliation. An entry that was running has no known
// outcome — the process that ran it is gone — so it is marked as unknown and
// left for the owner to finish, fail or withdraw after inspecting the
// repository. Nothing is retried automatically and no success is inferred. Then
// every repository that holds an authorized entry gets its run.
func (q *queueRuns) resume() {
	if q.deps.Store == nil {
		return
	}
	workspaces, err := q.deps.Store.ListWorkspaces()
	if err != nil {
		return
	}
	for _, ws := range workspaces {
		cwd := canonDir(ws.Path)
		repo := gitgraph.Key(cwd)
		if repo == "" {
			continue
		}
		if _, err := q.deps.Store.MarkRunningUnknown(repo, unknownOutcomeNote); err != nil {
			continue
		}
		q.launch(cwd, repo, ws.ID)
	}
}
