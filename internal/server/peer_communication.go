package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/store"
)

func registerPeerCommunication(mux Registrar, deps Deps) {
	// The MCP handler has mandatory scoped authentication even with auth off.
	mux.Handle(communication.Path, communication.Handler(deps.Store))
	mux.HandleFunc("GET /api/communication", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		owners, err := deps.Store.ListPeerOwners()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		peers, err := deps.Store.ListPeerConnections()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"owners": owners, "connections": peers, "endpoint": communication.Path})
	})
	mux.HandleFunc("POST /api/communication", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var in struct {
			Kind       string `json:"kind"`
			OwnerID    string `json:"ownerId"`
			SessionKey string `json:"sessionKey"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			writeErr(w, 400, "invalid connection request")
			return
		}
		p, secret, err := deps.Store.EnablePeer(in.Kind, in.OwnerID, in.SessionKey)
		if err != nil {
			status := statusForStore(err)
			if errors.Is(err, store.ErrPeerDenied) {
				status = 409
			}
			writeErr(w, status, err.Error())
			return
		}
		writeJSON(w, 201, map[string]any{"connection": p, "token": secret, "endpoint": communication.Path})
	})
	mux.HandleFunc("DELETE /api/communication/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.RevokePeer(r.PathValue("id")); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /api/communication/{id}/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		before := int64(0)
		var err error
		if raw := r.URL.Query().Get("before"); raw != "" {
			before, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || before < 0 {
				writeErr(w, 400, "invalid cursor")
				return
			}
		}
		messages, err := deps.Store.PeerHistory(r.PathValue("id"), before)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"messages": messages})
	})
}
