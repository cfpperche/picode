package server

// The two writes the Servers panel has: stop a listener PiCode started, and
// hide one the human does not want to see again (or show it again). Both are
// decided at action time, never from what a poll left on the screen: the port
// is re-derived, the process holding it is re-read, and the response's pid +
// start token must still match — a row that went stale can therefore never
// signal a process that happened to reuse the id.
//
// Stop only reaches processes that live inside a PiCode pane: the panel may
// end what this app launched, and nothing else on the machine. A port with no
// owner is readable, openable and hideable, never killable — the panel says
// so instead of guessing.
//
// A stop is a small destructive act, so it is visible afterwards: the write
// appends a devserver.stopped event (ADR-0048's event table is the audit log;
// the kill itself is not a store state change).

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"
)

// signalDevServerProcessFn is the seam a test replaces — an assertion never
// signals a real process. The signal itself lives in the two platform files,
// and its own test uses a child it started.
var signalDevServerProcessFn = signalDevServerProcess

func handleDevServerStop(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Port     int    `json:"port"`
			PID      int    `json:"pid"`
			StartKey string `json:"startKey"`
			Force    bool   `json:"force"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		owner, ok := devServerListenerOnPort(r.Context(), deps, req.Port)
		if !ok {
			writeErr(w, http.StatusConflict, "That port is not one PiCode started, or its process has already exited.")
			return
		}
		if owner.pid != req.PID || req.StartKey == "" || processStartToken(owner.pid) != req.StartKey {
			writeErr(w, http.StatusConflict, "That server changed. Refresh the list.")
			return
		}
		if err := signalDevServerProcessFn(owner.pid, req.Force); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		stopped := waitDevServerGone(r.Context(), owner.pid, owner.startKey, devServerStopWait)
		_ = deps.Store.AppendEvent("devserver.stopped", nil, nil, map[string]any{
			"port":    req.Port,
			"pid":     owner.pid,
			"tool":    owner.tool,
			"owner":   owner.name,
			"kind":    owner.kind,
			"forced":  req.Force,
			"stopped": stopped,
		})
		writeJSON(w, http.StatusOK, map[string]any{"stopped": stopped, "pid": owner.pid})
	}
}

func handleDevServerHide(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Port     int    `json:"port"`
			PID      int    `json:"pid"`
			StartKey string `json:"startKey"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		owner, ok := devServerListenerOnPort(r.Context(), deps, req.Port)
		if !ok || owner.pid != req.PID || req.StartKey == "" || processStartToken(owner.pid) != req.StartKey {
			writeErr(w, http.StatusConflict, "That server is already gone. Refresh the list.")
			return
		}
		hide, err := deps.Store.HideDevServer(req.Port, req.PID, req.StartKey, devServerOwnerLabel(owner), owner.tool)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		pruneDevServerHides(deps)
		writeJSON(w, http.StatusOK, map[string]any{"hideId": hide.ID})
	}
}

func handleDevServerUnhide(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			HideID int64 `json:"hideId"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		if err := deps.Store.UnhideDevServer(req.HideID); err != nil {
			writeErr(w, http.StatusNotFound, "That hidden server is already forgotten.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// devServerListenerOnPort answers who holds a port right now — the same walk
// the read does, with no probing and no cache, so an action can never be
// decided by facts a poll collected minutes ago. The second answer says
// PiCode can see the process at all: a port with an owner that is not inside
// a pane is not killable, by design.
func devServerListenerOnPort(ctx context.Context, deps Deps, port int) (devServerOwner, bool) {
	if port <= 0 {
		return devServerOwner{}, false
	}
	owners, _ := devServerOwnersOfDeps(ctx, deps)
	owner, ok := owners[port]
	if !ok || owner.pid <= 0 || owner.startKey == "" {
		return devServerOwner{}, false
	}
	return owner, true
}

// devServerOwnerLabel is how a stored hide remembers whose server it was, in
// the words the panel uses: `terminal "sidebar" · PiCode`.
func devServerOwnerLabel(owner devServerOwner) string {
	who := "not started here"
	if owner.name != "" {
		if owner.kind == "agent" {
			who = "agent \"" + owner.name + "\""
		} else {
			who = "terminal \"" + owner.name + "\""
		}
	}
	if owner.workspace != "" {
		return who + " · " + owner.workspace
	}
	return who
}

// waitDevServerGone waits for the signal to have taken effect, so the panel's
// "Stopping…" ends with the truth instead of a guess. A process that is gone
// or already reaped counts as gone.
func waitDevServerGone(ctx context.Context, pid int, startKey string, wait time.Duration) bool {
	deadline := time.Now().Add(wait)
	for {
		if devServerProcessGone(pid, startKey) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return devServerProcessGone(pid, startKey)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// devServerProcessGone reports whether the process behind a pid + start token
// has died: the token is unreadable (the pid is gone), it changed (the id was
// reused), or the process is a zombie its parent has not reaped yet.
func devServerProcessGone(pid int, startKey string) bool {
	token := processStartToken(pid)
	if token == "" || token != startKey {
		return true
	}
	return processZombie(pid)
}

// processZombie reads the state field of /proc/<pid>/stat — field 3, which
// sits right after the comm field's final ')'.
func processZombie(pid int) bool {
	if pid <= 0 {
		return true
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	raw := string(data)
	end := -1
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == ')' {
			end = i
			break
		}
	}
	if end < 0 || end+2 >= len(raw) {
		return false
	}
	return raw[end+2] == 'Z'
}

// pruneDevServerHides forgets the hides whose process is gone. The identity of
// a hide is (port, pid, start token): once the process is gone that identity
// can never match a listener again, so the row is dead weight — and a stale
// count would make the panel's "N hidden" line lie. One transaction, one
// event, and nothing at all when there is nothing to forget.
func pruneDevServerHides(deps Deps) {
	if deps.Store == nil {
		return
	}
	hides, err := deps.Store.ListDevServerHides()
	if err != nil {
		return
	}
	keep := make([]int64, 0, len(hides))
	for _, h := range hides {
		if !devServerProcessGone(h.PID, h.StartKey) {
			keep = append(keep, h.ID)
		}
	}
	if len(keep) == len(hides) {
		return
	}
	_, _ = deps.Store.PruneDevServerHides(keep)
}

// errDevServerStopUnsupported is the answer on a platform where the owner walk
// finds nothing, so no row ever offers Stop.
var errDevServerStopUnsupported = errors.New("Stopping a server is not supported on this platform.")
