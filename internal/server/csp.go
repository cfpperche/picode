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
// and inline style attributes (React).
//
// The policy rides the HTML responses only; assets and API answers carry
// none, so a same-origin worker (excalidraw's font subsetter uses `new
// Function`) keeps its own, unrestricted, context.

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

var cspOnce struct {
	sync.Once
	hashes string
}

// inlineScriptHashes lists 'sha256-…' sources for every inline <script>
// in index.html. Embedded builds compute once; a disk build (the UI can
// be rebuilt under a running daemon) recomputes per call.
func inlineScriptHashes() string {
	if web.Embedded() {
		cspOnce.Do(func() { cspOnce.hashes = hashInline(readIndex()) })
		return cspOnce.hashes
	}
	return hashInline(readIndex())
}

func readIndex() []byte {
	b, err := fs.ReadFile(web.UI(), "index.html")
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

// appCSP is the app shell's policy for the host the browser used, so
// WebSockets to this same server pass in every browser.
func appCSP(host string) string {
	ws := ""
	if h := strings.TrimSpace(host); h != "" {
		ws = " ws://" + h + " wss://" + h
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'wasm-unsafe-eval' " + inlineScriptHashes(),
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob: https:",
		"font-src 'self' data:",
		"connect-src 'self'" + ws,
		"media-src 'self' blob: data:",
		"worker-src 'self' blob:",
		"frame-src 'self'",
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
		if p == "/" || p == "/index.html" || strings.HasSuffix(p, ".html") {
			w.Header().Set("Content-Security-Policy", appCSP(r.Host))
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}
