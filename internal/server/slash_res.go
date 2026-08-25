package server

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/slashres"
	"github.com/cfpperche/picode/internal/store"
)

func registerSlashRes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/slash", handleAgentSlash(deps))
	mux.HandleFunc("GET /api/agents/{id}/export", handleAgentExport(deps))
	mux.HandleFunc("POST /api/agents/{id}/import", handleAgentImport(deps))
	mux.HandleFunc("GET /api/changelog/pi", handlePiChangelog)
	mux.HandleFunc("POST /api/agents/{id}/share", handleAgentShare(deps))
}

func handleAgentSlash(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd := store.AgentCwd(wk, agent)
		items := slashres.List(cwd, pisettings.Trusted(cwd))
		var skills, templates []slashres.Item
		for _, it := range items {
			if it.Kind == "template" {
				templates = append(templates, it)
			} else {
				skills = append(skills, it)
			}
		}
		if skills == nil {
			skills = []slashres.Item{}
		}
		if templates == nil {
			templates = []slashres.Item{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"skills": skills, "templates": templates})
	}
}

func handleAgentExport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if agent.SessionPath == nil || *agent.SessionPath == "" {
			writeErr(w, http.StatusNotFound, "no session file yet")
			return
		}
		path := *agent.SessionPath
		f, err := os.Open(path)
		if err != nil {
			writeErr(w, http.StatusNotFound, "session file missing")
			return
		}
		defer f.Close()
		name := filepath.Base(path)
		if name == "" || name == "." {
			name = "session.jsonl"
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		_, _ = io.Copy(w, f)
	}
}

func handleAgentImport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeErr(w, http.StatusBadRequest, "file required")
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "file required")
			return
		}
		defer file.Close()
		name := filepath.Base(hdr.Filename)
		if !strings.HasSuffix(strings.ToLower(name), ".jsonl") {
			writeErr(w, http.StatusBadRequest, "need a .jsonl file")
			return
		}
		dir := filepath.Join(store.AgentCwd(wk, agent), ".pi", "sessions")
		if agent.SessionPath != nil && *agent.SessionPath != "" {
			dir = filepath.Dir(*agent.SessionPath)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		dst := filepath.Join(dir, name)
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		_, copyErr := io.Copy(out, file)
		_ = out.Close()
		if copyErr != nil {
			writeErr(w, http.StatusInternalServerError, copyErr.Error())
			return
		}
		if _, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &dst}); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		mode := deps.runMode(r, agent.ID)
		if err := restartSameMode(r.Context(), deps, wk, agent.ID, mode); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "current": dst})
	}
}

func handlePiChangelog(w http.ResponseWriter, r *http.Request) {
	root, err := exec.Command("npm", "root", "-g").Output()
	if err != nil {
		writeErr(w, http.StatusNotFound, "pi changelog not found")
		return
	}
	p := filepath.Join(strings.TrimSpace(string(root)), "@earendil-works", "pi-coding-agent", "CHANGELOG.md")
	b, err := os.ReadFile(p)
	if err != nil {
		writeErr(w, http.StatusNotFound, "pi changelog not found")
		return
	}
	if len(b) > 48*1024 {
		b = b[:48*1024]
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": string(b)})
}
