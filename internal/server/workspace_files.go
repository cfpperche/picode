package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

// A workspace is a file-reading owner in its own right (ADR-0028): the file
// tree must work on a workspace with no agents and no terminals (ADR-0027),
// so these routes confine to the workspace's registered folder the same way
// the agent routes confine to the agent's cwd. The free workspace has no
// folder — its path is a sentinel — so it is refused.
func registerWorkspaceFileRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/workspaces/{id}/browse", handleWorkspaceBrowse(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/file", handleWorkspaceFile(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/text", handleWorkspaceText(deps))
	mux.HandleFunc("PUT /api/workspaces/{id}/text", handlePutWorkspaceText(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/blob", handleWorkspaceBlob(deps))
}

// workspaceFilesCwd resolves the workspace's folder, writing the error
// response itself when there is none to resolve.
func workspaceFilesCwd(deps Deps, w http.ResponseWriter, id string) (string, bool) {
	wk, err := deps.Store.GetWorkspace(id)
	if err != nil {
		writeStoreErr(w, err)
		return "", false
	}
	if store.IsFree(wk) {
		writeErr(w, http.StatusNotFound, "the free workspace has no folder")
		return "", false
	}
	return wk.Path, true
}

func handleWorkspaceBrowse(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		out, err := browseAgentDir(cwd, r.URL.Query().Get("dir"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleWorkspaceFile(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		out, err := readAgentImage(cwd, r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleWorkspaceText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		out, code, err := readAgentText(cwd, r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handlePutWorkspaceText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		var req agentTextPut
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Path == "" {
			req.Path = r.URL.Query().Get("path")
		}
		out, code, err := writeAgentText(cwd, req.Path, req.Text, req.Mtime)
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleWorkspaceBlob(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		mime, data, code, err := readAgentBlob(cwd, r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "private, max-age=0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
