package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/gitcmd"
	"github.com/cfpperche/picode/internal/gitgraph"
)

// ADR-0096: one composer, on the server. The browser sends an action and its
// arguments and gets back the exact command, its risk tier, the plain-words
// verb and the sentence an agent would be asked. It never builds a shell
// string of its own, so quoting is decided in one place for every surface.
//
// Composing is not delivering. What comes back travels to ADR-0078's three
// doors — POST .../terminals/{id}/type, .../run, .../agents/{id}/ask — which
// keep their own guards: the root precondition, the foreground check and the
// repository interlock. Nothing here runs git, and nothing here writes.
//
// The response is also the preview the form shows before anything is sent,
// so what the reader is promised and what the terminal receives are the same
// string by construction.
func registerGitComposeRoutes(mux Registrar, deps Deps) {
	// The catalog is static: which actions exist, their risk tier and the
	// fields each needs. The browser reads it instead of carrying its own
	// copy of the tiers, so a tier C action can never be rendered with a
	// tier B gate.
	mux.HandleFunc("GET /api/git/actions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"actions": gitcmd.Catalog()})
	})
	mux.HandleFunc("POST /api/agents/{id}/git/compose", handleAgentGitCompose(deps))
	mux.HandleFunc("POST /api/terminals/{id}/git/compose", handleTerminalGitCompose(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/git/compose", handleWorkspaceGitCompose(deps))
}

type composeRequest struct {
	Action  string `json:"action"`
	Target  string `json:"target"`
	Name    string `json:"name"`
	Message string `json:"message"`
	Root    string `json:"root"`
}

type composeView struct {
	Command string `json:"command"`
	Tier    string `json:"tier"`
	Verb    string `json:"verb"`
	Prompt  string `json:"prompt"`
	// Branch and Upstream are what the server read from the checkout, not
	// what the caller claimed: a push publishes the branch that is really
	// checked out here.
	Branch   string `json:"branch"`
	Upstream string `json:"upstream,omitempty"`
	Detached bool   `json:"detached,omitempty"`
}

func handleAgentGitCompose(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeCompose(w, r, cwd)
	}
}

func handleTerminalGitCompose(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeCompose(w, r, liveTermCwd(deps, r, term))
	}
}

func handleWorkspaceGitCompose(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		writeCompose(w, r, cwd)
	}
}

// composeErrStatus maps a composer error to the status the browser expects:
// a bad field is the caller's problem, an unknown action is a 404 because the
// route exists but that verb does not.
func composeErrStatus(err error) int {
	if errors.Is(err, gitcmd.ErrUnknownAction) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

// pickRemote prefers origin, falls back to the only remote a repository has,
// and leaves the choice to the composer's default when there is none — the
// menu hides remote actions in that case, so this is the belt to that brace.
func pickRemote(remotes []string) string {
	for _, name := range remotes {
		if name == "origin" {
			return "origin"
		}
	}
	if len(remotes) == 1 {
		return remotes[0]
	}
	return ""
}

func writeCompose(w http.ResponseWriter, r *http.Request, cwd string) {
	var req composeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// The root precondition of ADR-0074, as every other owner read applies
	// it: a command composed for one folder must never be handed to another.
	// It is checked from the body here because this is a POST.
	if req.Root != "" && req.Root != canonDir(cwd) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "This folder changed. Refresh the graph.", "reason": "moved", "cwd": cwd,
		})
		return
	}
	if gitgraph.Key(cwd) == "" {
		writeErr(w, http.StatusNotFound, "not a git repository")
		return
	}
	branch, upstream, detached := gitgraph.Checkout(cwd)
	args := gitcmd.Args{
		Target:   strings.TrimSpace(req.Target),
		Name:     strings.TrimSpace(req.Name),
		Message:  req.Message,
		Branch:   branch,
		Upstream: upstream,
		Remote:   pickRemote(gitgraph.Remotes(cwd)),
	}
	cmd, tier, err := gitcmd.Compose(req.Action, args)
	if err != nil {
		writeErr(w, composeErrStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, composeView{
		Command:  cmd,
		Tier:     string(tier),
		Verb:     gitcmd.Verb(req.Action, args),
		Prompt:   gitcmd.Prompt(req.Action, args, canonDir(cwd)),
		Branch:   branch,
		Upstream: upstream,
		Detached: detached,
	})
}
