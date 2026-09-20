package server

import (
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/cfpperche/picode/internal/web"
)

// Content-Security-Policy (ADR-0052 follow-up). The app shell carries one
// inline script — the theme bootstrap in index.html, which must run
// before the stylesheet to avoid a flash — so the policy names it by
// hash instead of allowing inline scripts. Everything else is same-origin:
// the Vite bundle, the service worker, the fonts. Exceptions are the ones
// the app actually uses: provider icons from unpkg (img), data/blob URLs
// for screenshots and recordings, WebAssembly (model-viewer, excalidraw),
// inline style attributes (React), and — on /desktop/ only — Tauri 2's
// IPC host so the Windows WebView2 can invoke native commands.
//
// The policy rides the HTML responses only; assets and API answers carry
// none, so a same-origin worker (excalidraw's font subsetter uses `new
// Function`) keeps its own, unrestricted, context.

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

var cspHashes sync.Map // request-path file → its 'sha256-…' sources

// inlineScriptHashes lists 'sha256-…' sources for every inline <script>
// in the HTML a request path serves. The path matters: the launcher at `/`
// has no inline script at all, while each shell's own index.html
// (`/browser/`, `/desktop/`, `/mobile/`) carries the theme bootstrap, and a
// hash taken from the wrong file is a policy that blocks the script it
// means to allow. Embedded builds compute once per file; a disk build (the
// UI can be rebuilt under a running daemon) recomputes per call.
func inlineScriptHashes(requestPath string) string {
	file := htmlFileFor(requestPath)
	if web.Embedded() {
		if h, ok := cspHashes.Load(file); ok {
			return h.(string)
		}
		h := hashInline(readUIHTML(file))
		cspHashes.Store(file, h)
		return h
	}
	return hashInline(readUIHTML(file))
}

// htmlFileFor maps a request path to the file in the UI bundle that serves
// it: a directory serves its own index.html, anything else serves itself.
func htmlFileFor(p string) string {
	t := strings.TrimPrefix(p, "/")
	if t == "" || strings.HasSuffix(t, "/") {
		return t + "index.html"
	}
	return t
}

func readUIHTML(file string) []byte {
	b, err := fs.ReadFile(web.UI(), file)
	if err != nil {
		return nil
	}
	return b
}

func hashInline(html []byte) string {
	var out []string
	for _, m := range inlineScript.FindAllSubmatch(html, -1) {
		sum := sha256.Sum256(m[1])
		out = append(out, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return strings.Join(out, " ")
}

// tauriIPCConnect is what Tauri 2's WebView2 fetch uses for plugin
// commands (plugin:window|is_maximized, plugin:event|listen, …). Without
// these, covering /desktop/ with the app policy blocks every invoke().
const tauriIPCConnect = " ipc: http://ipc.localhost https://ipc.localhost"

// desktopShellPath is the Windows Tauri shell (ADR-0120): the main window
// loads /desktop/, management loads /desktop/management.html. Browser and
// mobile shells never talk to ipc.localhost, so they keep the narrower
// connect-src.
func desktopShellPath(p string) bool {
	return p == "/desktop" || strings.HasPrefix(p, "/desktop/")
}

// appCSP is the app shell's policy for the host the browser used, so
// WebSockets to this same server pass in every browser, and for the request
// path, so the script hash names the file that path actually serves.
func appCSP(host, requestPath string) string {
	ws := ""
	if h := strings.TrimSpace(host); h != "" {
		ws = " ws://" + h + " wss://" + h
	}
	connect := "connect-src 'self'" + ws
	if desktopShellPath(requestPath) {
		connect += tauriIPCConnect
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'wasm-unsafe-eval' " + inlineScriptHashes(requestPath),
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob: https:",
		"font-src 'self' data:",
		connect,
		"media-src 'self' blob: data:",
		"worker-src 'self' blob:",
		// The work browser's frame fallback (dev-server preview) may show a page
		// served by this machine; anything else stays unframed, and the page is
		// a separate origin whatever it is.
		"frame-src 'self' http://localhost:* http://127.0.0.1:* https://localhost:* https://127.0.0.1:*",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'self'",
	}, "; ")
}

// PageCSP is the policy for the server-rendered pages (/pair and the
// gateway's own): inline styles, no scripts, nothing external, no framing.
const PageCSP = "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"

// securityHeaders wraps the UI handler: the app shell (and any HTML)
// gets the policy; hashed assets get nothing extra.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		// Previews carry their own policy (ADR-0136): the response's CSP
		// sandbox is what isolates a served document, and the app policy
		// would break the page instead.
		if strings.HasPrefix(p, "/preview/") {
			next.ServeHTTP(w, r)
			return
		}
		// The shells are served at directory URLs — the launcher sends every
		// viewer to /browser/ or /mobile/, the Windows shell loads /desktop/ —
		// so the policy has to cover those and not only "/" and "*.html":
		// without this the app in production ran with no policy at all, which
		// only looked harmless because nothing restricted it either.
		if p == "/" || strings.HasSuffix(p, "/") || strings.HasSuffix(p, ".html") {
			w.Header().Set("Content-Security-Policy", appCSP(r.Host, p))
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}
