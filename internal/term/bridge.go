// Package term bridges WebSocket connections to tmux sessions for the
// embedded terminal UI (ADR-0002 terminal channel).
//
// Protocol (per connection):
//   - Binary frames: raw terminal bytes, both directions.
//   - Text frames:   JSON control messages from the client, currently
//     {"type":"resize","cols":N,"rows":M}.
//
// The tmux session lives on the tmux server; each WebSocket attach spawns
// a short-lived `tmux attach` in a PTY. Closing the WebSocket (or the
// browser tab) ends only the attach — the agent keeps running.
//
// Shutdown contract (race-detector clean): exactly one goroutine owns pty
// writes + resize; each pump, on exit, unblocks its peer (pty close /
// ws close). The handler returns only after both pumps are done, so no
// fd or socket method runs after the handler scope ends.
package term

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"github.com/cfpperche/picode/internal/tmux"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
)

var upgrader = websocket.Upgrader{
	// Localhost-first tool (see architecture.md security model); the UI
	// is served from the same origin, and terminal-averse users won't be
	// blocked by origin quirks during local development.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Bridge returns the /ws/term handler. Query: ?session=<tmux session name>.
//
// resolve supplies the tmux options this session should be running with —
// each already carrying the scope to write it at — and is applied on every
// attach, so a setting changed while nobody was looking takes hold the next
// time the terminal is opened, and a session that predates the setting heals
// itself rather than staying odd forever. A nil resolve means PiCode manages
// no options.
func Bridge(tm *tmux.Manager, resolve func(session string) []tmux.ScopedValue) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("session")
		if !tmux.OwnedSessionName(name) {
			http.Error(w, "unknown or invalid session", http.StatusBadRequest)
			return
		}
		exists, err := tm.HasSession(r.Context(), name)
		if err != nil {
			http.Error(w, "tmux unavailable", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "session not running", http.StatusNotFound)
			return
		}

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return // Upgrade already wrote the HTTP error.
		}
		defer ws.Close()

		// Everything PiCode manages on the session — mouse, status bar,
		// passthrough, extended keys — comes through the resolver now
		// (ADR-0024): the old hardcoded forces are its *defaults*, so a user
		// override wins without a special case, and every attach re-applies,
		// healing sessions that predate a setting. What each option protects
		// is documented on the flag itself (internal/termopts).
		if resolve != nil {
			for _, sv := range resolve(name) {
				_ = tm.SetScopedOption(r.Context(), sv.Scope, name, sv.Key, sv.Value)
			}
		}

		// Initial size; the client sends a resize right after attach.
		cmd := exec.Command("tmux", "attach-session", "-t", "="+name)
		cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
		ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
		if err != nil {
			writeError(ws, "cannot attach to session: "+err.Error())
			return
		}
		defer func() { _ = ptyFile.Close() }() // second close is a harmless no-op

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		ptyDone := make(chan struct{}) // pty->ws pump finished
		wsDone := make(chan struct{})  // ws->pty pump finished

		// OWNER of pty writes and resize: the ws->pty pump. It is the only
		// goroutine that calls ptyFile.Write / pty.Setsize, and those calls
		// all happen before its deferred ptyFile.Close unblocks the reader.
		go func() {
			defer close(wsDone)
			defer func() { _ = ptyFile.Close() }() // unblock the pty reader below
			_ = ws.SetReadDeadline(time.Now().Add(pongWait))
			ws.SetPongHandler(func(string) error {
				return ws.SetReadDeadline(time.Now().Add(pongWait))
			})
			for {
				msgType, data, rerr := ws.ReadMessage()
				if rerr != nil {
					return
				}
				switch msgType {
				case websocket.TextMessage:
					handleControl(ptyFile, data)
				case websocket.BinaryMessage:
					if _, werr := ptyFile.Write(data); werr != nil {
						return
					}
				}
			}
		}()

		// pty -> websocket. On exit (pty closed by peer pump, or process
		// died) it closes the socket to unblock the ws->pty reader.
		go func() {
			defer close(ptyDone)
			defer ws.Close() // abrupt close unblocks ReadMessage; no close handshake needed
			buf := make([]byte, 32*1024)
			for {
				n, rerr := ptyFile.Read(buf)
				if n > 0 {
					_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
					if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
						return
					}
				}
				if rerr != nil {
					return
				}
			}
		}()

		// keepalive pings
		go func() {
			ticker := time.NewTicker(pingPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ptyDone:
					return
				case <-ticker.C:
					_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
					if perr := ws.WriteMessage(websocket.PingMessage, nil); perr != nil {
						return
					}
				}
			}
		}()

		<-ptyDone
		<-wsDone
	})
}

type controlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func handleControl(ptyFile *os.File, data []byte) {
	var msg controlMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.Type != "resize" || msg.Cols == 0 || msg.Rows == 0 {
		return
	}
	if err := pty.Setsize(ptyFile, &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols}); err != nil {
		log.Printf("term: resize %dx%d: %v", msg.Cols, msg.Rows, err)
	}
}

func writeError(ws *websocket.Conn, msg string) {
	_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
	_ = ws.WriteMessage(websocket.TextMessage,
		mustJSON(map[string]string{"type": "error", "message": msg}))
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error","message":"internal"}`)
	}
	return b
}
