// Browser surface stream proxy (ADR-0114): pumps the agent-browser
// session's loopback stream WebSocket to an authenticated client and back.
// The engine's stream is loopback-only by policy; this proxy is the only
// remote door, and it rides the same auth gate as every other route.
// Phase 1 is read-only: frames, url, status, console and tabs flow out;
// only pacing control (config/ack) flows in. Input events are refused.

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// BrowserStreamResolver reports the agent's engine stream port. A seam for
// tests; production uses ManagedAgent.BrowserStreamPort (the rendezvous
// discovery with the sidecar's safety checks).
type BrowserStreamResolver func(ctx context.Context, agentID string) (port int, ok bool)

// engine->client message types forwarded verbatim. The whitelist keeps a
// future engine message type from reaching clients unreviewed.
var browserForwardTypes = map[string]bool{
	"frame":   true,
	"url":     true,
	"status":  true,
	"console": true,
	"tabs":    true,
	"tab":     true,
}

// client->engine message types allowed in the read-only phase: pacing only.
var browserClientTypes = map[string]bool{
	"config": true,
	"ack":    true,
}

func defaultBrowserStreamResolver(deps Deps) BrowserStreamResolver {
	return func(ctx context.Context, agentID string) (int, bool) {
		if deps.Runtime == nil {
			return 0, false
		}
		ma := deps.Runtime.Get(agentID)
		if ma == nil {
			return 0, false
		}
		return ma.BrowserStreamPort(ctx)
	}
}

// browserWS upgrades, resolves the engine stream, and pumps both ways until
// either side closes. Delivery mirrors the engine's own contract: frames
// are latest-wins, control messages ride an ordered channel, and the
// renderer's acks are forwarded so pacing bounds the whole path, not one hop.
func browserWS(deps Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID := r.URL.Query().Get("agent")
		resolve := deps.BrowserStream
		if resolve == nil {
			resolve = defaultBrowserStreamResolver(deps)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		port, ok := resolve(ctx, agentID)
		cancel()

		ws, err := upgraderUpgrade(w, r)
		if err != nil {
			return
		}
		defer ws.Close()

		if !ok {
			writeWSJSON(ws, map[string]any{"type": "status", "connected": false, "reason": "no-browser"})
			return
		}
		engine, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/", port), nil)
		if err != nil {
			writeWSJSON(ws, map[string]any{"type": "status", "connected": false, "reason": "engine-unreachable"})
			return
		}
		defer engine.Close()

		done := make(chan struct{})

		// engine -> client: text messages, whitelisted by type, verbatim bytes.
		go func() {
			defer close(done)
			for {
				msgType, data, err := engine.ReadMessage()
				if err != nil {
					return
				}
				if msgType != websocket.TextMessage || !browserForwardable(data) {
					continue
				}
				if !writeWSRaw(ws, data) {
					return
				}
			}
		}()
		// When the engine side ends, break the client read loop too — the
		// surface must learn the stream died, not hang on a dead pipe.
		go func() {
			<-done
			ws.Close()
		}()

		// client -> engine: pacing control only. Anything else gets a
		// read-only refusal the UI can show, without tearing down the pipe.
		for {
			msgType, data, err := ws.ReadMessage()
			if err != nil {
				break
			}
			if msgType != websocket.TextMessage {
				continue
			}
			if !browserClientAllowed(data) {
				writeWSJSON(ws, map[string]any{
					"type":    "error",
					"code":    "read-only",
					"message": "This browser view is watch-only; interactive control arrives in a later phase.",
				})
				continue
			}
			if !writeWSRaw(engine, data) {
				break
			}
		}
		engine.Close()
		<-done
	})
}

// browserForwardable whitelists engine message types before fan-out.
func browserForwardable(data []byte) bool {
	var probe struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(data, &probe) != nil {
		return false
	}
	return browserForwardTypes[probe.Type]
}

// browserClientAllowed whitelists client message types (pacing only).
func browserClientAllowed(data []byte) bool {
	var probe struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(data, &probe) != nil {
		return false
	}
	return browserClientTypes[probe.Type]
}
