package clijob

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// fakeVendor writes a script that fails or succeeds on demand so the state
// machine runs against real exec, never a stub of the store.
func fakeVendor(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "vendor")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func newService(t *testing.T, st *store.Store, exe string, live int) *Service {
	t.Helper()
	s, err := New(Deps{
		Store: st,
		Resolve: func(cli, action, payload string) (Exec, error) {
			if action != "update" && action != "reinstall" && action != "uninstall" {
				return Exec{}, errors.New("bad action")
			}
			return Exec{Exe: exe, Args: []string{}}, nil
		},
		LiveTerminals: func(string) int { return live },
		RunTimeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func waitFor(t *testing.T, s *Service, id, state string) store.CLIJob {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		j, err := s.deps.Store.CLIJob(id)
		if err == nil && j.State == state {
			return j
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job %s never reached %s", id, state)
	return store.CLIJob{}
}

func TestServiceRunsVendorCommandOnce(t *testing.T) {
	st := store2(t)
	exe := fakeVendor(t, "echo installed 1.2.3")
	s := newService(t, st, exe, 0)
	j, err := s.Start("pi", "update", "k1", "", false)
	if err != nil {
		t.Fatal(err)
	}
	j = waitFor(t, s, j.ID, "succeeded")
	if !strings.Contains(j.Output, "installed 1.2.3") {
		t.Fatalf("output tail missing: %q", j.Output)
	}
	// Same request key returns the finished job without re-running.
	again, err := s.Start("pi", "update", "k1", "", false)
	if err != nil || again.ID != j.ID {
		t.Fatalf("idempotent start: %v %s", err, again.ID)
	}
}

func TestServiceFailureKeepsOutputAndState(t *testing.T) {
	st := store2(t)
	exe := fakeVendor(t, "echo boom >&2; exit 1")
	s := newService(t, st, exe, 0)
	j, _ := s.Start("grok", "reinstall", "k1", "", false)
	j = waitFor(t, s, j.ID, "failed")
	if !strings.Contains(j.Output, "boom") || j.Message != "boom" {
		t.Fatalf("failure details: %+v", j)
	}
}

func TestServiceTerminalGuard(t *testing.T) {
	st := store2(t)
	exe := fakeVendor(t, "exit 0")
	s := newService(t, st, exe, 2)
	if _, err := s.Start("pi", "update", "k1", "", false); !errors.Is(err, ErrTerminalsRunning) {
		t.Fatalf("guard: %v", err)
	}
	if _, err := s.Start("pi", "update", "k1", "", true); err != nil {
		t.Fatalf("confirmed start: %v", err)
	}
}

func TestRestartMarksInterruptedNeverReplays(t *testing.T) {
	st := store2(t)
	exe := fakeVendor(t, "sleep 0.4; exit 0")
	s := newService(t, st, exe, 0)
	j, _ := s.Start("hermes", "uninstall", "k1", "", false)
	waitFor(t, s, j.ID, "running")
	s.Close() // graceful shutdown cancels the command
	stopped, err := st.CLIJob(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.State != "interrupted" {
		t.Fatalf("graceful shutdown state = %s (%s)", stopped.State, stopped.Message)
	}
	// A restart with a job still active in the store (crash recovery) marks
	// it interrupted and never replays it.
	seeded, _, err := st.BeginCLIJob(store.CLIJob{CLI: "pi", Action: "update", RequestKey: "k2"})
	if err != nil {
		t.Fatal(err)
	}
	seeded.State = "running"
	if _, err := st.UpdateCLIJob(seeded); err != nil {
		t.Fatal(err)
	}
	newService(t, st, exe, 0)
	recovered, err := st.CLIJob(seeded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.State != "interrupted" {
		t.Fatalf("crash recovery state = %s", recovered.State)
	}
}

func TestAfterSuccessFiresOnce(t *testing.T) {
	st := store2(t)
	exe := fakeVendor(t, "exit 0")
	var mu sync.Mutex
	calls := 0
	s, err := New(Deps{
		Store: st,
		Resolve: func(cli, action, payload string) (Exec, error) {
			return Exec{Exe: exe}, nil
		},
		AfterSuccess: func(cli, action string) { mu.Lock(); calls++; mu.Unlock() },
		RunTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	j, _ := s.Start("pi", "update", "k1", "", false)
	waitFor(t, s, j.ID, "succeeded")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := calls
		mu.Unlock()
		if n == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("AfterSuccess calls = %d, want 1", calls)
}

func store2(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
