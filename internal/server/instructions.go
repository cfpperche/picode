package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/cliinstructions"
	"github.com/cfpperche/picode/internal/clilaunch"
)

// The Instructions tab (docs/architecture/cli-instructions.md): which
// instruction files — AGENTS.md, CLAUDE.md and their kin — each agent CLI
// reads for a session started in this workspace, and why it leaves the
// others out. Read from disk on every request and stored nowhere; the answer
// carries paths, sizes and verdicts, never a file's text.
func registerInstructionsRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/workspaces/{id}/instructions", handleWorkspaceInstructions(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/instructions/fix", handleInstructionsFixGet(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/instructions/fix", handleInstructionsFixApply(deps))
}

// The two halves of a fix (ADR-0204): GET shows the exact change, computed
// from the files on disk; POST writes it only if every file still has the
// hash the person saw, and otherwise answers 409 with the fresh change. No
// git command runs: the edit shows up as an unstaged change.
func handleInstructionsFixGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		fix, err := cliinstructions.ProposeFix(cwd, r.URL.Query().Get("id"))
		if writeFixErr(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, fix)
	}
}

func handleInstructionsFixApply(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		var body struct {
			ID     string            `json:"id"`
			Hashes map[string]string `json:"hashes"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil || body.ID == "" {
			writeErr(w, http.StatusBadRequest, "Name the fix to apply.")
			return
		}
		fix, err := cliinstructions.ApplyFix(cwd, body.ID, body.Hashes)
		if errors.Is(err, cliinstructions.ErrDrift) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "These files changed since this change was shown.", "fix": fix})
			return
		}
		if writeFixErr(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "fix": fix})
	}
}

func writeFixErr(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, cliinstructions.ErrNoFix):
		writeErr(w, http.StatusNotFound, "This fix no longer applies.")
	case errors.Is(err, os.ErrPermission):
		writeErr(w, http.StatusBadRequest, "That file is outside this workspace.")
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
	return true
}

func handleWorkspaceInstructions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
			writeErr(w, http.StatusNotFound, "This workspace's folder is gone.")
			return
		}
		start := cwd
		if rel := r.URL.Query().Get("start"); rel != "" {
			clean := filepath.Clean(filepath.FromSlash(rel))
			if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				writeErr(w, http.StatusBadRequest, "Choose a folder inside this workspace.")
				return
			}
			start = filepath.Join(cwd, clean)
			if st, err := os.Stat(start); err != nil || !st.IsDir() {
				writeErr(w, http.StatusNotFound, "That folder is gone.")
				return
			}
		}
		rep, err := cliinstructions.Resolve(cliinstructions.Env{Root: cwd, Start: start, Installed: installedCLIs(deps)})
		if errors.Is(err, os.ErrPermission) {
			writeErr(w, http.StatusBadRequest, "Choose a folder inside this workspace.")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		rep.Agents = observedReads(deps, r.PathValue("id"), cwd)
		writeJSON(w, http.StatusOK, rep)
	}
}

// observedReads lists, for each terminal of the workspace whose pinned
// session (ADR-0084) belongs to a CLI that records its instruction files,
// the files that session loaded. A terminal with no pinned session, or a
// record that cannot be read, is left out rather than guessed.
func observedReads(deps Deps, wsID, root string) []cliinstructions.Read {
	out := []cliinstructions.Read{}
	terms, err := deps.Store.ListTerminals()
	if err != nil {
		return out
	}
	home, _ := os.UserHomeDir()
	for _, t := range terms {
		if t.WorkspaceID != wsID {
			continue
		}
		launch, err := deps.Store.TerminalLaunch(t.ID)
		if err != nil || launch == nil || launch.LastSession == nil || !cliinstructions.Records(launch.LastSession.CLI) {
			continue
		}
		ls := launch.LastSession
		files, ok := cliinstructions.ObservedFiles(ls.CLI, ls.SessionID, ls.Path)
		if !ok {
			continue
		}
		shown := make([]string, 0, len(files))
		for _, f := range files {
			shown = append(shown, cliinstructions.Display(root, home, f))
		}
		out = append(out, cliinstructions.Read{TerminalID: t.ID, Name: t.Name, CLI: ls.CLI, UpdatedAt: ls.UpdatedAt, Files: shown})
	}
	return out
}

// installedCLIs answers the way the Agent CLIs page does: the catalog row's
// executable, resolved through its launch settings (resolveCLIExecutable).
// No retry here: a CLI missing mid-update shows as not installed for one read.
func installedCLIs(deps Deps) func(string) bool {
	have := map[string]bool{}
	for _, cli := range clilaunch.Catalog() {
		c, err := cliConfig(deps, cli.ID)
		if err != nil {
			continue
		}
		if _, err := resolveCLIExecutable(cli, c); err == nil {
			have[cli.ID] = true
		}
	}
	return func(id string) bool { return have[id] }
}
