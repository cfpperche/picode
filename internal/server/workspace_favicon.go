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
const maxAppDirs = 24      // bound the apps/* listing on a giant monorepo

// faviconDirs is ordered: project root first, then the common build-tool
// spots, then a frontend living in a subfolder (web/public is PiCode's
// shape). apps/<name>/{public,app,src/app} is appended per workspace by
// faviconDirsFor — Next.js App Router in a turborepo lives at
// apps/web/app/icon.svg.
// Within each dir, vectors first; favicon.* of a type outranks icon.* of
// the same type, but icon.svg still outranks favicon.png / favicon.ico so
// a Next.js mark beats the stock create-next-app ico.
var faviconDirs = []string{
	"", "public", "static", "app", "src/app", "www", "docs",
	"web/public", "web/app", "web/src/app",
	"www/public", "ui/public", "frontend/public", "client/public",
}

var faviconNames = []string{
	"favicon.svg", "icon.svg",
	"favicon.png", "icon.png",
	"favicon.ico", "icon.ico",
}

func faviconRels(root string) []string {
	dirs := faviconDirsFor(root)
	out := make([]string, 0, len(dirs)*len(faviconNames))
	for _, d := range dirs {
		for _, n := range faviconNames {
			if d == "" {
				out = append(out, n)
			} else {
				out = append(out, d+"/"+n)
			}
		}
	}
	return out
}

func faviconDirsFor(root string) []string {
	dirs := append([]string{}, faviconDirs...)
	entries, err := os.ReadDir(filepath.Join(root, "apps"))
	if err != nil {
		return dirs
	}
	var web, other []string
	n := 0
	for _, e := range entries {
		if n >= maxAppDirs {
			break
		}
		if !e.IsDir() || !isAppDirName(e.Name()) {
			continue
		}
		n++
		name := e.Name()
		subs := []string{
			"apps/" + name + "/public",
			"apps/" + name + "/app",
			"apps/" + name + "/src/app",
		}
		if name == "web" {
			web = append(web, subs...)
		} else {
			other = append(other, subs...)
		}
	}
	return append(append(dirs, web...), other...)
}

func isAppDirName(name string) bool {
	if name == "" || name == "." || name == ".." || name == "node_modules" {
		return false
	}
	return !strings.HasPrefix(name, ".") && name == filepath.Base(name)
}

func findWorkspaceFavicon(root string) (abs string, ok bool) {
	for _, rel := range faviconRels(root) {
		cand, _, err := relUnderCwd(root, rel)
		if err != nil {
			continue
		}
		st, err := os.Stat(cand)
		if err != nil || st.IsDir() || st.Size() == 0 || st.Size() > maxFavicon {
			continue
		}
		return cand, true
	}
	return "", false
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
		abs, ok := findWorkspaceFavicon(wk.Path)
		if !ok {
			writeErr(w, http.StatusNotFound, "no favicon")
			return
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			writeErr(w, http.StatusNotFound, "no favicon")
			return
		}
		w.Header().Set("Content-Type", faviconMIME[strings.ToLower(filepath.Ext(abs))])
		w.Header().Set("Cache-Control", "private, max-age=300")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// An SVG from the project folder could script if navigated to
		// directly; sandboxed it stays a picture.
		w.Header().Set("Content-Security-Policy", "sandbox")
		_, _ = w.Write(b)
	}
}
