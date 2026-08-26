package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/backup"
)

func registerBackupRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/backup", handleBackupGet(deps))
	mux.HandleFunc("PUT /api/backup", handleBackupPut(deps))
	mux.HandleFunc("POST /api/backup/now", handleBackupNow(deps))
	mux.HandleFunc("GET /api/backup/snapshots", handleBackupList(deps))
	mux.HandleFunc("POST /api/backup/restore", handleBackupRestore(deps))
	mux.HandleFunc("POST /api/backup/reveal", handleBackupReveal(deps))
	mux.HandleFunc("DELETE /api/backup/snapshots/{id}", handleBackupDelete(deps))
}

func backupEngine(deps Deps) *backup.Engine {
	if deps.Backup != nil {
		return deps.Backup
	}
	return &backup.Engine{Store: deps.Store, DataDir: deps.DataDir}
}

func handleBackupGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s, err := backup.LoadSettings(deps.Store, deps.DataDir)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

func handleBackupPut(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in backup.Settings
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		in.Dir = strings.TrimSpace(in.Dir)
		if in.Dir != "" {
			if err := backup.ValidateDest(in.Dir, deps.DataDir, backup.PiHome()); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		if err := backup.SaveSettings(deps.Store, in); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		s, _ := backup.LoadSettings(deps.Store, deps.DataDir)
		writeJSON(w, http.StatusOK, s)
	}
}

func handleBackupNow(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := backup.LoadSettings(deps.Store, deps.DataDir)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if s.Dir == "" {
			writeErr(w, http.StatusBadRequest, "choose a backup folder first")
			return
		}
		eng := backupEngine(deps)
		snap, err := eng.Snapshot(s.Sessions, s.Secrets, s.Dir)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		_ = backup.Prune(s.Dir, s.KeepDays, time.Now().UTC())
		writeJSON(w, http.StatusOK, snap)
	}
}

func handleBackupList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := backup.LoadSettings(deps.Store, deps.DataDir)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		list, err := backup.List(s.Dir)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if list == nil {
			list = []backup.Snapshot{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"snapshots": list})
	}
}

func handleBackupRestore(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
			writeErr(w, http.StatusBadRequest, "id is required")
			return
		}
		s, err := backup.LoadSettings(deps.Store, deps.DataDir)
		if err != nil || s.Dir == "" {
			writeErr(w, http.StatusBadRequest, "no backup folder")
			return
		}
		schema, err := deps.Store.SchemaVersion()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agents, _ := deps.Store.ListAllAgents()
		for _, a := range agents {
			deps.stopAgent(r.Context(), a.ID)
		}
		if err := backupEngine(deps).Restore(s.Dir, req.ID, schema); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func handleBackupReveal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID   string `json:"id"`
			Root bool   `json:"root"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		s, err := backup.LoadSettings(deps.Store, deps.DataDir)
		if err != nil || s.Dir == "" {
			writeErr(w, http.StatusBadRequest, "no backup folder")
			return
		}
		path := backup.Root(s.Dir)
		if !req.Root && strings.TrimSpace(req.ID) != "" {
			list, err := backup.List(s.Dir)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			found := ""
			for _, snap := range list {
				if snap.ID == req.ID {
					found = snap.Path
					break
				}
			}
			if found == "" {
				writeErr(w, http.StatusNotFound, "snapshot not found")
				return
			}
			path = found
		}
		if err := backup.Reveal(path); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func handleBackupDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		s, err := backup.LoadSettings(deps.Store, deps.DataDir)
		if err != nil || s.Dir == "" {
			writeErr(w, http.StatusBadRequest, "no backup folder")
			return
		}
		if err := backup.Remove(s.Dir, id); err != nil {
			code := http.StatusBadRequest
			if strings.Contains(err.Error(), "not found") {
				code = http.StatusNotFound
			}
			writeErr(w, code, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
