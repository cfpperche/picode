package tmux

// The drain (ADR-0139), tested against real servers: -S rides every command,
// a session on the legacy server is found and acted on through the primary
// Manager, pane-addressed commands fall back, and the server-wide reads
// merge both sides.
//
// These tests use explicit socket paths, so they never touch the default
// socket: -S beats $TMUX and TMUX_TMPDIR (the trap class of 2026-09-15).

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSocketFlagIsCarried(t *testing.T) {
	sock := socketPath(t, "socket.sock")
	m := NewWithSocket(sock)
	var got [][]string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append(got, append([]string(nil), args...))
		return []byte("x\n"), nil
	}
	if _, err := m.HasSession(context.Background(), "x"); err != nil {
		t.Fatalf("has-session: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no tmux call recorded")
	}
	if got[0][0] != "-S" || got[0][1] != sock {
		t.Fatalf("argv = %v, want it to start with -S <path>", got[0])
	}
	if got[0][2] != "has-session" {
		t.Fatalf("argv = %v, want the subcommand after the socket flag", got[0])
	}
}

func TestDefaultManagerKeepsThePlainShape(t *testing.T) {
	m := New()
	var got [][]string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append(got, append([]string(nil), args...))
		return []byte("x\n"), nil
	}
	if _, err := m.HasSession(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0][0] != "has-session" {
		t.Fatalf("argv = %v, want the plain subcommand (no socket)", got)
	}
}

func requireDrainTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
}

func TestDrainFindsAndActsOnALegacySession(t *testing.T) {
	requireDrainTmux(t)
	ctx := context.Background()
	root := socketDir(t)
	legacyPath := filepath.Join(root, "legacy.sock")
	primaryPath := filepath.Join(root, "primary.sock")
	legacy := NewWithSocket(legacyPath)
	primary := NewWithSocket(primaryPath).WithLegacy(legacy)

	name := SessionName("drain-" + time.Now().Format("150405-000000000"))
	if err := legacy.NewSession(ctx, name, root, "sleep", "60"); err != nil {
		t.Fatalf("legacy session: %v", err)
	}
	t.Cleanup(func() { _ = primary.KillSession(context.Background(), name) })

	// The primary Manager resolves the session on the legacy server.
	has, err := primary.HasSession(ctx, name)
	if err != nil || !has {
		t.Fatalf("HasSession via primary = %v, %v; want true", has, err)
	}
	if got := primary.SocketFor(ctx, name); got != legacyPath {
		t.Fatalf("SocketFor = %q, want the legacy path %q", got, legacyPath)
	}
	if out, err := primary.CaptureTail(ctx, name, 3); err != nil || out == "" && err != nil {
		t.Fatalf("CaptureTail via primary: %v (%q)", err, out)
	}
	if pid, err := primary.PanePID(ctx, name); err != nil || pid <= 0 {
		t.Fatalf("PanePID via primary = %d, %v", pid, err)
	}

	// Reads merge: a primary session joins the legacy one in the listing.
	name2 := SessionName("drain2-" + time.Now().Format("150405-000000000"))
	if err := primary.NewSession(ctx, name2, root, "sleep", "60"); err != nil {
		t.Fatalf("primary session: %v", err)
	}
	t.Cleanup(func() { _ = primary.KillSession(context.Background(), name2) })
	sessions, err := primary.ServerSessions(ctx)
	if err != nil {
		t.Fatalf("ServerSessions: %v", err)
	}
	seen := map[string]bool{}
	for _, s := range sessions {
		seen[s.Name] = true
	}
	if !seen[name] || !seen[name2] {
		t.Fatalf("merged read missed a server: %v", seen)
	}
	if got := primary.SocketFor(ctx, name2); got != primaryPath {
		t.Fatalf("SocketFor(primary session) = %q, want %q", got, primaryPath)
	}

	// A kill through the primary lands on the server that owns the session.
	if err := primary.KillSession(ctx, name); err != nil {
		t.Fatalf("KillSession via primary: %v", err)
	}
	if has, err := legacy.HasSession(ctx, name); err != nil || has {
		t.Fatalf("legacy session survived: %v, %v", has, err)
	}
}

func TestDrainPaneCommandFallsBackToLegacy(t *testing.T) {
	requireDrainTmux(t)
	ctx := context.Background()
	root := socketDir(t)
	legacy := NewWithSocket(filepath.Join(root, "legacy.sock"))
	primary := NewWithSocket(filepath.Join(root, "primary.sock")).WithLegacy(legacy)

	name := SessionName("drainpane-" + time.Now().Format("150405-000000000"))
	if err := legacy.NewSession(ctx, name, root, "sh", "-c", "sleep 60"); err != nil {
		t.Fatalf("legacy session: %v", err)
	}
	t.Cleanup(func() { _ = primary.KillSession(context.Background(), name) })

	sessions, err := primary.ServerSessions(ctx)
	if err != nil {
		t.Fatalf("ServerSessions: %v", err)
	}
	var paneID string
	for _, s := range sessions {
		if s.Name == name {
			paneID = s.PaneID
		}
	}
	if paneID == "" {
		t.Fatalf("legacy pane id not found in %v", sessions)
	}

	// Pane ids are server-scoped; the primary is tried first and the legacy
	// server answers.
	if err := primary.PasteOnly(ctx, paneID, "echo DRAINOK"); err != nil {
		t.Fatalf("PasteOnly via primary: %v", err)
	}
	if err := primary.SubmitPane(ctx, paneID); err != nil {
		t.Fatalf("SubmitPane via primary: %v", err)
	}
	out, err := primary.CaptureTail(ctx, name, 5)
	if err != nil {
		t.Fatalf("CaptureTail: %v", err)
	}
	if !strings.Contains(out, "DRAINOK") {
		t.Fatalf("pane fallback did not reach the legacy pane: %q", out)
	}
}

func TestDrainWithoutLegacyBehavesLikeBefore(t *testing.T) {
	requireDrainTmux(t)
	ctx := context.Background()
	root := socketDir(t)
	primary := NewWithSocket(filepath.Join(root, "only.sock"))
	if _, err := os.Stat(filepath.Join(root, "only.sock")); !os.IsNotExist(err) {
		t.Fatal("socket exists before any call")
	}
	has, err := primary.HasSession(ctx, SessionName("absent-"+time.Now().Format("150405")))
	if err != nil || has {
		t.Fatalf("absent session on a lone Manager = %v, %v", has, err)
	}
	if got := primary.SocketFor(ctx, "anything"); got == "" {
		t.Fatal("SocketFor must name this Manager's socket")
	}
}
