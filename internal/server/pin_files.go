package server

import (
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/cfpperche/picode/internal/store"
)

var pinIDOK = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,78}[a-z0-9]$`)

func registerPinFiles(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/pins/{id}/files", handleUploadPinFile(deps))
	mux.HandleFunc("POST /api/pins/{id}/sketches", handleSavePinSketch(deps))
	mux.HandleFunc("GET /api/pins/{id}/files/{fid}", handleGetPinFileBytes(deps))
	mux.HandleFunc("GET /api/pins/{id}/files/{fid}/scene", handleGetPinScene(deps))
	mux.HandleFunc("DELETE /api/pins/{id}/files/{fid}", handleDeletePinFile(deps))
}

func pinDir(dataDir, pinID string) string {
	return filepath.Join(dataDir, "pins", pinID)
}

func pinFilePath(dataDir, pinID, fileID string) string {
	return filepath.Join(pinDir(dataDir, pinID), fileID)
}

func pinScenePath(dataDir, pinID, fileID string) string {
	return filepath.Join(pinDir(dataDir, pinID), fileID+".scene")
}

// writeFileAtomic lands data under path through a sibling temp file and a
// rename, so a crash mid-write never leaves a half sketch behind the id
// an existing row still points at.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

// sweepPinDirs removes attachment directories whose pin row is gone: a
// failed RemoveAll at delete time, or a restore of an older database over
// a newer pins/ tree, used to leave bytes behind forever. Runs at boot.
func sweepPinDirs(dataDir string, st *store.Store) {
	if dataDir == "" || st == nil {
		return
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, "pins"))
	if err != nil {
		return
	}
	ids, err := st.PinIDs()
	if err != nil {
		return
	}
	live := make(map[string]bool, len(ids))
	for _, id := range ids {
		live[id] = true
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() || live[e.Name()] || !pinIDOK.MatchString(e.Name()) {
			continue
		}
		if os.RemoveAll(pinDir(dataDir, e.Name())) == nil {
			removed++
		}
	}
	if removed > 0 {
		log.Printf("pins: removed %d orphan attachment director%s", removed, map[bool]string{true: "y", false: "ies"}[removed == 1])
	}
}

func handleUploadPinFile(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !pinIDOK.MatchString(id) {
			writeErr(w, http.StatusBadRequest, "invalid pin")
			return
		}
		if err := r.ParseMultipartForm(store.MaxPinFileSize + 1<<20); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid upload")
			return
		}
		fh, hdr, err := r.FormFile("file")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "file is required")
			return
		}
		defer fh.Close()
		data, err := io.ReadAll(io.LimitReader(fh, store.MaxPinFileSize+1))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "could not read file")
			return
		}
		mime := hdr.Header.Get("Content-Type")
		meta, err := deps.Store.AddPinFile(id, hdr.Filename, mime, int64(len(data)))
		if err != nil {
			writePinErr(w, err)
			return
		}
		if err := os.MkdirAll(pinDir(deps.DataDir, id), 0o755); err != nil {
			_ = deps.Store.DeletePinFile(id, meta.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := writeFileAtomic(pinFilePath(deps.DataDir, id, meta.ID), data); err != nil {
			_ = deps.Store.DeletePinFile(id, meta.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, meta)
	}
}

// contentDisposition is RFC 6266: a name that is not plain ASCII travels as
// filename*=UTF-8”… (mime.FormatMediaType does the encoding), so "café.png"
// downloads as café.png and not as a Go-quoted escape.
func contentDisposition(disp, name string) string {
	if v := mime.FormatMediaType(disp, map[string]string{"filename": name}); v != "" {
		return v
	}
	return disp
}

func handleGetPinFileBytes(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, fid := r.PathValue("id"), r.PathValue("fid")
		if !pinIDOK.MatchString(id) || !pinIDOK.MatchString(fid) {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		meta, err := deps.Store.GetPinFile(id, fid)
		if err != nil {
			writePinErr(w, err)
			return
		}
		b, err := os.ReadFile(pinFilePath(deps.DataDir, id, fid))
		if err != nil {
			writeErr(w, http.StatusNotFound, "file missing")
			return
		}
		if meta.Mime != "" {
			w.Header().Set("Content-Type", meta.Mime)
		}
		disp := "inline"
		if meta.Kind != "image" {
			disp = "attachment"
		}
		w.Header().Set("Content-Disposition", contentDisposition(disp, meta.Name))
		w.Header().Set("Cache-Control", "private, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

func handleDeletePinFile(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, fid := r.PathValue("id"), r.PathValue("fid")
		if !pinIDOK.MatchString(id) || !pinIDOK.MatchString(fid) {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := deps.Store.DeletePinFile(id, fid); err != nil {
			writePinErr(w, err)
			return
		}
		_ = os.Remove(pinFilePath(deps.DataDir, id, fid))
		_ = os.Remove(pinScenePath(deps.DataDir, id, fid))
		w.WriteHeader(http.StatusNoContent)
	}
}

func removePinDir(dataDir, pinID string) {
	if pinIDOK.MatchString(pinID) {
		_ = os.RemoveAll(pinDir(dataDir, pinID))
	}
}

var errFormTooBig = errors.New("too big")

// handleSavePinSketch stores a sketch: the scene (Excalidraw JSON, without
// the annotated picture's bytes — the browser strips them and rebuilds the
// background from baseFileId) and the PNG preview. Same order as upload:
// the row first, then the bytes atomically, and the row is rolled back if
// the bytes fail; an update writes both files before touching the row.
func handleSavePinSketch(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !pinIDOK.MatchString(id) {
			writeErr(w, http.StatusBadRequest, "invalid pin")
			return
		}
		if err := r.ParseMultipartForm(store.MaxPinFileSize + store.MaxPinSceneSize + 1<<20); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid upload")
			return
		}
		scene, err := readFormFile(r, "scene", store.MaxPinSceneSize)
		if errors.Is(err, errFormTooBig) {
			writeErr(w, http.StatusBadRequest, "sketch is too large (max 2 MB of drawing)")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, "scene is required")
			return
		}
		preview, err := readFormFile(r, "preview", store.MaxPinImageSize)
		if errors.Is(err, errFormTooBig) {
			writeErr(w, http.StatusBadRequest, "image too large (max 8 MB)")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, "preview is required")
			return
		}
		name := r.FormValue("name")
		source := r.FormValue("source")
		baseID := r.FormValue("baseFileId")
		fid := r.FormValue("id")
		if err := os.MkdirAll(pinDir(deps.DataDir, id), 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeBoth := func(fileID string) error {
			if err := writeFileAtomic(pinFilePath(deps.DataDir, id, fileID), preview); err != nil {
				return err
			}
			return writeFileAtomic(pinScenePath(deps.DataDir, id, fileID), scene)
		}
		if fid != "" {
			if !pinIDOK.MatchString(fid) {
				writeErr(w, http.StatusBadRequest, "invalid id")
				return
			}
			if cur, err := deps.Store.GetPinFile(id, fid); err != nil || cur.Kind != "sketch" {
				writeErr(w, http.StatusNotFound, "sketch not found")
				return
			}
			if err := writeBoth(fid); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			meta, err := deps.Store.UpdatePinSketch(id, fid, name, int64(len(preview)))
			if err != nil {
				writePinErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, meta)
			return
		}
		meta, err := deps.Store.AddPinSketch(id, name, source, baseID, int64(len(preview)))
		if err != nil {
			writePinErr(w, err)
			return
		}
		if err := writeBoth(meta.ID); err != nil {
			_ = deps.Store.DeletePinFile(id, meta.ID)
			_ = os.Remove(pinFilePath(deps.DataDir, id, meta.ID))
			_ = os.Remove(pinScenePath(deps.DataDir, id, meta.ID))
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, meta)
	}
}

func handleGetPinScene(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, fid := r.PathValue("id"), r.PathValue("fid")
		if !pinIDOK.MatchString(id) || !pinIDOK.MatchString(fid) {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		meta, err := deps.Store.GetPinFile(id, fid)
		if err != nil {
			writePinErr(w, err)
			return
		}
		if meta.Kind != "sketch" {
			writeErr(w, http.StatusBadRequest, "not a sketch")
			return
		}
		b, err := os.ReadFile(pinScenePath(deps.DataDir, id, fid))
		if err != nil {
			writeErr(w, http.StatusNotFound, "scene missing")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

func readFormFile(r *http.Request, field string, max int) ([]byte, error) {
	fh, _, err := r.FormFile(field)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	b, err := io.ReadAll(io.LimitReader(fh, int64(max)+1))
	if err != nil {
		return nil, err
	}
	if len(b) > max {
		return nil, errFormTooBig
	}
	return b, nil
}
