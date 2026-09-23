package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestProgressLineIsOneObject pins the wire format the shell streams: one
// JSON object per line, so the reader can tell a step from the final outcome
// by its keys without buffering the whole run.
func TestProgressLineIsOneObject(t *testing.T) {
	line := progressLine("converting the disk file to sparse")
	if !strings.HasPrefix(line, `{"progress":`) || !strings.HasSuffix(line, `}`) {
		t.Fatalf("got %q", line)
	}
	var v struct {
		Progress string `json:"progress"`
	}
	if err := json.Unmarshal([]byte(line), &v); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if v.Progress != "converting the disk file to sparse" {
		t.Errorf("progress = %q", v.Progress)
	}
	// A step name with a quote must survive as data, not break the object.
	weird := progressLine(`say "now"`)
	var w struct {
		Progress string `json:"progress"`
	}
	if err := json.Unmarshal([]byte(weird), &w); err != nil || w.Progress != `say "now"` {
		t.Errorf("quoted step: %q / %q / %v", weird, w.Progress, err)
	}
}

// TestStoppedTravelsOnTheWire pins the field the Management page reads to
// tell "failed before the stop" from "failed after the sessions ended".
func TestStoppedTravelsOnTheWire(t *testing.T) {
	b, err := json.Marshal(compactOutcome{Distro: "Ubuntu", Error: "convert to sparse: x", Stopped: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"stopped":true`) {
		t.Errorf("outcome JSON lacks stopped: %s", b)
	}
	b, _ = json.Marshal(compactOutcome{Distro: "Ubuntu", Refused: "busy"})
	if strings.Contains(string(b), `"stopped"`) {
		t.Errorf("a refusal must not claim a stop: %s", b)
	}
}
