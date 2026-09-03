package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestTuiWorking(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	dir := t.TempDir()
	sfx := time.Now().Format("150405-000000")
	idleID, busyID := "tui-idle-"+sfx, "tui-busy-"+sfx

	idle := tmux.SessionName(idleID)
	if err := m.NewSession(ctx, idle, dir, "sleep", "30"); err != nil {
		t.Fatalf("idle session: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, idle) })

	busy := tmux.SessionName(busyID)
	// pi's spinner frame, not the word: LooksWorking anchors to the frame.
	if err := m.NewSession(ctx, busy, dir, "sh", "-c", "printf '\\342\\240\\213 Working...\\n'; sleep 30"); err != nil {
		t.Fatalf("busy session: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, busy) })

	deadline := time.Now().Add(3 * time.Second)
	var last []string
	for time.Now().Before(deadline) {
		got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/tui-working?ids="+idleID+","+busyID+",tui-gone"))
		if got.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", got.StatusCode)
		}
		var bag struct {
			Working []string `json:"working"`
		}
		if err := json.NewDecoder(got.Body).Decode(&bag); err != nil {
			t.Fatal(err)
		}
		last = bag.Working
		if len(bag.Working) == 1 && bag.Working[0] == busyID {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("working = %v, want only %q", last, busyID)
}

func TestLooksWorkingUnit(t *testing.T) {
	if !tmux.LooksWorking("  ⠋ Thinking… (esc to interrupt)") {
		t.Fatal("spinner line not detected")
	}
	if tmux.LooksWorking("$ echo done\nno code here") {
		t.Fatal("false positive")
	}
	if tmux.LooksWorking("Working... on the word alone") {
		t.Fatal("the word is not the spinner")
	}
}
