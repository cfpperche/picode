package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestAgentUIDecisionTable(t *testing.T) {
	ts := bashTestServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Ask", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add = %d", res.StatusCode)
	}
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	id := wk.Agent.ID

	// Row 8: stopped → no card / 409.
	stopped := postJSON(t, ts, "/api/agents/"+id+"/ui", map[string]any{"id": "ui-ask", "cancelled": true})
	if stopped.StatusCode != http.StatusConflict {
		t.Fatalf("stopped ui = %d", stopped.StatusCode)
	}

	start := postJSON(t, ts, "/api/agents/"+id+"/managed/start", map[string]string{})
	if start.StatusCode != http.StatusCreated && start.StatusCode != http.StatusOK {
		t.Fatalf("start = %d", start.StatusCode)
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/agent?agent=" + id
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws: %v", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(8 * time.Second))

	snap := readWSEvent(t, ws)
	if snap["type"] != "snapshot" || snap["waiting"] == true {
		t.Fatalf("snapshot = %v", snap)
	}

	// Row 6: notify is not waiting.
	enq := postJSON(t, ts, "/api/agents/"+id+"/tasks", map[string]string{"kind": "prompt", "payload": "ASK:notify", "source": "user"})
	if enq.StatusCode != http.StatusCreated && enq.StatusCode != http.StatusOK {
		t.Fatalf("enqueue notify = %d", enq.StatusCode)
	}
	note := waitWSType(t, ws, "extension_ui_request", 5*time.Second)
	if note["method"] != "notify" {
		t.Fatalf("notify event = %v", note)
	}

	// Row 2: confirm → waiting.
	enq2 := postJSON(t, ts, "/api/agents/"+id+"/tasks", map[string]string{"kind": "prompt", "payload": "ASK:confirm", "source": "user"})
	if enq2.StatusCode != http.StatusCreated && enq2.StatusCode != http.StatusOK {
		t.Fatalf("enqueue confirm = %d", enq2.StatusCode)
	}
	ask := waitWSType(t, ws, "extension_ui_request", 5*time.Second)
	if ask["method"] != "confirm" || ask["id"] != "ui-ask" {
		t.Fatalf("confirm event = %v", ask)
	}

	// Reconnect snapshot includes the dialog.
	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws2: %v", err)
	}
	_ = ws2.SetReadDeadline(time.Now().Add(3 * time.Second))
	snap2 := readWSEvent(t, ws2)
	ws2.Close()
	if snap2["waiting"] != true {
		t.Fatalf("waiting snapshot = %v", snap2)
	}

	// Row 3: Yes.
	yes := postJSON(t, ts, "/api/agents/"+id+"/ui", map[string]any{"id": "ui-ask", "confirmed": true})
	if yes.StatusCode != http.StatusOK {
		t.Fatalf("yes = %d", yes.StatusCode)
	}

	// Row 4: cancel on a fresh confirm.
	enq3 := postJSON(t, ts, "/api/agents/"+id+"/tasks", map[string]string{"kind": "prompt", "payload": "ASK:confirm", "source": "user"})
	if enq3.StatusCode != http.StatusCreated && enq3.StatusCode != http.StatusOK {
		t.Fatalf("enqueue confirm2 = %d", enq3.StatusCode)
	}
	_ = waitWSType(t, ws, "extension_ui_request", 5*time.Second)
	cancel := postJSON(t, ts, "/api/agents/"+id+"/ui", map[string]any{"id": "ui-ask", "cancelled": true})
	if cancel.StatusCode != http.StatusOK {
		t.Fatalf("cancel = %d", cancel.StatusCode)
	}

	gone := postJSON(t, ts, "/api/agents/"+id+"/ui", map[string]any{"id": "ui-ask", "cancelled": true})
	if gone.StatusCode != http.StatusConflict {
		t.Fatalf("second cancel = %d", gone.StatusCode)
	}

	missing := postJSON(t, ts, "/api/agents/ag_nope/ui", map[string]any{"id": "x", "cancelled": true})
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing agent = %d", missing.StatusCode)
	}
}

func readWSEvent(t *testing.T, ws *websocket.Conn) map[string]any {
	t.Helper()
	_, raw, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	var env struct {
		Event map[string]any `json:"event"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("ws json: %v %s", err, raw)
	}
	return env.Event
}

func waitWSType(t *testing.T, ws *websocket.Conn, typ string, d time.Duration) map[string]any {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(d))
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		ev := readWSEvent(t, ws)
		got, _ := ev["type"].(string)
		if got == typ {
			return ev
		}
	}
	t.Fatalf("ws never saw %s", typ)
	return nil
}
