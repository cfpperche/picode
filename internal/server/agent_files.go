package server

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

const (
	fileScanCap = 200
	fileHitCap  = 20
)

var skipFileDir = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".cache": true,
	".next": true, ".turbo": true, ".venv": true, ".tox": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"target": true, "coverage": true, "venv": true, "__pycache__": true,
}

type fileHit struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func registerAgentFileRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/files", handleAgentFiles(deps))
}

func handleAgentFiles(deps Deps) http.HandlerFunc {
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
		hits, ok := searchAgentFiles(store.AgentCwd(wk, agent), r.URL.Query().Get("q"))
		writeJSON(w, http.StatusOK, map[string]any{"hits": hits, "cwdOk": ok})
	}
}

func searchAgentFiles(cwd, q string) ([]fileHit, bool) {
	st, err := os.Stat(cwd)
	if err != nil || !st.IsDir() {
		return []fileHit{}, false
	}
	q = strings.ToLower(strings.TrimSpace(q))
	var hits []fileHit
	scanned := 0
	_ = filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != cwd && skipFileDir[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		scanned++
		if scanned > fileScanCap {
			return fs.SkipAll
		}
		rel, err := filepath.Rel(cwd, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") {
			return nil
		}
		if q == "" && hiddenFile(rel) {
			return nil
		}
		if q != "" {
			base := strings.ToLower(d.Name())
			low := strings.ToLower(rel)
			if !strings.Contains(base, q) && !strings.Contains(low, q) {
				return nil
			}
		}
		hits = append(hits, fileHit{Path: rel, Name: d.Name()})
		return nil
	})
	sort.SliceStable(hits, func(i, j int) bool {
		if q != "" {
			si, sj := fileScore(hits[i], q), fileScore(hits[j], q)
			if si != sj {
				return si < sj
			}
		}
		di := strings.Count(hits[i].Path, "/")
		dj := strings.Count(hits[j].Path, "/")
		if di != dj {
			return di < dj
		}
		return hits[i].Path < hits[j].Path
	})
	if len(hits) > fileHitCap {
		hits = hits[:fileHitCap]
	}
	if hits == nil {
		hits = []fileHit{}
	}
	return hits, true
}

func hiddenFile(rel string) bool {
	for _, part := range strings.Split(rel, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	switch strings.ToLower(filepath.Base(rel)) {
	case "go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "cargo.lock":
		return true
	default:
		return false
	}
}

func fileScore(h fileHit, q string) int {
	base := strings.ToLower(h.Name)
	p := strings.ToLower(h.Path)
	switch {
	case base == q:
		return 0
	case strings.HasPrefix(base, q):
		return 1
	case strings.Contains(base, q):
		return 2
	case strings.Contains(p, q):
		return 3
	default:
		return 4
	}
}
