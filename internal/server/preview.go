package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/preview"
)

// HTML file preview (ADR-0136): an authenticated mint hands the pane a
// short-lived capability ticket, and the ticket-bearing route serves the
// document and its relative assets to an opaque-origin sandbox. The route is
// deliberately outside /api: the sandbox sends no cookie (measured), so the
// token — not the session — is the gate. The session binding is re-checked
// here on every request so revoking a device kills its previews too.
func registerPreviewRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/previews", handlePreviewMint(deps))
	// Method-less on purpose: a method-scoped pattern would let a POST fall
	// through to the UI handler instead of answering 405 here.
	mux.HandleFunc("/preview/{token}/{path...}", handlePreviewServe(deps))
	mux.HandleFunc("/preview/", func(w http.ResponseWriter, _ *http.Request) { previewDeny(w) })
}

type previewMintRequest struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Path string `json:"path"`
	Root string `json:"root"`
}

func handlePreviewMint(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Previews == nil {
			writeErr(w, http.StatusServiceUnavailable, "previews are not available on this server")
			return
		}
		var req previewMintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		cwd, ok := previewOwnerCwd(deps, w, r, req.Kind, req.ID)
		if !ok {
			return
		}
		cwd, ok = resolveGitWorktree(w, r, cwd)
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		// The body's root is the same precondition as the query's (ADR-0074):
		// the pane sends it in the JSON body, the tree's equality check stands.
		if req.Root != "" && req.Root != canonDir(cwd) {
			writeErr(w, http.StatusConflict, "This folder changed. Refresh the file tree.")
			return
		}
		root := canonDir(cwd)
		if root == "" {
			writeErr(w, http.StatusBadRequest, "that folder has no path")
			return
		}
		rel, code, err := previewDocument(root, req.Path)
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		sessionID := ""
		if p := auth.From(r); p != nil {
			sessionID = p.Session.ID
		}
		tk, err := deps.Previews.Mint(req.Kind, req.ID, root, rel, sessionID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "could not start this preview")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"url":       "/preview/" + tk.Token + "/" + url.PathEscape(tk.Path),
			"path":      tk.Path,
			"expiresAt": tk.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
}

// previewOwnerCwd resolves the same directory the owner's file routes read
// through: agent cwd, live terminal cwd, or the workspace folder.
func previewOwnerCwd(deps Deps, w http.ResponseWriter, r *http.Request, kind, id string) (string, bool) {
	if deps.Store == nil {
		writeErr(w, http.StatusServiceUnavailable, "the store is not available")
		return "", false
	}
	switch kind {
	case "agent":
		cwd, err := agentCwd(deps, id)
		if err != nil {
			writeStoreErr(w, err)
			return "", false
		}
		return cwd, true
	case "term":
		term, err := deps.Store.GetTerminal(id)
		if err != nil {
			writeStoreErr(w, err)
			return "", false
		}
		return liveTermCwd(deps, r, term), true
	case "workspace":
		return workspaceFilesCwd(deps, w, id)
	default:
		writeErr(w, http.StatusBadRequest, "kind must be agent, term or workspace")
		return "", false
	}
}

// previewDocument validates the one file a ticket may be minted for and
// answers its slash-relative path.
func previewDocument(root, rel string) (string, int, error) {
	if strings.TrimSpace(rel) == "" {
		return "", http.StatusBadRequest, errors.New("path is required")
	}
	abs, outRel, err := relUnderCwd(root, rel)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	if preview.Hidden(outRel) {
		return "", http.StatusNotFound, errors.New("that file is not available")
	}
	if !preview.IsDocument(outRel) {
		return "", http.StatusBadRequest, errors.New("only .html and .htm files preview")
	}
	// Stat before resolving symlinks so a deleted file reads as gone (the
	// pane's own wording), not as some generic refusal.
	if st, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			return "", http.StatusNotFound, errors.New("that file is gone")
		}
		return "", http.StatusBadRequest, err
	} else if st.IsDir() {
		return "", http.StatusBadRequest, errors.New("that's a folder")
	}
	resolved, err := previewWithin(root, abs)
	if err != nil {
		return "", http.StatusNotFound, errors.New("that file is not available")
	}
	st, err := os.Stat(resolved)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	if st.IsDir() {
		return "", http.StatusBadRequest, errors.New("that's a folder")
	}
	if !st.Mode().IsRegular() {
		return "", http.StatusBadRequest, errors.New("that file is not available")
	}
	if st.Size() > maxAgentBlob {
		return "", http.StatusRequestEntityTooLarge, errors.New("this file is too large")
	}
	return outRel, http.StatusOK, nil
}

// previewWithin resolves symlinks and refuses anything that lands outside
// the ticket root, so a symlink inside the project cannot alias a secret.
func previewWithin(root, abs string) (string, error) {
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("preview: path escapes the root")
	}
	return resolved, nil
}

type previewAsset struct {
	abs  string
	mime string
	info os.FileInfo
}

// previewAssetFor answers the file a request really serves: containment and
// the MIME allowlist first, then a directory's own index.html.
func previewAssetFor(root, rel string) (previewAsset, error) {
	if strings.TrimSpace(rel) == "" {
		return previewAsset{}, errors.New("path is required")
	}
	abs, outRel, err := relUnderCwd(root, rel)
	if err != nil {
		return previewAsset{}, err
	}
	if preview.Hidden(outRel) {
		return previewAsset{}, errors.New("hidden")
	}
	resolved, err := previewWithin(root, abs)
	if err != nil {
		return previewAsset{}, err
	}
	st, err := os.Stat(resolved)
	if err != nil {
		return previewAsset{}, err
	}
	if st.IsDir() {
		resolved, err = previewWithin(root, filepath.Join(abs, "index.html"))
		if err != nil {
			return previewAsset{}, err
		}
		if st, err = os.Stat(resolved); err != nil {
			return previewAsset{}, err
		}
	}
	if !st.Mode().IsRegular() || st.Size() > maxAgentBlob {
		return previewAsset{}, errors.New("not a servable file")
	}
	mime, ok := preview.MIMEType(resolved)
	if !ok {
		return previewAsset{}, errors.New("not a servable type")
	}
	return previewAsset{abs: resolved, mime: mime, info: st}, nil
}

func handlePreviewServe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			previewDenyStatus(w, http.StatusMethodNotAllowed)
			return
		}
		if deps.Previews == nil {
			previewDeny(w)
			return
		}
		tk, ok := deps.Previews.Get(r.PathValue("token"))
		if !ok || !previewSessionLive(deps, tk.SessionID) {
			previewDeny(w)
			return
		}
		rel := r.PathValue("path")
		if rel == "" {
			rel = tk.Path
		}
		asset, err := previewAssetFor(tk.Root, rel)
		if err != nil {
			previewDeny(w)
			return
		}
		f, err := os.Open(asset.abs)
		if err != nil {
			previewDeny(w)
			return
		}
		defer func() { _ = f.Close() }()
		for k, v := range previewHeaders() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", asset.mime)
		w.Header().Set("ETag", fmt.Sprintf(`W/"%x-%x"`, asset.info.ModTime().UnixNano(), asset.info.Size()))
		http.ServeContent(w, r, filepath.Base(asset.abs), asset.info.ModTime(), f)
	}
}

// previewSessionLive keeps a ticket alive only while the session that asked
// for it lives. An empty id is anonymous mode (auth off), nothing to check.
func previewSessionLive(deps Deps, sessionID string) bool {
	if sessionID == "" || deps.Store == nil {
		return true
	}
	sess, err := deps.Store.SessionByID(sessionID)
	if err != nil || sess.RevokedAt != nil {
		return false
	}
	if sess.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339Nano, *sess.ExpiresAt); err == nil && time.Now().After(t) {
			return false
		}
	}
	return true
}

// previewSandbox is the one policy the preview cannot escape: an opaque
// origin (no cookie, no storage, no same-origin reads) with the fidelity
// flags the product chose. Never add allow-same-origin or top-navigation.
const previewSandbox = "sandbox allow-scripts allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-pointer-lock allow-downloads"

func previewHeaders() map[string]string {
	return map[string]string{
		"Content-Security-Policy":      previewSandbox + "; frame-ancestors 'self'",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
		"Cache-Control":                "no-store",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=(), usb=(), serial=(), hid=(), payment=()",
		"Access-Control-Allow-Origin":  "*",
		"Cross-Origin-Resource-Policy": "cross-origin",
	}
}

// previewDeny answers every refusal the same way: no oracle, no detail.
func previewDeny(w http.ResponseWriter) {
	previewDenyStatus(w, http.StatusNotFound)
}

func previewDenyStatus(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write([]byte("This preview is not available.\n"))
}
