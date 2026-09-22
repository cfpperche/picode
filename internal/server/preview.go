package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/preview"
)

// HTML file preview (ADR-0136): an authenticated mint hands the pane a
// short-lived capability ticket, and the ticket-bearing route serves the
// document and its relative assets. Two shapes share one ticket (ADR-0137):
// the sandboxed path form (/preview/<token>/<path>) and the ticket's own
// origin, http://<label>.localhost:<port>. The route is deliberately outside
// /api: the sandbox sends no cookie (measured), so the token — not the
// session — is the gate. The session binding is re-checked here on every
// request so revoking a device kills its previews too.
func registerPreviewRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/previews", handlePreviewMint(deps))
	// Method-less on purpose: a method-scoped pattern would let a POST fall
	// through to the UI handler instead of answering 405 here.
	mux.HandleFunc("/preview/{token}/{path...}", handlePreviewServe(deps))
	// The live-reload stream at the same ticket; more specific than the
	// wildcard above, so __events never shadows a file lookup.
	mux.HandleFunc("/preview/{token}/__events", handlePreviewEvents(deps))
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
		tk, err := deps.Previews.Mint(preview.Request{
			OwnerKind:   req.Kind,
			OwnerID:     req.ID,
			Root:        root,
			Path:        rel,
			SessionID:   sessionID,
			FrameOrigin: previewFrameOrigin(r),
		})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "could not start this preview")
			return
		}
		origin := previewOriginURL(deps, r, tk)
		out := map[string]any{
			"path":      tk.Path,
			"expiresAt": tk.ExpiresAt.UTC().Format(time.RFC3339),
			// The sandboxed path form always exists; the pane resolves it
			// against the UI origin so the Vite proxy keeps working in dev.
			"sandbox": map[string]string{
				"url":    "/preview/" + tk.Token + "/" + url.PathEscape(tk.Path),
				"events": "/preview/" + tk.Token + "/__events",
			},
		}
		if origin != "" {
			// The real origin (ADR-0137): only offered when the browser that
			// minted is on this machine's loopback, so a remote UI is never
			// handed a localhost URL it cannot reach.
			out["origin"] = map[string]string{
				"url":    origin + "/" + escapePath(tk.Path),
				"events": origin + "/__events",
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// previewFrameOrigin is the origin that asked for the ticket: the only one
// allowed to frame or read it. Empty for a non-browser mint (curl, tests).
func previewFrameOrigin(r *http.Request) string {
	o := strings.TrimSpace(r.Header.Get("Origin"))
	if o == "" {
		return ""
	}
	u, err := url.Parse(o)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// previewOriginURL answers the ticket's own origin, or "" when it must not
// be offered: the minting browser is not on loopback (a tunnel, a gateway, a
// LAN address — <label>.localhost would resolve on the wrong machine), or
// the daemon does not know a port to name.
func previewOriginURL(deps Deps, r *http.Request, tk preview.Ticket) string {
	if !loopbackHostname(r.Host) {
		return ""
	}
	port := 0
	if deps.PortSnapshot != nil {
		port = deps.PortSnapshot().Current
	}
	if port == 0 {
		// Tests and minimal embeddings: fall back to the port the request
		// arrived on, which is the listener's own.
		if _, p, err := net.SplitHostPort(r.Host); err == nil {
			port, _ = strconv.Atoi(p)
		}
	}
	if port <= 0 {
		return ""
	}
	return "http://" + tk.Label + ".localhost:" + strconv.Itoa(port)
}

// loopbackHostname reports a Host whose hostname is this machine's loopback:
// only there does `<label>.localhost` mean the same machine for the browser.
func loopbackHostname(host string) bool {
	h := host
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		h = hostname
	}
	h = strings.Trim(strings.ToLower(h), "[]")
	switch h {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0", "[::1]":
		return true
	}
	return false
}

// escapePath keeps a relative document path usable in a URL without letting
// a segment escape: each segment is escaped, separators stay.
func escapePath(p string) string {
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
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
//
// Both sides are resolved, which they were not: the file was canonicalised
// and compared against the root as written, so a root reached *through* a
// symlink made every file look like an escape. That is every path on macOS,
// where /var is /private/var — a workspace there served 404 for its own
// files (TestPreviewTicketDiesWithItsSession under a symlinked TMPDIR) — and
// any Linux host whose project sits under a linked mount or home. Comparing
// two canonical paths is strictly more correct than comparing one: nothing
// becomes reachable that resolved outside the root before.
func previewWithin(root, abs string) (string, error) {
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	// A root that cannot be resolved (deleted between mint and read) keeps
	// the raw spelling: the comparison below then refuses, which is the
	// answer a missing root deserves.
	if canonRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = canonRoot
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
		if deps.Previews == nil {
			previewDeny(w)
			return
		}
		tk, ok := deps.Previews.Get(r.PathValue("token"))
		if !ok || !previewSessionLive(deps, tk.SessionID) {
			previewDeny(w)
			return
		}
		switch r.Method {
		case http.MethodPut:
			handlePreviewOverlayPut(deps, tk, r.PathValue("path"), w, r)
			return
		case http.MethodGet, http.MethodHead:
		default:
			w.Header().Set("Allow", "GET, HEAD, PUT")
			previewDenyStatus(w, http.StatusMethodNotAllowed)
			return
		}
		servePreviewDocument(deps, tk, formSandbox, w, r, r.PathValue("path"))
	}
}

// previewForm is which of the two shapes is answering (ADR-0137): the
// sandboxed path form, or the ticket's own origin.
type previewForm int

const (
	formSandbox previewForm = iota
	formOrigin
)

// previewHostSuffix is the one host suffix a ticket's origin owns.
const previewHostSuffix = ".localhost"

// previewHostLabel answers the ticket label in a Host, when the Host is one:
// exactly `<label>.localhost[:port]`, one label, nothing nested.
func previewHostLabel(host string) (string, bool) {
	h := host
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		h = hostname
	}
	h = strings.ToLower(strings.TrimSpace(h))
	if !strings.HasSuffix(h, previewHostSuffix) {
		return "", false
	}
	label := strings.TrimSuffix(h, previewHostSuffix)
	if label == "" || strings.Contains(label, ".") || strings.Contains(label, ":") {
		return "", false
	}
	return label, true
}

// previewHostHandler serves a ticket's own origin (ADR-0137). It sits
// *outside* the auth gate on purpose: that origin is not the app, the ticket
// is its gate, and a preview request must never be judged by PiCode's
// session. Everything that is not a live ticket label — including the app
// shell and /api — is out of reach from it, and an unknown label is a plain
// 404, never a fall-through to the UI.
func previewHostHandler(deps Deps, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		label, ok := previewHostLabel(r.Host)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if deps.Previews == nil {
			previewDeny(w)
			return
		}
		tk, live := deps.Previews.ByLabel(label)
		if !live || !previewSessionLive(deps, tk.SessionID) {
			previewDeny(w)
			return
		}
		servePreviewOrigin(deps, tk, w, r)
	})
}

// servePreviewOrigin answers one request on a ticket's origin: the live-reload
// stream at /__events, the pane's CORS preflight and overlay writes, and the
// document and its assets at every other path. Every answer here carries the
// minting origin's CORS allowance — the pane's own calls are cross-origin on
// this form, and an answer without it is a bare network failure in the
// browser even when the work succeeded.
func servePreviewOrigin(deps Deps, tk preview.Ticket, w http.ResponseWriter, r *http.Request) {
	for k, v := range previewOriginCORS(tk) {
		w.Header().Set(k, v)
	}
	rel := strings.TrimPrefix(r.URL.Path, "/")
	if rel == "__events" {
		servePreviewEvents(deps, tk, formOrigin, w, r)
		return
	}
	switch r.Method {
	case http.MethodOptions:
		// The pane's PUT/DELETE are cross-origin here (pane and frame are
		// different origins), so the browser preflights them.
		w.Header().Set("Allow", "GET, HEAD, PUT, DELETE, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodPut:
		handlePreviewOverlayPut(deps, tk, rel, w, r)
		return
	case http.MethodDelete:
		handlePreviewOverlayClear(deps, tk, rel, w, r)
		return
	case http.MethodGet, http.MethodHead:
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT, DELETE, OPTIONS")
		previewDenyStatus(w, http.StatusMethodNotAllowed)
		return
	}
	servePreviewDocument(deps, tk, formOrigin, w, r, rel)
}

// servePreviewDocument answers the document, the pane's overlay standing in
// for it, or one of the page's relative assets. Every file the page pulls in
// joins the ticket's watch set: a live-reload stream stats exactly what this
// preview served.
func servePreviewDocument(deps Deps, tk preview.Ticket, form previewForm, w http.ResponseWriter, r *http.Request, rel string) {
	if rel == "" {
		rel = tk.Path
	}
	// The pane's unsaved buffer, when there is one, is the document.
	if text, ok := deps.Previews.Overlay(tk.Token); ok && previewIsDocument(tk, rel) {
		servePreviewOverlay(form, tk, w, r, text)
		return
	}
	asset, err := previewAssetFor(tk.Root, rel)
	if err != nil {
		previewDeny(w)
		return
	}
	deps.Previews.Touch(tk.Token, asset.abs)
	f, err := os.Open(asset.abs)
	if err != nil {
		previewDeny(w)
		return
	}
	defer func() { _ = f.Close() }()
	for k, v := range previewHeaders(form, tk) {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", asset.mime)
	w.Header().Set("ETag", fmt.Sprintf(`W/"%x-%x"`, asset.info.ModTime().UnixNano(), asset.info.Size()))
	http.ServeContent(w, r, filepath.Base(asset.abs), asset.info.ModTime(), f)
}

// previewWatchInterval is how often a live-reload stream stats the files
// its ticket served; a test shortens it.
var previewWatchInterval = time.Second

// handlePreviewEvents is the live-reload stream (ADR-0136 v1.5): the pane
// holds one EventSource per open preview, the daemon stats the files that
// ticket served, and a change emits one `change` frame. The page itself is
// never injected into and never asked for anything: the frame is reloaded
// by its parent, which keeps the sandbox intact. On the ticket's own origin
// the pane's EventSource is cross-origin (ADR-0137), so the stream carries
// the minting origin's CORS allowance.
func handlePreviewEvents(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Previews == nil {
			previewDeny(w)
			return
		}
		tk, ok := deps.Previews.Get(r.PathValue("token"))
		if !ok || !previewSessionLive(deps, tk.SessionID) {
			previewDeny(w)
			return
		}
		servePreviewEvents(deps, tk, formSandbox, w, r)
	}
}

func servePreviewEvents(deps Deps, tk preview.Ticket, form previewForm, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		previewDenyStatus(w, http.StatusMethodNotAllowed)
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		previewDenyStatus(w, http.StatusInternalServerError)
		return
	}
	h := w.Header()
	if form == formOrigin {
		for k, v := range previewOriginCORS(tk) {
			h.Set(k, v)
		}
	}
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusOK)
	write := func(event, data string) bool {
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return false
		}
		fl.Flush()
		return true
	}
	if !write("hello", `{}`) {
		return
	}

	// seen is this connection's snapshot: a path first seen now is seeded
	// without an event; a path whose mtime/size moved emits.
	seen := map[string]string{}
	changed := func() bool {
		hit := false
		for _, p := range deps.Previews.Watched(tk.Token) {
			sig := "gone"
			if st, err := os.Stat(p); err == nil {
				sig = fmt.Sprintf("%d-%d", st.ModTime().UnixNano(), st.Size())
			}
			if prev, ok := seen[p]; ok {
				if prev != sig {
					hit = true
				}
			}
			seen[p] = sig
		}
		return hit
	}
	changed() // seed

	poll := time.NewTicker(previewWatchInterval)
	defer poll.Stop()
	beat := time.NewTicker(eventsHeartbeat)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-poll.C:
			if changed() {
				if !write("change", `{}`) {
					return
				}
			}
		case <-beat.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			fl.Flush()
		}
	}
}

// previewIsDocument reports whether the request path is the document this
// ticket was minted for — the only file an overlay may stand in for.
func previewIsDocument(tk preview.Ticket, rel string) bool {
	_, outRel, err := relUnderCwd(tk.Root, rel)
	return err == nil && outRel == tk.Path
}

// servePreviewOverlay answers the document from the editor buffer instead of
// disk, under the same policy as any preview document.
func servePreviewOverlay(form previewForm, tk preview.Ticket, w http.ResponseWriter, r *http.Request, text string) {
	for k, v := range previewHeaders(form, tk) {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("ETag", fmt.Sprintf(`W/"overlay-%x"`, len(text)))
	http.ServeContent(w, r, "index.html", time.Time{}, strings.NewReader(text))
}

// handlePreviewOverlayPut takes the pane's unsaved editor buffer. The ticket
// is the gate and only the ticket's own document can be overlaid, so a
// sandboxed page replacing its own preview gains nothing it did not have.
func handlePreviewOverlayPut(deps Deps, tk preview.Ticket, rel string, w http.ResponseWriter, r *http.Request) {
	if !previewIsDocument(tk, rel) {
		previewDeny(w)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxAgentText))
	if err != nil {
		previewDenyStatus(w, http.StatusRequestEntityTooLarge)
		return
	}
	if bytes.IndexByte(body, 0) >= 0 || !utf8.Valid(body) {
		previewDenyStatus(w, http.StatusBadRequest)
		return
	}
	if !deps.Previews.SetOverlay(tk.Token, string(body)) {
		previewDeny(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePreviewOverlayClear drops the overlay so the file on disk serves
// again. It exists for the ticket's own origin: on Save the buffer and the
// file are equal, and clearing keeps the same origin — the page's storage
// survives the save (ADR-0137 D5) without an overlay masking later disk
// changes.
func handlePreviewOverlayClear(deps Deps, tk preview.Ticket, rel string, w http.ResponseWriter, r *http.Request) {
	if !previewIsDocument(tk, rel) {
		previewDeny(w)
		return
	}
	if !deps.Previews.ClearOverlay(tk.Token) {
		previewDeny(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// previewSandbox is the one policy the sandboxed form cannot escape: an
// opaque origin (no cookie, no storage, no same-origin reads) with the
// fidelity flags the product chose. Never add allow-same-origin or
// top-navigation.
const previewSandbox = "sandbox allow-scripts allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-pointer-lock allow-downloads"

// previewPermissions is the same feature lock in both forms.
const previewPermissions = "camera=(), microphone=(), geolocation=(), usb=(), serial=(), hid=(), payment=()"

// previewHeaders is the whole policy for one served preview. The sandboxed
// path form keeps the CSP sandbox (and `ACAO: *`, which an opaque-origin
// page needs for its own `fetch` and ES modules); the ticket's own origin
// (ADR-0137) drops it — that would re-impose an opaque origin — and names
// the minting UI as the only ancestor and the only origin allowed to read.
func previewHeaders(form previewForm, tk preview.Ticket) map[string]string {
	if form == formOrigin {
		h := map[string]string{
			"Referrer-Policy":              "no-referrer",
			"X-Content-Type-Options":       "nosniff",
			"Cache-Control":                "no-store",
			"Permissions-Policy":           previewPermissions,
			"Cross-Origin-Resource-Policy": "cross-origin",
		}
		for k, v := range previewOriginCORS(tk) {
			h[k] = v
		}
		if tk.FrameOrigin != "" {
			// Only the UI that minted it may frame it; a ticket minted by a
			// script (no Origin) claims no ancestor list rather than a wrong one.
			h["Content-Security-Policy"] = "frame-ancestors " + tk.FrameOrigin
		}
		return h
	}
	return map[string]string{
		"Content-Security-Policy":      previewSandbox + "; frame-ancestors 'self'",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
		"Cache-Control":                "no-store",
		"Permissions-Policy":           previewPermissions,
		"Access-Control-Allow-Origin":  "*",
		"Cross-Origin-Resource-Policy": "cross-origin",
	}
}

// previewOriginCORS is the origin form's CORS answer: exactly the UI that
// minted the ticket, since on that origin the pane's own HEAD/PUT/DELETE and
// its EventSource are all cross-origin.
func previewOriginCORS(tk preview.Ticket) map[string]string {
	if tk.FrameOrigin == "" {
		return nil
	}
	return map[string]string{
		"Access-Control-Allow-Origin":  tk.FrameOrigin,
		"Access-Control-Allow-Methods": "GET, HEAD, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
		"Access-Control-Max-Age":       "60",
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
