package tmux

// ADR-0140: the Manager stamps its own identity into every session it
// creates, and a caller cannot supply one of its own — "whose session is
// this" is not a caller's claim to make. Real tmux, isolated socket.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSessionCarriesTheInstanceStamp(t *testing.T) {
	requireDrainTmux(t)
	ctx := context.Background()
	data := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewWithSocket(filepath.Join(data, "tmux.sock"))
	name := SessionName("stamp-" + time.Now().Format("150405-000000000"))

	if err := m.NewSessionEnv(ctx, name, data, []string{"PICODE_INSTANCE=/somewhere/else"}, "sleep", "30"); err != nil {
		t.Fatalf("NewSessionEnv: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(context.Background(), name) })

	receipt, err := m.SessionReceipt(ctx, name)
	if err != nil {
		t.Fatalf("SessionReceipt: %v", err)
	}
	if receipt.Instance != data {
		t.Fatalf("instance = %q, want the data directory %q (a caller must not be able to claim another identity)", receipt.Instance, data)
	}
	if m.Instance() != data {
		t.Fatalf("Manager.Instance() = %q, want %q", m.Instance(), data)
	}
}

// A Manager on tmux's shared default socket has no data directory of its own,
// so it stamps nothing and claims nothing: the drain's legacy manager only
// reads.
func TestDefaultSocketManagerHasNoInstance(t *testing.T) {
	if got := New().Instance(); got != "" {
		t.Fatalf("Instance() = %q, want empty on the shared default socket", got)
	}
}
