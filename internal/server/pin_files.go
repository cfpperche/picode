package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/cfpperche/picode/internal/store"
)

var pinIDOK = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,78}[a-z0-9]$`)

func registerPinFiles(mux *http.ServeMux, deps Deps) {
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
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if err := os.MkdirAll(pinDir(deps.DataDir, id), 0o755); err != nil {
			_ = deps.Store.DeletePinFile(id, meta.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(pinFilePath(deps.DataDir, id, meta.ID), data, 0o644); err != nil {
			_ = deps.Store.DeletePinFile(id, meta.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, meta)
	}
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
			writeErr(w, statusForStore(err), err.Error())
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
		w.Header().Set("Content-Disposition", disp+"; filename="+strconv.Quote(meta.Name))
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
			writeErr(w, statusForStore(err), err.Error())
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
		if err != nil {
			writeErr(w, http.StatusBadRequest, "scene is required")
			return
		}
		preview, err := readFormFile(r, "preview", store.MaxPinImageSize)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "preview is required")
			return
		}
		name := r.FormValue("name")
		source := r.FormValue("source")
		baseID := r.FormValue("baseFileId")
		fid := r.FormValue("id")
		var meta store.PinFile
		if fid != "" {
			if !pinIDOK.MatchString(fid) {
				writeErr(w, http.StatusBadRequest, "invalid id")
				return
			}
			meta, err = deps.Store.UpdatePinSketch(id, fid, name, int64(len(preview)))
		} else {
			meta, err = deps.Store.AddPinSketch(id, name, source, baseID, int64(len(preview)))
		}
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if err := os.MkdirAll(pinDir(deps.DataDir, id), 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(pinFilePath(deps.DataDir, id, meta.ID), preview, 0o644); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(pinScenePath(deps.DataDir, id, meta.ID), scene, 0o644); err != nil {
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
			writeErr(w, statusForStore(err), err.Error())
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
		return nil, io.ErrUnexpectedEOF
	}
	return b, nil
}
