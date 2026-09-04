package binwatch

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestChanged(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bin")
	if err := os.WriteFile(p, []byte("a"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	start := Stamp{Path: p, Mtime: st.ModTime(), Size: st.Size()}
	if Changed(start) {
		t.Fatal("fresh file must not look changed")
	}
	if err := os.WriteFile(p, []byte("ab"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !Changed(start) {
		t.Fatal("rewritten file must look changed")
	}
}

func TestSupervised(t *testing.T) {
	t.Setenv("INVOCATION_ID", "")
	if Supervised() {
		t.Fatal("empty INVOCATION_ID must not look supervised")
	}
	t.Setenv("INVOCATION_ID", "aabbcc")
	if !Supervised() {
		t.Fatal("INVOCATION_ID must mean systemd owns the process")
	}
}

func TestWatchDecisionTable(t *testing.T) {
	// Conditions → action. A systemd-owned process never re-execs, even
	// when the binary on disk is newer (deploy copies, then SIGTERM).
	// A cancelled foreground watch must also not re-exec.
	rows := []struct {
		name       string
		supervised bool
		cancel     bool
		change     bool
		wantReload bool
	}{
		{name: "foreground unchanged", change: false, wantReload: false},
		{name: "foreground newer binary", change: true, wantReload: true},
		{name: "foreground cancelled before change", cancel: true, change: true, wantReload: false},
		{name: "systemd newer binary", supervised: true, change: true, wantReload: false},
		{name: "systemd cancelled and newer", supervised: true, cancel: true, change: true, wantReload: false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if row.supervised {
				t.Setenv("INVOCATION_ID", "test-invocation")
			} else {
				t.Setenv("INVOCATION_ID", "")
			}
			dir := t.TempDir()
			p := filepath.Join(dir, "picode")
			if err := os.WriteFile(p, []byte("old"), 0o755); err != nil {
				t.Fatal(err)
			}
			st, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			start := Stamp{Path: p, Mtime: st.ModTime(), Size: st.Size()}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if row.cancel {
				cancel()
			}
			var got atomic.Bool
			if row.supervised {
				Watch(ctx, start, func() { got.Store(true) })
			} else {
				watch(ctx, 15*time.Millisecond, start, func() { got.Store(true) })
			}
			if row.change {
				if err := os.WriteFile(p, []byte("newer"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			deadline := time.Now().Add(120 * time.Millisecond)
			for time.Now().Before(deadline) {
				if got.Load() {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if got.Load() != row.wantReload {
				t.Fatalf("reload=%v, want %v", got.Load(), row.wantReload)
			}
		})
	}
}
