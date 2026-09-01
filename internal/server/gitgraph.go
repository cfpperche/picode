package server

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
)

// ADR-0022: the graph belongs to a repository, and the route belongs to the
// owner that asked. The server never takes a repository path from the URL, so
// the cwd confinement that guards file reads guards this too.

const (
	defaultGraphLimit = 250
	// Load earlier doubles the window client-side; the ceiling only exists so
	// a hand-written URL cannot ask for the unbounded history of a huge repo.
	maxGraphLimit = 10000
)

// occupant is an agent living in one of the repository's worktrees. It is why
// the graph is worth showing in PiCode rather than in any git client: it says
// which of your agents is on which branch.
//
// Liveness is deliberately absent. Answering it means one tmux call per agent
// inside this loop, and the agents feed the sidebar already reads carries it —
// the browser joins on ID rather than paying for a second source of truth.
type occupant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type worktreeView struct {
	gitgraph.Worktree
	Agents []occupant `json:"agents"`
}

type graphView struct {
	Key         string                    `json:"key"`
	Name        string                    `json:"name"`
	Head        string                    `json:"head"`
	Commits     []gitgraph.Commit         `json:"commits"`
	Refs        []gitgraph.Ref            `json:"refs"`
	Worktrees   []worktreeView            `json:"worktrees"`
	Uncommitted *gitgraph.UncommittedInfo `json:"uncommitted,omitempty"`
	More        bool                      `json:"more"`
	Token       string                    `json:"token,omitempty"`
}

// gitHeadView is the cheap poll target: the browser refetches the graph only
// when Token changes (ADR-0038).
type gitHeadView struct {
	Key         string `json:"key"`
	Token       string `json:"token"`
	Uncommitted int    `json:"uncommitted"`
}

// gitKeyOf resolves a directory to its repository. A var so a test can count
// how often it runs — which is the whole point of the memo below.
var gitKeyOf = gitgraph.Key

func registerGitGraphRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/git", handleAgentGraph(deps))
	mux.HandleFunc("GET /api/terminals/{id}/git", handleTerminalGraph(deps))
	mux.HandleFunc("GET /api/agents/{id}/git/commit", handleAgentCommit(deps))
	mux.HandleFunc("GET /api/terminals/{id}/git/commit", handleTerminalCommit(deps))
	mux.HandleFunc("GET /api/agents/{id}/git/head", handleAgentGitHead(deps))
	mux.HandleFunc("GET /api/terminals/{id}/git/head", handleTerminalGitHead(deps))
}

func handleAgentGitHead(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeGitHead(w, cwd)
	}
}

func handleTerminalGitHead(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeGitHead(w, liveTermCwd(deps, r, term))
	}
}

func writeGitHead(w http.ResponseWriter, cwd string) {
	key, token, dirty := gitgraph.Token(cwd)
	if key == "" {
		writeErr(w, http.StatusNotFound, "not a git repository")
		return
	}
	writeJSON(w, http.StatusOK, gitHeadView{Key: key, Token: token, Uncommitted: dirty})
}

func handleAgentGraph(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeGraph(w, r, deps, cwd)
	}
}

func handleTerminalGraph(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		// The terminal's cwd is live, so the repository is resolved from where
		// the pane is *now*. The browser pins the answer; the server does not
		// remember it (ADR-0022).
		writeGraph(w, r, deps, liveTermCwd(deps, r, term))
	}
}

func writeGraph(w http.ResponseWriter, r *http.Request, deps Deps, cwd string) {
	// Token first, load second: if a commit lands in between, the token is
	// older than the data and the next poll refetches once for nothing — the
	// safe direction. The other order can hand out a token newer than the
	// graph and go quiet on a stale picture.
	_, token, _ := gitgraph.Token(cwd)
	g := gitgraph.Load(cwd, graphLimit(r))
	if g == nil {
		writeErr(w, http.StatusNotFound, "not a git repository")
		return
	}
	view := deps.graphView(g)
	view.Token = token
	writeJSON(w, http.StatusOK, view)
}

func graphLimit(r *http.Request) int {
	n, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if err != nil || n <= 0 {
		return defaultGraphLimit
	}
	if n > maxGraphLimit {
		return maxGraphLimit
	}
	return n
}

func (deps Deps) graphView(g *gitgraph.Graph) graphView {
	view := graphView{
		Key: g.Key, Name: g.Name, Head: g.Head,
		Commits: g.Commits, Refs: g.Refs, More: g.More,
		Uncommitted: g.Uncommitted,
		Worktrees:   make([]worktreeView, 0, len(g.Worktrees)),
	}
	byWorktree := deps.occupantsByWorktree(g)
	for _, wt := range g.Worktrees {
		agents := byWorktree[canonDir(wt.Path)]
		if agents == nil {
			agents = []occupant{}
		}
		view.Worktrees = append(view.Worktrees, worktreeView{Worktree: wt, Agents: agents})
	}
	return view
}

// occupantsByWorktree attributes every known agent to at most one worktree of
// this repository, keyed by the canonical worktree path.
//
// | Agent cwd relative to a worktree | What happens |
// |---|---|
// | equal to it | attributed, no git call — the path alone is proof |
// | strictly inside it | attributed to the *deepest* containing worktree, but only after `gitgraph.Key` confirms the cwd is this repo, because an unrelated repo can be nested inside a checkout |
// | inside none of them | skipped |
//
// The confirmation is what costs a subprocess, and it only runs for the rare
// agent whose cwd sits below a worktree root rather than at it.
func (deps Deps) occupantsByWorktree(g *gitgraph.Graph) map[string][]occupant {
	agents, err := deps.Store.ListAllAgents()
	if err != nil || len(g.Worktrees) == 0 {
		return nil
	}
	roots := make([]string, 0, len(g.Worktrees))
	for _, wt := range g.Worktrees {
		if c := canonDir(wt.Path); c != "" {
			roots = append(roots, c)
		}
	}

	out := map[string][]occupant{}
	wsCache := map[string]store.Workspace{}
	// Resolving a directory to its repository spawns git, ~23ms measured, and
	// agents commonly share a directory. Remembered for this request only: a
	// folder cannot change repository inside one graph load, and remembering
	// longer would mean answering from a stale note after someone moves one.
	keyOf := map[string]string{}
	for _, a := range agents {
		wk, ok := wsCache[a.WorkspaceID]
		if !ok {
			wk, err = deps.Store.GetWorkspace(a.WorkspaceID)
			if err != nil {
				continue
			}
			wsCache[a.WorkspaceID] = wk
		}
		cwd := canonDir(store.AgentCwd(wk, a))
		if cwd == "" {
			continue
		}
		root, exact := deepestRoot(roots, cwd)
		if root == "" {
			continue
		}
		if !exact {
			key, seen := keyOf[cwd]
			if !seen {
				key = gitKeyOf(cwd)
				keyOf[cwd] = key
			}
			if key != g.Key {
				continue // a different repository nested inside this checkout
			}
		}
		out[root] = append(out[root], occupant{ID: a.ID, Name: a.Name})
	}
	return out
}

// deepestRoot returns the longest root containing cwd, and whether cwd is that
// root itself. Comparison is by path component: /a/b must not match /a/bc.
func deepestRoot(roots []string, cwd string) (root string, exact bool) {
	for _, r := range roots {
		if r == cwd {
			return r, true
		}
		if strings.HasPrefix(cwd, r+string(filepath.Separator)) && len(r) > len(root) {
			root = r
		}
	}
	return root, false
}

func handleAgentCommit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeCommit(w, r, cwd)
	}
}

func handleTerminalCommit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeCommit(w, r, liveTermCwd(deps, r, term))
	}
}

// writeCommit reads one commit through the owner's cwd. The hash is validated
// inside gitgraph.Show, which refuses anything but a full object name — so a
// caller cannot smuggle a flag or a ref into the git command line.
func writeCommit(w http.ResponseWriter, r *http.Request, cwd string) {
	hash := strings.TrimSpace(r.URL.Query().Get("hash"))
	if hash == "" {
		writeErr(w, http.StatusBadRequest, "pass ?hash=<commit>")
		return
	}
	detail := gitgraph.Show(cwd, hash)
	if detail == nil {
		writeErr(w, http.StatusNotFound, "no such commit in this repository")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
