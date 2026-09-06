package session

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptBoundsCaptureWithoutRewritingPiFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.jsonl")
	raw := `{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"capture","name":"any_tool","arguments":{}}]}}
{"type":"message","message":{"role":"toolResult","toolCallId":"capture","content":[{"type":"text","text":"kept"}],"details":{"verified":false,"preview":{"image":"https://example.com/private"}}}}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	events, err := Transcript(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Result["verified"] != false || events[0].Detail != "kept" {
		t.Fatalf("events: %+v", events)
	}
	p := events[0].Result["preview"].(map[string]any)
	if p["image"] != nil || p["unavailable"] != true {
		t.Fatalf("unsafe preview: %v", p)
	}
	stored, err := os.ReadFile(path)
	if err != nil || string(stored) != raw {
		t.Fatal("rewrote native session")
	}
}

func TestTranscriptAttachesSidecarFinalFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 2)), nil); err != nil {
		t.Fatal(err)
	}
	dataURI := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	entry := map[string]any{"toolCallId": "call_side", "image": dataURI, "ts": 42, "url": "http://127.0.0.1:9/page", "title": "P"}
	entryJSON, _ := json.Marshal(entry)
	path := filepath.Join(t.TempDir(), "sidecar.jsonl")
	// The final entry precedes its result row: order must not matter.
	raw := `{"type":"custom","customType":"browser-capture-final","data":` + string(entryJSON) + `}
{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_side","name":"agent_browser","arguments":{}}]}}
{"type":"message","message":{"role":"toolResult","toolCallId":"call_side","content":[{"type":"text","text":"done"}]}}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	events, err := Transcript(path)
	if err != nil {
		t.Fatal(err)
	}
	var tool *Event
	for i := range events {
		if events[i].Kind == "tool" && events[i].ID == "call_side" {
			tool = &events[i]
		}
	}
	if tool == nil {
		t.Fatal("tool event missing")
	}
	p, ok := tool.Result["preview"].(map[string]any)
	if !ok || p["image"] != dataURI {
		t.Fatalf("final frame not attached: %v", tool.Result)
	}
	if p["url"] != "http://127.0.0.1:9/page" {
		t.Fatalf("metadata lost: %v", p)
	}
}

func TestTranscriptSkipsUnboundedSidecarFrame(t *testing.T) {
	entry := map[string]any{"toolCallId": "call_bad", "image": "data:image/jpeg;base64,AAAA", "ts": 1}
	entryJSON, _ := json.Marshal(entry)
	path := filepath.Join(t.TempDir(), "sidecar.jsonl")
	raw := `{"type":"custom","customType":"browser-capture-final","data":` + string(entryJSON) + `}
{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"call_bad","name":"agent_browser","arguments":{}}]}}
{"type":"message","message":{"role":"toolResult","toolCallId":"call_bad","content":[{"type":"text","text":"done"}]}}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	events, err := Transcript(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range events {
		if events[i].Kind == "tool" && events[i].ID == "call_bad" {
			if _, has := events[i].Result["preview"]; has {
				t.Fatal("unbounded frame must not attach")
			}
		}
	}
}
