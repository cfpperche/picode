package server

import (
	"encoding/json"
	"github.com/cfpperche/picode/internal/osopen"
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
	Label  string    `json:"label,omitempty"`
	Parent string    `json:"parent,omitempty"`
	Dirs   []fsEntry `json:"dirs"`
	Places []fsEntry `json:"places,omitempty"`
}

func registerFolderRoutes(mux Registrar) {
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
	} else if wsl, ok := winPathToWSL(p); ok && isWSL() {
		p = wsl
	}
	return filepath.Abs(filepath.Clean(p))
}

func isWSL() bool { return osopen.RunningWSL() }

// winPathToWSL maps "C:\Users\x" or "C:/Users/x" to "/mnt/c/Users/x".
func winPathToWSL(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if len(p) < 2 || p[1] != ':' {
		return "", false
	}
	drive := p[0]
	if (drive < 'A' || drive > 'Z') && (drive < 'a' || drive > 'z') {
		return "", false
	}
	rest := strings.ReplaceAll(p[2:], "\\", "/")
	rest = strings.TrimPrefix(rest, "/")
	letter := strings.ToLower(string(drive))
	if rest == "" {
		return "/mnt/" + letter, true
	}
	return "/mnt/" + letter + "/" + rest, true
}

func windowsMounts() []fsEntry {
	ents, err := os.ReadDir("/mnt")
	if err != nil {
		return nil
	}
	var out []fsEntry
	for _, e := range ents {
		name := e.Name()
		if len(name) != 1 || name[0] < 'a' || name[0] > 'z' {
			continue
		}
		p := filepath.Join("/mnt", name)
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			continue
		}
		out = append(out, fsEntry{Name: strings.ToUpper(name) + ":", Path: p})
	}
	return out
}

func wslPlaces(home string) []fsEntry {
	if !isWSL() {
		return nil
	}
	out := []fsEntry{{Name: "Home", Path: home}}
	out = append(out, windowsMounts()...)
	if len(out) == 1 {
		return out
	}
	return out
}

func winLabel(abs string) string {
	slash := filepath.ToSlash(abs)
	const prefix = "/mnt/"
	if !strings.HasPrefix(slash, prefix) || len(slash) < 6 {
		return ""
	}
	rest := slash[len(prefix):]
	if rest[0] < 'a' || rest[0] > 'z' {
		return ""
	}
	if len(rest) > 1 && rest[1] != '/' {
		return ""
	}
	drive := strings.ToUpper(rest[:1]) + ":"
	if len(rest) == 1 {
		return drive + "\\"
	}
	return drive + strings.ReplaceAll(rest[1:], "/", "\\")
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
	home, _ := os.UserHomeDir()
	out := fsList{Path: abs, Label: winLabel(abs), Dirs: []fsEntry{}, Places: wslPlaces(home)}
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
