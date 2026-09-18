package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

const webappDecodeLimit = 64 << 10

func registerWebappRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/webapps", handleListWebapps(deps))
	mux.HandleFunc("POST /api/webapps", handleCreateWebapp(deps))
	mux.HandleFunc("POST /api/webapps/resolve", handleResolveWebapp(deps))
	mux.HandleFunc("GET /api/webapps/{id}/icon", handleWebappIcon(deps))
	mux.HandleFunc("PATCH /api/webapps/{id}", handlePatchWebapp(deps))
	mux.HandleFunc("DELETE /api/webapps/{id}", handleDeleteWebapp(deps))
}

type webappResolveOut struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	IconAvailable bool   `json:"iconAvailable"`
	PWA           bool   `json:"pwa"`
	StartURL      string `json:"startUrl,omitempty"`
	Scope         string `json:"scope,omitempty"`
	Display       string `json:"display,omitempty"`
	ThemeColor    string `json:"themeColor,omitempty"`
}

func writeWebappErr(w http.ResponseWriter, err error) {
	var dup store.DuplicateWebappError
	switch {
	case errors.As(err, &dup):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":    "A webapp for this URL already exists.",
			"reason":   "duplicate",
			"existing": dup.Existing,
		})
	default:
		writePinErr(w, err)
	}
}

func webappDecode(w http.ResponseWriter, r *http.Request, into any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, webappDecodeLimit)
	return decodeBody(w, r, into)
}

func handleListWebapps(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListWebapps()
		if err != nil {
			writeWebappErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func handleResolveWebapp(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		if !webappDecode(w, r, &req) {
			return
		}
		target, err := store.NormalizeWebappURL(req.URL)
		if err != nil {
			writeWebappErr(w, err)
			return
		}
		client := webappNewClient()
		page, err := webappFetchPage(r.Context(), client, target, webappBodyCap)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "site is not reachable: "+err.Error())
			return
		}
		if page.status >= 400 {
			writeErr(w, http.StatusBadGateway, fmt.Sprintf("site answered HTTP %d", page.status))
			return
		}
		id := webappLookupMetadata(r.Context(), client, page)
		name := id.Name
		if name == "" {
			name = webappDeriveName(page.finalURL, "")
		}
		if id.IconURL == "" {
			if u, ok := webappFindIconURL(page.finalURL, page.body); ok {
				id.IconURL = u
			}
		}
		iconAvail := false
		if id.IconURL != "" {
			if _, _, err := webappFetchIcon(r.Context(), client, id.IconURL); err == nil {
				iconAvail = true
			}
		}
		writeJSON(w, http.StatusOK, webappResolveOut{Name: name, URL: target, IconAvailable: iconAvail, PWA: id.PWA, StartURL: id.StartURL, Scope: id.Scope, Display: id.Display, ThemeColor: id.ThemeColor})
	}
}

func handleCreateWebapp(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL  string `json:"url"`
			Name string `json:"name"`
		}
		if !webappDecode(w, r, &req) {
			return
		}
		target, err := store.NormalizeWebappURL(req.URL)
		if err != nil {
			writeWebappErr(w, err)
			return
		}
		client := webappNewClient()
		page, err := webappFetchPage(r.Context(), client, target, webappBodyCap)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "site is not reachable; nothing was installed: "+err.Error())
			return
		}
		if page.status >= 400 {
			writeErr(w, http.StatusBadGateway, fmt.Sprintf("site answered HTTP %d; nothing was installed", page.status))
			return
		}
		if row, dupErr := deps.Store.GetWebappByURL(target); dupErr == nil {
			writeWebappErr(w, store.DuplicateWebappError{Existing: row.Webapp})
			return
		}
		id := webappLookupMetadata(r.Context(), client, page)
		name := id.Name
		if name == "" {
			name = webappDeriveName(page.finalURL, "")
		}
		if trimmed := strings.TrimSpace(req.Name); trimmed != "" {
			name = trimmed
		}
		if id.IconURL == "" {
			if u, ok := webappFindIconURL(page.finalURL, page.body); ok {
				id.IconURL = u
			}
		}
		var icon []byte
		iconMime := ""
		if id.IconURL != "" {
			if data, mime, ferr := webappFetchIcon(r.Context(), client, id.IconURL); ferr == nil {
				icon, iconMime = data, mime
			}
		}
		app, err := deps.Store.CreateWebapp(store.WebappInput{Name: name, URL: target, StartURL: id.StartURL, Scope: id.Scope, Display: id.Display, ThemeColor: id.ThemeColor, Icon: icon, IconMime: iconMime})
		if err != nil {
			writeWebappErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, app)
	}
}

func handlePatchWebapp(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if !webappDecode(w, r, &req) {
			return
		}
		app, err := deps.Store.UpdateWebappName(r.PathValue("id"), req.Name)
		if err != nil {
			writeWebappErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, app)
	}
}

func handleDeleteWebapp(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteWebapp(r.PathValue("id")); err != nil {
			writeWebappErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleWebappIcon(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		icon, mime, err := deps.Store.GetWebappIcon(r.PathValue("id"))
		if err != nil {
			writeWebappErr(w, err)
			return
		}
		if len(icon) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(icon)
	}
}
