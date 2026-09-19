package server

// The browser annotation endpoints (v2c). An annotation is staged where the
// agent can read it: a small markdown note plus the crop, both under
// `<terminal cwd>/.picode/drop/` — the folder the prompt door already reads
// (2026-09-06 attach study: paths, never bytes). The store keeps the pointer;
// delivery to the agent is the existing prompt door with those paths, not a
// new input path. The cwd is resolved from the terminal, never taken from the
// request, so a caller cannot stage a file outside a workspace.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

const (
	annotationImageCap = 4 << 20 // 4 MB, the attach study's cap
	annotationTextCap  = 64 << 10
)

func registerBrowserAnnotationRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/browser/annotations", handleBrowserAnnotationCreate(deps))
	mux.HandleFunc("GET /api/browser/annotations", handleBrowserAnnotationList(deps))
	mux.HandleFunc("DELETE /api/browser/annotations/{id}", handleBrowserAnnotationDelete(deps))
}

// clipText keeps a capture inside the cap and says so, rather than losing the
// whole annotation to one huge DOM.
func clipText(s string) string {
	if len(s) <= annotationTextCap {
		return s
	}
	return s[:annotationTextCap] + "\n…[truncated]"
}

func handleBrowserAnnotationCreate(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			TerminalID string `json:"terminalId"`
			URL        string `json:"url"`
			Title      string `json:"title"`
			Selector   string `json:"selector"`
			Comment    string `json:"comment"`
			DOM        string `json:"dom"`
			CSS        string `json:"css"`
			Image      string `json:"image"` // base64 PNG, no data: prefix
		}
		if r.Body != nil {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, annotationImageCap+4<<20)).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid annotation body")
				return
			}
		}
		var image []byte
		if strings.TrimSpace(req.Image) != "" {
			var err error
			image, err = base64.StdEncoding.DecodeString(strings.TrimSpace(req.Image))
			if err != nil {
				writeErr(w, http.StatusBadRequest, "the screenshot is not valid base64")
				return
			}
			if len(image) > annotationImageCap {
				writeErr(w, http.StatusRequestEntityTooLarge, "the screenshot is over 4 MB")
				return
			}
		}
		terminals, err := deps.Store.ListTerminals()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var cwd, workspaceID string
		for _, t := range terminals {
			if t.ID == req.TerminalID {
				cwd, workspaceID = t.Cwd, t.WorkspaceID
				break
			}
		}
		if cwd == "" {
			writeErr(w, http.StatusBadRequest, "unknown terminal — an annotation is staged in the terminal's own folder")
			return
		}
		now := time.Now().UTC()
		base := "annotation-" + now.Format("20060102-150405") + "-" + strconv.FormatInt(now.UnixNano()%100000, 10)
		dir := filepath.Join(cwd, ".picode", "drop")
		shot, note := "", ""
		if len(image) > 0 {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				writeErr(w, http.StatusInternalServerError, "staging folder: "+err.Error())
				return
			}
			shot = base + ".png"
			if err := os.WriteFile(filepath.Join(dir, shot), image, 0o644); err != nil {
				writeErr(w, http.StatusInternalServerError, "screenshot: "+err.Error())
				return
			}
		}
		body := annotationNote(req.URL, req.Title, req.Selector, req.Comment, req.DOM, req.CSS, shot)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, "staging folder: "+err.Error())
			return
		}
		note = base + ".md"
		if err := os.WriteFile(filepath.Join(dir, note), []byte(body), 0o644); err != nil {
			writeErr(w, http.StatusInternalServerError, "note: "+err.Error())
			return
		}
		row, err := deps.Store.CreateBrowserAnnotation(store.BrowserAnnotation{
			TerminalID:  req.TerminalID,
			WorkspaceID: workspaceID,
			URL:         req.URL,
			Title:       req.Title,
			Selector:    req.Selector,
			Comment:     req.Comment,
			DOM:         clipText(req.DOM),
			CSS:         clipText(req.CSS),
			Shot:        shot,
			Note:        note,
		})
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		paths := []string{filepath.Join(dir, note)}
		if shot != "" {
			paths = append(paths, filepath.Join(dir, shot))
		}
		writeJSON(w, http.StatusOK, map[string]any{"annotation": row, "paths": paths})
	}
}

func handleBrowserAnnotationList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 50
		}
		rows, err := deps.Store.ListBrowserAnnotations(limit, r.URL.Query().Get("q"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"annotations": rows})
	}
}

func handleBrowserAnnotationDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := deps.Store.DeleteBrowserAnnotation(id); err != nil {
			writeErr(w, http.StatusNotFound, "no such annotation")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// annotationNote is what the agent reads: the human's sentence first, then the
// page, the element and the captures — a tiny document, not a data dump.
func annotationNote(url, title, selector, comment, dom, css, shot string) string {
	var b strings.Builder
	if strings.TrimSpace(comment) != "" {
		b.WriteString(strings.TrimSpace(comment) + "\n\n")
	} else {
		b.WriteString("Annotated element (no comment written).\n\n")
	}
	fmt.Fprintf(&b, "Page: %s\n", strings.TrimSpace(url))
	if strings.TrimSpace(title) != "" {
		fmt.Fprintf(&b, "Title: %s\n", strings.TrimSpace(title))
	}
	if strings.TrimSpace(selector) != "" {
		fmt.Fprintf(&b, "Element: %s\n", strings.TrimSpace(selector))
	}
	if shot != "" {
		fmt.Fprintf(&b, "Screenshot: %s (same folder)\n", shot)
	}
	if strings.TrimSpace(dom) != "" {
		fmt.Fprintf(&b, "\nHTML:\n```html\n%s\n```\n", clipText(strings.TrimSpace(dom)))
	}
	if strings.TrimSpace(css) != "" {
		// "Styles", not "computed styles": the block carries what the page had
		// AND — after a /* proposed */ marker — what the human wants instead
		// (the style inspector, v2c step 5).
		fmt.Fprintf(&b, "\nStyles:\n```css\n%s\n```\n", clipText(strings.TrimSpace(css)))
	}
	return b.String()
}
