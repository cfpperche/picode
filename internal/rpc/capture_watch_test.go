package rpc

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/toolpreview"
)

func captureTestJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 2)), nil); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if toolpreview.EncodeJPEG(raw) == "" {
		t.Fatalf("fixture must be a bounded JPEG")
	}
	return raw
}

func TestCaptureWatchForwardsFramesAndFinal(t *testing.T) {
	ma := &ManagedAgent{AgentID: "a1", hub: NewHub(), done: make(chan struct{})}
	ma.capture = newCaptureState()
	dir := t.TempDir()
	w := &captureWatch{toolCallID: "call_1", dir: dir, startedAt: time.Now()}
	ma.capture.watches[w.toolCallID] = w

	sub, unsub := ma.hub.Subscribe()
	defer unsub()

	jpeg := captureTestJPEG(t)
	if err := os.WriteFile(filepath.Join(dir, "current.jpg"), jpeg, 0o600); err != nil {
		t.Fatal(err)
	}
	meta := map[string]any{"toolCallId": "call_1", "seq": 3, "ts": 1234, "final": false, "bytes": len(jpeg), "url": "http://127.0.0.1:9/x", "title": "T"}
	b, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, "current.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	ma.forwardCaptureFrame(ma.capture, w)

	select {
	case msg := <-sub:
		var env struct {
			AgentID string `json:"agentId"`
			Event   struct {
				Type       string `json:"type"`
				ToolCallID string `json:"toolCallId"`
				Seq        int    `json:"seq"`
				Final      bool   `json:"final"`
				Image      string `json:"image"`
			} `json:"event"`
		}
		if json.Unmarshal(msg, &env) != nil {
			t.Fatalf("bad envelope: %s", msg)
		}
		if env.AgentID != "a1" || env.Event.Type != "capture_frame" || env.Event.ToolCallID != "call_1" || env.Event.Seq != 3 || env.Event.Final {
			t.Fatalf("unexpected frame envelope: %+v", env)
		}
		if env.Event.Image == "" {
			t.Fatalf("frame image missing")
		}
	case <-time.After(time.Second):
		t.Fatal("no frame broadcast")
	}

	// Same seq again: no duplicate broadcast.
	ma.forwardCaptureFrame(ma.capture, w)
	select {
	case msg := <-sub:
		t.Fatalf("unexpected duplicate broadcast: %s", msg)
	case <-time.After(150 * time.Millisecond):
	}

	// Final marker re-sends the newest image flagged final, then retires.
	meta["final"] = true
	b, _ = json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, "current.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	ma.forwardCaptureFrame(ma.capture, w)
	select {
	case msg := <-sub:
		var env struct {
			Event struct {
				Final bool `json:"final"`
			} `json:"event"`
		}
		if json.Unmarshal(msg, &env) != nil || !env.Event.Final {
			t.Fatalf("expected final frame, got: %s", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("no final broadcast")
	}
	ma.capture.mu.Lock()
	finalSent := w.finalSent
	ma.capture.mu.Unlock()
	if !finalSent {
		t.Fatal("final marker must be recorded; the pump loop then retires the watch")
	}
}

func TestCaptureWatchBoundsRejectGarbage(t *testing.T) {
	ma := &ManagedAgent{AgentID: "a1", hub: NewHub(), done: make(chan struct{})}
	ma.capture = newCaptureState()
	dir := t.TempDir()
	w := &captureWatch{toolCallID: "call_2", dir: dir, startedAt: time.Now()}
	ma.capture.watches[w.toolCallID] = w
	sub, unsub := ma.hub.Subscribe()
	defer unsub()

	if err := os.WriteFile(filepath.Join(dir, "current.jpg"), []byte("not a jpeg"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"seq": 1, "final": false})
	if err := os.WriteFile(filepath.Join(dir, "current.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	ma.forwardCaptureFrame(ma.capture, w)
	select {
	case msg := <-sub:
		t.Fatalf("garbage frame must not broadcast: %s", msg)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestCaptureWatchArmsOncePerCall(t *testing.T) {
	ma := &ManagedAgent{AgentID: "a1", hub: NewHub(), done: make(chan struct{}), capture: newCaptureState()}
	dir := t.TempDir()
	ma.capture.sessionFile = dir + "/session.jsonl"
	ma.startCaptureWatch("call_x")
	ma.startCaptureWatch("call_x")
	ma.capture.mu.Lock()
	n := len(ma.capture.watches)
	ma.capture.mu.Unlock()
	if n != 1 {
		t.Fatalf("expected one watch, got %d", n)
	}
	ma.capture.mu.Lock()
	w := ma.capture.watches["call_x"]
	ma.capture.mu.Unlock()
	if w == nil || w.dir != ma.capture.sessionFile+".capture" {
		t.Fatalf("watch dir mismatch")
	}
}
