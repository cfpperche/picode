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

// annotationItem is one pin of a Send: what the human pointed at, what they
// said, the evidence, and the crop's file name once it is staged.
type annotationItem struct {
	Selector string
	Comment  string
	DOM      string
	CSS      string
	Shot     string
}

func handleBrowserAnnotationCreate(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		// A Send is a SET: the whole set arrives in one call, one crop per
		// pin, and one note for all of them — the reference's "N annotations"
		// as a single context. How many pins a Send may carry is the human's
		// business, not the transport's (owner 2026-09-19); the only bounds
		// are the per-file size cap and the disk.
		var req struct {
			TerminalID string `json:"terminalId"`
			URL        string `json:"url"`
			Title      string `json:"title"`
			Items      []struct {
				Selector string `json:"selector"`
				Comment  string `json:"comment"`
				DOM      string `json:"dom"`
				CSS      string `json:"css"`
				Image    string `json:"image"` // base64 PNG, no data: prefix
			} `json:"items"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, (annotationImageCap+4<<20)*4)).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid annotation body")
				return
			}
		}
		if len(req.Items) == 0 {
			writeErr(w, http.StatusBadRequest, "an annotation needs at least one item")
			return
		}
		// Every image is decoded and checked BEFORE anything is written: a
		// Send that cannot be staged whole must not leave half a package on
		// disk and half in the store.
		images := make([][]byte, len(req.Items))
		for i, it := range req.Items {
			if strings.TrimSpace(it.Image) == "" {
				continue
			}
			raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(it.Image))
			if err != nil {
				writeErr(w, http.StatusBadRequest, fmt.Sprintf("the screenshot of item %d is not valid base64", i+1))
				return
			}
			if len(raw) > annotationImageCap {
				writeErr(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("the screenshot of item %d is over 4 MB", i+1))
				return
			}
			images[i] = raw
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
		if err := os.MkdirAll(dir, 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, "staging folder: "+err.Error())
			return
		}
		items := make([]annotationItem, len(req.Items))
		for i, it := range req.Items {
			items[i] = annotationItem{Selector: it.Selector, Comment: it.Comment, DOM: it.DOM, CSS: it.CSS}
			if len(images[i]) == 0 {
				continue
			}
			shot := fmt.Sprintf("%s-%d.png", base, i+1)
			if err := os.WriteFile(filepath.Join(dir, shot), images[i], 0o644); err != nil {
				writeErr(w, http.StatusInternalServerError, "screenshot: "+err.Error())
				return
			}
			items[i].Shot = shot
		}
		note := base + ".md"
		if err := os.WriteFile(filepath.Join(dir, note), []byte(annotationNote(req.URL, req.Title, items)), 0o644); err != nil {
			writeErr(w, http.StatusInternalServerError, "note: "+err.Error())
			return
		}
		rows := make([]store.BrowserAnnotation, 0, len(items))
		for _, it := range items {
			row, err := deps.Store.CreateBrowserAnnotation(store.BrowserAnnotation{
				TerminalID:  req.TerminalID,
				WorkspaceID: workspaceID,
				URL:         req.URL,
				Title:       req.Title,
				Selector:    it.Selector,
				Comment:     it.Comment,
				DOM:         clipText(it.DOM),
				CSS:         clipText(it.CSS),
				Shot:        it.Shot,
				Note:        note,
			})
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			rows = append(rows, row)
		}
		// The note first: it is the package, and the prompt door's own file
		// cap (ADR-0089) then decides how many crops ride along.
		paths := []string{filepath.Join(dir, note)}
		for _, it := range items {
			if it.Shot != "" {
				paths = append(paths, filepath.Join(dir, it.Shot))
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"annotations": rows, "paths": paths})
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

// annotationNote is what the agent reads: the whole Send as ONE document —
// the human's sentences, in order, each with its element, evidence and crop.
// It is per Send, not per pin, so the number of pins a human may pin is not
// bounded by how many notes a transport will carry (owner 2026-09-19).
func annotationNote(url, title string, items []annotationItem) string {
	var b strings.Builder
	if len(items) == 1 {
		fmt.Fprintf(&b, "1 annotation on %s\n", strings.TrimSpace(url))
	} else {
		fmt.Fprintf(&b, "%d annotations on %s\n", len(items), strings.TrimSpace(url))
	}
	if strings.TrimSpace(title) != "" {
		fmt.Fprintf(&b, "Title: %s\n", strings.TrimSpace(title))
	}
	for i, it := range items {
		comment := strings.TrimSpace(it.Comment)
		if comment == "" {
			comment = "Annotated element (no comment written)."
		}
		fmt.Fprintf(&b, "\n## %d. %s\n", i+1, strings.ReplaceAll(comment, "\n", " "))
		if strings.TrimSpace(it.Selector) != "" {
			fmt.Fprintf(&b, "Element: %s\n", strings.TrimSpace(it.Selector))
		}
		if it.Shot != "" {
			fmt.Fprintf(&b, "Screenshot: %s (same folder)\n", it.Shot)
		}
		if strings.TrimSpace(it.DOM) != "" {
			fmt.Fprintf(&b, "\nHTML:\n```html\n%s\n```\n", clipText(strings.TrimSpace(it.DOM)))
		}
		if strings.TrimSpace(it.CSS) != "" {
			// "Styles", not "computed styles": the block carries what the page
			// had AND — after a /* proposed */ marker — what the human wants
			// instead (the style inspector, v2c step 5).
			fmt.Fprintf(&b, "\nStyles:\n```css\n%s\n```\n", clipText(strings.TrimSpace(it.CSS)))
		}
	}
	return b.String()
}
