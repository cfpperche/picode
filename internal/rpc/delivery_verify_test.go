package rpc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeliveryBaselineRejectsOldRepeatedText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	payload := `Human reply to your question "[Teste burst]": reply resolvido por mim :D`
	old := userLine(payload)
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := CaptureDeliveryBaseline(path)

	if AwaitUserMessageAfter(baseline, payload, 20*time.Millisecond) {
		t.Fatal("pre-existing identical reply proved a new delivery")
	}
	appendLine(t, path, messageLine("assistant", payload))
	if AwaitUserMessageAfter(baseline, payload, 20*time.Millisecond) {
		t.Fatal("new assistant text proved a user delivery")
	}
	appendLine(t, path, userLine(payload))
	if !AwaitUserMessageAfter(baseline, payload, 20*time.Millisecond) {
		t.Fatal("newly appended user reply was not found")
	}
}

func TestDeliveryBaselineMatchesFullNormalizedPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := CaptureDeliveryBaseline(path)
	payload := "unique prefix that must not be dropped   \n  repeated reply tail"
	appendLine(t, path, userLine("some other prefix repeated reply tail"))
	if AwaitUserMessageAfter(baseline, payload, 20*time.Millisecond) {
		t.Fatal("matching only the payload tail produced a false positive")
	}
	appendLine(t, path, userLine("unique prefix that must not be dropped repeated reply tail plus unrelated text"))
	if AwaitUserMessageAfter(baseline, payload, 20*time.Millisecond) {
		t.Fatal("a longer user message was mistaken for the exact payload")
	}
	appendLine(t, path, userLine("unique prefix that must not be dropped repeated reply tail"))
	if !AwaitUserMessageAfter(baseline, payload, 20*time.Millisecond) {
		t.Fatal("whitespace-normalized full payload was not found")
	}
}

func TestDeliveryBaselineVerifiesEscapedPayloadBeyondFixedReadFloor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := CaptureDeliveryBaseline(path)
	// encoding/json expands '<' to six bytes, so this row is larger than the
	// old fixed 512 KiB verification window even though the reply is smaller.
	payload := strings.Repeat("<", 90*1024)
	appendLine(t, path, userLine(payload))
	if !UserMessageAfter(baseline, payload) {
		t.Fatal("a valid escaped reply beyond the fixed read floor was not verified")
	}
}

func TestDeliveryBaselineReadsReplacedOrTruncatedFileFromZero(t *testing.T) {
	for _, replace := range []bool{false, true} {
		name := "truncated"
		if replace {
			name = "replaced"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.jsonl")
			if err := os.WriteFile(path, []byte(userLine("old reply")+userLine("padding padding padding")), 0o644); err != nil {
				t.Fatal(err)
			}
			baseline := CaptureDeliveryBaseline(path)
			if replace {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(path, []byte(userLine("fresh reply")), 0o644); err != nil {
				t.Fatal(err)
			}
			if !AwaitUserMessageAfter(baseline, "fresh reply", 20*time.Millisecond) {
				t.Fatal("fresh file content was not scanned from byte zero")
			}
		})
	}
}

func TestDeliveryBaselineFindsFileCreatedAfterDirectorySnapshot(t *testing.T) {
	dir := t.TempDir()
	baseline := CaptureDeliveryBaseline(dir)
	path := filepath.Join(dir, "new.jsonl")
	if err := os.WriteFile(path, []byte(userLine("new reply")), 0o644); err != nil {
		t.Fatal(err)
	}
	if !AwaitUserMessageAfter(baseline, "new reply", 20*time.Millisecond) {
		t.Fatal("session created after baseline was not scanned from byte zero")
	}
}

func TestCancelledDeliveryWaitStillHonoursMaterializedRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := CaptureDeliveryBaseline(path)
	appendLine(t, path, userLine("reply won the race"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !AwaitUserMessageAfterContext(ctx, baseline, "reply won the race", time.Second) {
		t.Fatal("cancellation hid a durably materialized reply")
	}
	if !UserMessageAfter(baseline, "reply won the race") {
		t.Fatal("non-blocking delivery probe missed the row")
	}
}

func userLine(text string) string { return messageLine("user", text) }

func messageLine(role, text string) string {
	row := map[string]any{
		"type": "message",
		"message": map[string]any{
			"role":    role,
			"content": []map[string]string{{"type": "text", "text": text}},
		},
	}
	b, _ := json.Marshal(row)
	return string(b) + "\n"
}

func appendLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		t.Fatal(err)
	}
}
