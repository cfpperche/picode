package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

const (
	fileScanCap  = 200
	fileHitCap   = 20
	maxAgentText = 1 << 20
	maxAgentBlob = 32 << 20
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
	mux.HandleFunc("GET /api/agents/{id}/text", handleAgentText(deps))
	mux.HandleFunc("PUT /api/agents/{id}/text", handlePutAgentText(deps))
	mux.HandleFunc("GET /api/agents/{id}/blob", handleAgentBlob(deps))
	mux.HandleFunc("GET /api/agents/{id}/cwd", handleAgentCwd(deps))
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

var blobMIME = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".gif": "image/gif", ".webp": "image/webp",
	".pdf": "application/pdf",
	".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg", ".m4a": "audio/mp4",
	".mp4": "video/mp4", ".webm": "video/webm", ".mkv": "video/x-matroska",
	".glb": "model/gltf-binary", ".gltf": "model/gltf+json",
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

func handleAgentCwd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fallback, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd := fallback
		if deps.Tmux != nil && deps.Tmux.Available() {
			if p, err := deps.Tmux.PaneCwd(r.Context(), tmux.SessionName(r.PathValue("id"))); err == nil && strings.TrimSpace(p) != "" {
				cwd = p
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"cwd": cwd})
	}
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

func handleAgentBlob(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
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

func handleAgentText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
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

type agentTextPut struct {
	Path  string `json:"path"`
	Text  string `json:"text"`
	Mtime int64  `json:"mtime"`
}

func handlePutAgentText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
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

func expandUser(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel != "~" && !strings.HasPrefix(rel, "~/") {
		return rel, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if rel == "~" {
		return home, nil
	}
	return filepath.Join(home, rel[2:]), nil
}

func relUnderCwd(cwd, rel string) (abs, outRel string, err error) {
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", err
	}
	rel, err = expandUser(rel)
	if err != nil {
		return "", "", err
	}
	var cand string
	if filepath.IsAbs(rel) {
		cand = filepath.Clean(rel)
	} else {
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "." {
			rel = ""
		}
		cand = cwdAbs
		if rel != "" {
			cand = filepath.Join(cwdAbs, filepath.FromSlash(rel))
		}
	}
	abs, err = filepath.Abs(cand)
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

func readAgentText(cwd, rel string) (map[string]any, int, error) {
	if strings.TrimSpace(rel) == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("path is required")
	}
	abs, outRel, err := relUnderCwd(cwd, rel)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, fmt.Errorf("that file is gone")
		}
		return nil, http.StatusBadRequest, err
	}
	if st.IsDir() {
		return nil, http.StatusBadRequest, fmt.Errorf("that's a folder")
	}
	if st.Size() > maxAgentText {
		return nil, http.StatusRequestEntityTooLarge, fmt.Errorf("this file is too large")
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	if len(b) > maxAgentText {
		return nil, http.StatusRequestEntityTooLarge, fmt.Errorf("this file is too large")
	}
	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("can't show this file")
	}
	return map[string]any{
		"path":  outRel,
		"name":  filepath.Base(outRel),
		"text":  string(b),
		"bytes": len(b),
		"mtime": st.ModTime().UnixMilli(),
	}, http.StatusOK, nil
}

func readAgentBlob(cwd, rel string) (string, []byte, int, error) {
	if strings.TrimSpace(rel) == "" {
		return "", nil, http.StatusBadRequest, fmt.Errorf("path is required")
	}
	abs, _, err := relUnderCwd(cwd, rel)
	if err != nil {
		return "", nil, http.StatusBadRequest, err
	}
	ext := strings.ToLower(filepath.Ext(abs))
	mime := blobMIME[ext]
	if mime == "" {
		return "", nil, http.StatusUnsupportedMediaType, fmt.Errorf("can't show this file")
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, http.StatusNotFound, fmt.Errorf("that file is gone")
		}
		return "", nil, http.StatusBadRequest, err
	}
	if st.IsDir() {
		return "", nil, http.StatusBadRequest, fmt.Errorf("that's a folder")
	}
	if st.Size() > maxAgentBlob {
		return "", nil, http.StatusRequestEntityTooLarge, fmt.Errorf("this file is too large")
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", nil, http.StatusBadRequest, err
	}
	return mime, b, http.StatusOK, nil
}

func writeAgentText(cwd, rel, text string, mtime int64) (map[string]any, int, error) {
	if strings.TrimSpace(rel) == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("path is required")
	}
	if mtime == 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("mtime is required")
	}
	if len(text) > maxAgentText {
		return nil, http.StatusRequestEntityTooLarge, fmt.Errorf("this file is too large")
	}
	if strings.ContainsRune(text, 0) {
		return nil, http.StatusBadRequest, fmt.Errorf("can't write this file")
	}
	abs, outRel, err := relUnderCwd(cwd, rel)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, fmt.Errorf("that file is gone")
		}
		return nil, http.StatusBadRequest, err
	}
	if st.IsDir() {
		return nil, http.StatusBadRequest, fmt.Errorf("that's a folder")
	}
	if st.ModTime().UnixMilli() != mtime {
		return nil, http.StatusConflict, fmt.Errorf("file changed on disk")
	}
	if err := os.WriteFile(abs, []byte(text), st.Mode().Perm()); err != nil {
		return nil, http.StatusBadRequest, err
	}
	return readAgentText(cwd, outRel)
}

type browseHit struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func browseAgentDir(cwd, rel string) (map[string]any, error) {
	st, err := os.Stat(cwd)
	if err != nil || !st.IsDir() {
		return map[string]any{"cwdOk": false, "root": "", "dirs": []browseHit{}, "files": []browseHit{}}, nil
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
	// root names the tree's identity: the same canonical answer for every
	// owner whose reads are confined to this folder, so the app can fold
	// their tabs into one (ADR-0028).
	return map[string]any{
		"cwdOk": true, "root": canonDir(cwd), "dir": outRel, "parent": parent,
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
