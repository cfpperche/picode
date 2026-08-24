package server

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var agentUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // localhost-first (see term bridge)
}

func upgraderUpgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return agentUpgrader.Upgrade(w, r, nil)
}

func writeWSJSON(ws *websocket.Conn, v any) bool {
	return writeWSRaw2(ws, v)
}

func writeWSRaw(ws *websocket.Conn, msg []byte) bool {
	_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return ws.WriteMessage(websocket.TextMessage, msg) == nil
}

func writeWSRaw2(ws *websocket.Conn, v any) bool {
	_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return ws.WriteJSON(v) == nil
}
