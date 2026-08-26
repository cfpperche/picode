package server

import (
	"encoding/base64"
	"fmt"
	"io"
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
	mux.HandleFunc("GET /api/agents/{id}/browse", handleAgentBrowse(deps))
	mux.HandleFunc("GET /api/agents/{id}/file", handleAgentFile(deps))
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

var imageMIME = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".gif": "image/gif", ".webp": "image/webp",
}

func agentCwd(deps Deps, id string) (string, error) {
	agent, err := deps.Store.GetAgent(id)
	if err != nil {
		return "", err
	}
	wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
	if err != nil {
		return "", err
	}
	return store.AgentCwd(wk, agent), nil
}

func handleAgentBrowse(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
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

func handleAgentFile(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
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

func relUnderCwd(cwd, rel string) (abs, outRel string, err error) {
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", err
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." {
		rel = ""
	}
	abs = cwdAbs
	if rel != "" {
		abs = filepath.Join(cwdAbs, filepath.FromSlash(rel))
	}
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", "", err
	}
	outRel, err = filepath.Rel(cwdAbs, abs)
	if err != nil {
		return "", "", err
	}
	outRel = filepath.ToSlash(outRel)
	if outRel == ".." || strings.HasPrefix(outRel, "../") {
		return "", "", fmt.Errorf("path escapes workspace")
	}
	if outRel == "." {
		outRel = ""
	}
	return abs, outRel, nil
}

type browseHit struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func browseAgentDir(cwd, rel string) (map[string]any, error) {
	st, err := os.Stat(cwd)
	if err != nil || !st.IsDir() {
		return map[string]any{"cwdOk": false, "dirs": []browseHit{}, "files": []browseHit{}}, nil
	}
	abs, outRel, err := relUnderCwd(cwd, rel)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	var dirs, files []browseHit
	for _, e := range ents {
		name := e.Name()
		child := name
		if outRel != "" {
			child = outRel + "/" + name
		}
		if e.IsDir() {
			if skipFileDir[name] {
				continue
			}
			dirs = append(dirs, browseHit{Name: name, Path: child})
			continue
		}
		files = append(files, browseHit{Name: name, Path: child})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	if dirs == nil {
		dirs = []browseHit{}
	}
	if files == nil {
		files = []browseHit{}
	}
	parent := ""
	if outRel != "" {
		parent = filepath.ToSlash(filepath.Dir(outRel))
		if parent == "." {
			parent = ""
		}
	}
	return map[string]any{
		"cwdOk": true, "dir": outRel, "parent": parent,
		"dirs": dirs, "files": files,
	}, nil
}

func readAgentImage(cwd, rel string) (map[string]any, error) {
	abs, outRel, err := relUnderCwd(cwd, rel)
	if err != nil {
		return nil, err
	}
	mime := imageMIME[strings.ToLower(filepath.Ext(abs))]
	if mime == "" {
		return nil, fmt.Errorf("not an image")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImageB64+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > 4*1024*1024 {
		return nil, fmt.Errorf("each image must be under 4 MB")
	}
	return map[string]any{
		"name": filepath.Base(outRel), "path": outRel, "mime": mime,
		"data": base64.StdEncoding.EncodeToString(data),
	}, nil
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
