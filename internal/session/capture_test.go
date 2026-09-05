package session

import (
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
