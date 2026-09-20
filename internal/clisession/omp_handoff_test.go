package clisession

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// seedOmpTranscript writes one omp session file in the measured real shape:
// header, title record (no id), model_change, an id/parentId-chained
// conversation with a tool call/result, a runtime custom record and an
// abandoned branch that the live path must drop.
func seedOmpTranscript(t *testing.T, home, bucket, name string) string {
	t.Helper()
	dir := filepath.Join(home, ".omp", "agent", "sessions", bucket)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"title","v":1,"title":"","updatedAt":"2026-09-17T15:00:00.000Z"}` + "\n" +
		`{"type":"session","version":3,"id":"01a0b00a-read-aaaa-bbbb-cccccccccccc","timestamp":"2026-09-17T15:00:00.000Z","cwd":"/home/goat/proj"}` + "\n" +
		`{"type":"model_change","id":"aaaa0001","parentId":null,"timestamp":"2026-09-17T15:00:00.010Z","model":"google/gemini-3.6-flash","resolvedModelIsFallback":false}` + "\n" +
		`{"type":"message","id":"aaaa0002","parentId":"aaaa0001","timestamp":"2026-09-17T15:00:01.000Z","message":{"role":"user","content":[{"type":"text","text":"fix the race"}],"attribution":"user","timestamp":1758123601000}}` + "\n" +
		// An abandoned branch: its descendant exists but is never the last entry.
		`{"type":"message","id":"aaaa0003","parentId":"aaaa0002","timestamp":"2026-09-17T15:00:02.000Z","message":{"role":"assistant","content":[{"type":"text","text":"wrong turn"}],"stopReason":"stop","timestamp":1758123602000}}` + "\n" +
		`{"type":"message","id":"aaaa0004","parentId":"aaaa0002","timestamp":"2026-09-17T15:00:03.000Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hmm"},{"type":"toolCall","id":"call_1","name":"bash","arguments":{"command":"go test ./..."}}],"stopReason":"toolUse","timestamp":1758123603000}}` + "\n" +
		`{"type":"custom","customType":"tool_execution_start","data":{"toolCallId":"call_1"},"id":"aaaa0005","parentId":"aaaa0004","timestamp":"2026-09-17T15:00:03.100Z"}` + "\n" +
		`{"type":"message","id":"aaaa0006","parentId":"aaaa0005","timestamp":"2026-09-17T15:00:04.000Z","message":{"role":"toolResult","toolCallId":"call_1","toolName":"bash","content":[{"type":"text","text":"ok"}],"isError":false,"timestamp":1758123604000}}` + "\n" +
		`{"type":"message","id":"aaaa0007","parentId":"aaaa0006","timestamp":"2026-09-17T15:00:05.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Done."}],"stopReason":"stop","timestamp":1758123605000}}` + "\n" +
		`{"type":"custom","customType":"session_exit","data":{"reason":"dispose"},"id":"aaaa0008","parentId":"aaaa0007","timestamp":"2026-09-17T15:00:06.000Z"}` + "\n"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOmpReadTimeline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := seedOmpTranscript(t, home, "-home-goat-proj", "2026-09-17T15-00-00-000Z_01a0b00a-read-aaaa-bbbb-cccccccccccc.jsonl")
	tl, err := (OmpSource{}).Read(context.Background(), Ref{ID: "01a0b00a-read-aaaa-bbbb-cccccccccccc", Path: path, Cwd: "/home/goat/proj"})
	if err != nil {
		t.Fatal(err)
	}
	if tl.Header.SourceID != "01a0b00a-read-aaaa-bbbb-cccccccccccc" || tl.Header.Cwd != "/home/goat/proj" {
		t.Errorf("header identity = %+v", tl.Header)
	}
	if tl.Header.Provider != "google" || tl.Header.Model != "gemini-3.6-flash" {
		t.Errorf("provider/model = %s/%s", tl.Header.Provider, tl.Header.Model)
	}
	// The wrong turn is a dead branch; the live path carries the rest.
	var kinds []string
	for _, e := range tl.Events {
		switch e.Kind {
		case transcript.KindMessage:
			kinds = append(kinds, e.Role)
		case transcript.KindToolCall:
			kinds = append(kinds, "tool_call")
		case transcript.KindToolResult:
			kinds = append(kinds, "tool_result")
		case transcript.KindThinking:
			kinds = append(kinds, "thinking")
		default:
			kinds = append(kinds, string(e.Kind))
		}
	}
	want := "user thinking tool_call tool_result assistant"
	if got := strings.Join(kinds, " "); got != want {
		t.Fatalf("events = %q, want %q", got, want)
	}
	last := tl.Events[len(tl.Events)-1]
	if last.Text != "Done." {
		t.Errorf("last event = %+v", last)
	}
	if n := tl.Manifest.Dropped["omp.branch"]; n != 1 {
		t.Errorf("dropped omp.branch = %d, want 1", n)
	}
	if n := tl.Manifest.Dropped["omp.custom.tool_execution_start"]; n != 1 {
		t.Errorf("dropped custom record = %d", n)
	}
	if n := tl.Manifest.Dropped["omp.custom.session_exit"]; n != 1 {
		t.Errorf("dropped session_exit = %d", n)
	}
	// A file outside the omp sessions root is refused.
	if _, err := (OmpSource{}).Read(context.Background(), Ref{ID: "x", Path: filepath.Join(home, "elsewhere.jsonl")}); err != ErrNotUnderRoot {
		t.Errorf("outside root err = %v", err)
	}
}

func TestOmpWriteRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 9, 17, 18, 0, 0, 0, time.UTC)
	got, err := (OmpSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(home, ".omp", "agent", "sessions", "-home-goat-proj")
	if filepath.Dir(got.Path) != wantDir {
		t.Fatalf("path = %s, want under %s", got.Path, wantDir)
	}
	if !reflect.DeepEqual(got.ResumeArgs, []string{"--resume", got.ID}) {
		t.Errorf("resumeArgs = %v", got.ResumeArgs)
	}
	if got.Name != "Race fix" || got.Messages != 4 || got.Model != "claude-sonnet-5" {
		t.Fatalf("summary = %+v", got)
	}
	// Thinking never travels; the source's provider does not leak in.
	raw, _ := os.ReadFile(got.Path)
	if strings.Contains(string(raw), "hmm") || strings.Contains(string(raw), `"provider":"anthropic"`) {
		t.Fatal("thinking or foreign provider written")
	}
	// The written header is omp's own schema (key order is Go's map
	// marshal, irrelevant to a JSON parser) and the title record carries
	// the handoff name without joining the entry chain.
	if !strings.Contains(string(raw), `"type":"session","version":3`) {
		t.Fatal("missing omp session header")
	}
	if !strings.Contains(string(raw), `"type":"title"`) || !strings.Contains(string(raw), `"title":"Race fix"`) {
		t.Fatal("missing omp-shaped title record")
	}
	// Listing sees the new session (the picker is where it lands).
	list, err := OmpSource{}.List("/home/goat/proj")
	if err != nil || len(list) != 1 || list[0].ID != got.ID {
		t.Fatalf("list after write = %v, %v", list, err)
	}
}

func TestOmpWriteRequiresFolderAndTurns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := (OmpSource{}).Write(context.Background(), sample(), WriteRequest{}); err == nil {
		t.Fatal("write without a folder succeeded")
	}
	empty := transcript.Timeline{Header: transcript.Header{Cwd: "/home/goat/proj"}}
	if _, err := (OmpSource{}).Write(context.Background(), empty, WriteRequest{Cwd: "/home/goat/proj"}); err == nil || !strings.Contains(err.Error(), "no turns") {
		t.Fatalf("empty timeline err = %v", err)
	}
}
