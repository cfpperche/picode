package server

// POST /api/workspaces/clone — the one git write reachable from the GUI
// (ADR-0034): clone a remote repository into a fresh directory, then
// register it as a workspace. The destination must not exist or be empty;
// an occupied destination that is already a clone of the same repository
// is adopted instead of re-cloned.

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gitclone"
)

// cloneFn is swapped in tests: a real clone needs a network remote.
var cloneFn = gitclone.Clone

const cloneTimeout = 10 * time.Minute

func registerWorkspaceCloneRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/workspaces/clone", handleCloneWorkspace(deps))
}

func handleCloneWorkspace(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL  string `json:"url"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		rem, err := gitclone.ParseRemote(req.URL)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = rem.Name
		}
		if strings.TrimSpace(req.Path) == "" {
			writeErr(w, http.StatusBadRequest, "destination folder is required")
			return
		}
		dest, err := expandPath(req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "bad destination: "+err.Error())
			return
		}
		if _, err := exec.LookPath("git"); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "git is not installed or not on PATH")
			return
		}

		exists, empty := gitclone.DirUsable(dest)
		if exists && !empty {
			// Already a clone of this very repository → adopt it as the
			// workspace (AddWorkspace is idempotent by absolute path).
			if !gitclone.SameOrigin(dest, rem.URL) {
				writeErr(w, http.StatusConflict,
					"that folder exists and is a different project — pick another destination")
				return
			}
			wk, err := deps.Store.AddWorkspace(name, dest)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			v, err := deps.view(r, wk)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, struct {
				workspaceView
				Adopted bool `json:"adopted"`
			}{v, true})
			return
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			writeErr(w, http.StatusBadRequest, "create parent folder: "+err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), cloneTimeout)
		defer cancel()
		if err := cloneFn(ctx, rem, dest); err != nil {
			writeErr(w, http.StatusBadRequest, cloneFailureMessage(err))
			return
		}
		wk, err := deps.Store.AddWorkspace(name, dest)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		v, err := deps.view(r, wk)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, v)
	}
}

func cloneFailureMessage(err error) string {
	ce, ok := err.(*gitclone.CloneError)
	if !ok {
		return "git clone failed: " + err.Error()
	}
	switch ce.Class {
	case "auth":
		return "private or unreachable repo — run `gh auth login` or add an SSH key on this machine, then retry"
	case "notfound":
		return "repository not found — check the URL (a private repo needs credentials on this machine)"
	case "network":
		return "could not reach the git host — check the URL and your connection"
	}
	msg := ce.Stderr
	if len(msg) > 300 {
		msg = msg[len(msg)-300:]
	}
	if msg == "" {
		msg = "git clone failed"
	}
	return msg
}
