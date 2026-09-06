package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// fakeSource is a scripted tmux for the forensics decision table.
type fakeSource struct {
	avail bool
	list  []tmux.OwnedSession
	err   error
}

func (f *fakeSource) Available() bool { return f.avail }
func (f *fakeSource) ListOwned(context.Context) ([]tmux.OwnedSession, error) {
	return f.list, f.err
}

func TestShutdownSnapshotAndBootDiffDecisionTable(t *testing.T) {
	ctx := context.Background()
	owned := []tmux.OwnedSession{
		{Name: "picode-sh-survivor-1", PanePID: 101, RootCmd: "bash"},
		{Name: "picode-sh-victim-2", PanePID: 102, RootCmd: "/bin/sh launch.sh"},
	}

	t.Run("graceful shutdown then boot with losses", func(t *testing.T) {
		dir := t.TempDir()
		WriteShutdownSnapshotWith(ctx, dir, &fakeSource{avail: true, list: owned})

		// The snapshot file exists and is readable JSON.
		raw, err := os.ReadFile(filepath.Join(dir, snapshotName))
		if err != nil {
			t.Fatal(err)
		}
		var snap ShutdownSnapshot
		if err := json.Unmarshal(raw, &snap); err != nil {
			t.Fatal(err)
		}
		if len(snap.Sessions) != 2 || !snap.Graceful {
			t.Fatalf("snapshot = %+v", snap)
		}

		// Boot: only the survivor is alive — the victim is reported lost.
		lost := BootDiffWith(ctx, dir, &fakeSource{avail: true, list: owned[:1]})
		if !lost["picode-sh-victim-2"] || lost["picode-sh-survivor-1"] {
			t.Fatalf("lost = %v", lost)
		}
		// The report is durable and says so.
		ents, _ := os.ReadDir(dir)
		found := false
		for _, e := range ents {
			if len(e.Name()) > len(reportPrefix) && e.Name()[:len(reportPrefix)] == reportPrefix {
				found = true
				raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
				if err != nil {
					t.Fatal(err)
				}
				var rep RestartReport
				if err := json.Unmarshal(raw, &rep); err != nil {
					t.Fatal(err)
				}
				if rep.Alive != 1 || len(rep.Lost) != 1 || rep.Lost[0] != "picode-sh-victim-2" {
					t.Fatalf("report = %+v", rep)
				}
			}
		}
		if !found {
			t.Fatal("no restart report written")
		}
	})

	t.Run("everything survived", func(t *testing.T) {
		dir := t.TempDir()
		WriteShutdownSnapshotWith(ctx, dir, &fakeSource{avail: true, list: owned})
		lost := BootDiffWith(ctx, dir, &fakeSource{avail: true, list: owned})
		if len(lost) != 0 {
			t.Fatalf("lost = %v, want empty", lost)
		}
	})

	t.Run("unclean exit: no snapshot, sessions alive", func(t *testing.T) {
		dir := t.TempDir()
		lost := BootDiffWith(ctx, dir, &fakeSource{avail: true, list: owned})
		if len(lost) != 0 {
			t.Fatalf("lost = %v, want empty (cannot judge without a snapshot)", lost)
		}
	})

	t.Run("nothing alive anywhere", func(t *testing.T) {
		dir := t.TempDir()
		WriteShutdownSnapshotWith(ctx, dir, &fakeSource{avail: true})
		lost := BootDiffWith(ctx, dir, &fakeSource{avail: true})
		if len(lost) != 0 {
			t.Fatalf("lost = %v, want empty", lost)
		}
	})

	t.Run("nil or unavailable sources are no-ops", func(t *testing.T) {
		dir := t.TempDir()
		WriteShutdownSnapshotWith(ctx, dir, &fakeSource{avail: false})
		if _, err := os.Stat(filepath.Join(dir, snapshotName)); !os.IsNotExist(err) {
			t.Fatal("snapshot written by unavailable source")
		}
		if lost := BootDiffWith(ctx, dir, nil); len(lost) != 0 {
			t.Fatalf("lost = %v", lost)
		}
	})
}

// TestListOwnedReportsPaneFacts is the real-tmux half of the flight
// recorder: owned sessions come back with their pane root and pid.
func TestListOwnedReportsPaneFacts(t *testing.T) {
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	name := "picode-sh-forensics-qa"
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	_ = m.KillSession(ctx, name)
	if err := m.NewSession(ctx, name, "/tmp", "sleep", "30"); err != nil {
		t.Fatal(err)
	}
	owned, err := m.ListOwned(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var found *tmux.OwnedSession
	for i := range owned {
		if owned[i].Name == name {
			found = &owned[i]
		}
	}
	if found == nil {
		t.Fatalf("session %s missing from %+v", name, owned)
	}
	if found.PanePID <= 0 || found.RootCmd == "" {
		t.Fatalf("pane facts incomplete: %+v", found)
	}
}

// TestPaneRootSurvivesSIGHUP is the ADR-0085 experiment in miniature: a
// trapped pane root must ignore SIGHUP exactly like interactive bash does;
// an untrapped one must die, proving the test can tell the difference.
func TestPaneRootSurvivesSIGHUP(t *testing.T) {
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	write := func(name, extra string) string {
		dir := t.TempDir()
		p := filepath.Join(dir, "pane.sh")
		script := "#!/bin/sh\n" + extra + "\nexec sleep 30\n"
		if err := os.WriteFile(p, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := m.NewSession(ctx, name, dir, "/bin/sh", p); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = m.KillSession(ctx, name) })
		return p
	}
	alive := func(name string) bool {
		has, _ := m.HasSession(ctx, name)
		return has
	}

	// Trapped root survives the exact signal that killed the incident panes.
	trapped := "picode-sh-hup-trapped-qa"
	write(trapped, "trap '' HUP")
	pid, err := m.PanePID(ctx, trapped)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscallHUP(pid); err != nil {
		t.Fatalf("HUP: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if !alive(trapped) {
		t.Fatal("trapped pane root died on SIGHUP")
	}

	// Control: an untrapped root dies, so the assertion above means something.
	control := "picode-sh-hup-control-qa"
	write(control, "")
	cpid, err := m.PanePID(ctx, control)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscallHUP(cpid); err != nil {
		t.Fatalf("HUP: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if alive(control) {
		t.Fatal("untrapped pane root survived SIGHUP — control failed")
	}
}

// syscallHUP is SIGHUP without importing syscall in a _test file that also
// serves as documentation: this is the signal the incident panes received.
func syscallHUP(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(hupSignal())
}
