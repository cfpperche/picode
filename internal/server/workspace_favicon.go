package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

const maxFavicon = 1 << 20 // favicons are tiny; anything bigger is not one

// faviconRels is ordered: project root first, then the common build-tool
// spots. Within each dir svg > png > ico — the 16px sidebar slot renders
// vectors crispest.
var faviconRels = []string{
	"favicon.svg", "favicon.png", "favicon.ico",
	"public/favicon.svg", "public/favicon.png", "public/favicon.ico",
	"static/favicon.svg", "static/favicon.png", "static/favicon.ico",
	"app/favicon.ico", "src/app/favicon.ico",
	"www/favicon.svg", "www/favicon.png", "www/favicon.ico",
	"docs/favicon.svg", "docs/favicon.png", "docs/favicon.ico",
}

// A local map, not imageMIME: that one feeds the composer's image attach,
// and .ico/.svg there would change that feature's contract.
var faviconMIME = map[string]string{
	".ico": "image/x-icon",
	".png": "image/png",
	".svg": "image/svg+xml",
}

// handleWorkspaceFavicon serves the project's favicon so the workspace card
// can wear it (ADR-0027). Read-only, confined to the workspace folder.
func handleWorkspaceFavicon(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, err := deps.Store.GetWorkspace(r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) || (err == nil && store.IsFree(wk)) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, rel := range faviconRels {
			abs, _, err := relUnderCwd(wk.Path, rel)
			if err != nil {
				continue
			}
			st, err := os.Stat(abs)
			if err != nil || st.IsDir() || st.Size() == 0 || st.Size() > maxFavicon {
				continue
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			w.Header().Set("Content-Type", faviconMIME[strings.ToLower(filepath.Ext(abs))])
			w.Header().Set("Cache-Control", "private, max-age=300")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// An SVG from the project folder could script if navigated to
			// directly; sandboxed it stays a picture.
			w.Header().Set("Content-Security-Policy", "sandbox")
			_, _ = w.Write(b)
			return
		}
		writeErr(w, http.StatusNotFound, "no favicon")
	}
}
