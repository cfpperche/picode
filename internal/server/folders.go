package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const fsListCap = 500

type fsEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type fsList struct {
	Path   string    `json:"path"`
	Parent string    `json:"parent,omitempty"`
	Dirs   []fsEntry `json:"dirs"`
}

func registerFolderRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/fs", handleFsList)
	mux.HandleFunc("POST /api/fs/mkdir", handleFsMkdir)
}

func expandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if p == "" || p == "~" {
		return home, nil
	}
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(home, p[2:])
	}
	return filepath.Abs(filepath.Clean(p))
}

func handleFsList(w http.ResponseWriter, r *http.Request) {
	abs, err := expandPath(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	st, err := os.Stat(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !st.IsDir() {
		writeErr(w, http.StatusBadRequest, "not a directory")
		return
	}
	ents, err := os.ReadDir(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out := fsList{Path: abs, Dirs: []fsEntry{}}
	parent := filepath.Dir(abs)
	if parent != abs {
		out.Parent = parent
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		out.Dirs = append(out.Dirs, fsEntry{Name: name, Path: filepath.Join(abs, name)})
		if len(out.Dirs) >= fsListCap {
			break
		}
	}
	sort.Slice(out.Dirs, func(i, j int) bool { return strings.ToLower(out.Dirs[i].Name) < strings.ToLower(out.Dirs[j].Name) })
	writeJSON(w, http.StatusOK, out)
}

func handleFsMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		writeErr(w, http.StatusBadRequest, "path is required")
		return
	}
	abs, err := expandPath(req.Path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, fsEntry{Name: filepath.Base(abs), Path: abs})
}
