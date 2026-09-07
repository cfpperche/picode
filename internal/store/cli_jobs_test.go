package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func testCLIJob(cli, action, key string) CLIJob {
	return CLIJob{CLI: cli, Action: action, RequestKey: key}
}

func testStore(t *testing.T) *Store {
	s, err := Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestBeginCLIJobDecisionTable(t *testing.T) {
	s := testStore(t)
	j, created, err := s.BeginCLIJob(testCLIJob("pi", "update", "k1"))
	if err != nil || !created || j.State != "queued" {
		t.Fatalf("first job: created=%v err=%v state=%s", created, err, j.State)
	}
	// Same key, same input → same job, not created.
	same, created, err := s.BeginCLIJob(testCLIJob("pi", "update", "k1"))
	if err != nil || created || same.ID != j.ID {
		t.Fatalf("repeat key: created=%v err=%v id=%s", created, err, same.ID)
	}
	// Same key, different input → conflict.
	if _, _, err := s.BeginCLIJob(testCLIJob("pi", "uninstall", "k1")); !errors.Is(err, ErrCLILifecycleConflict) {
		t.Fatalf("changed input with same key: err=%v", err)
	}
	// Any active job blocks a second one.
	if _, _, err := s.BeginCLIJob(testCLIJob("codex", "update", "k2")); !errors.Is(err, ErrCLILifecycleConflict) {
		t.Fatalf("second active job: err=%v", err)
	}
	// Invalid requests refuse before touching the table.
	if _, _, err := s.BeginCLIJob(testCLIJob("pi", "explode", "k3")); err == nil {
		t.Fatal("invalid action must refuse")
	}
	if _, _, err := s.BeginCLIJob(testCLIJob("nope", "update", "k4")); err == nil {
		t.Fatal("unknown CLI must refuse")
	}
	if _, _, err := s.BeginCLIJob(testCLIJob("pi", "update", "")); err == nil {
		t.Fatal("empty request key must refuse")
	}
	// Terminal states free the lane.
	j.State = "running"
	if j, err = s.UpdateCLIJob(j); err != nil {
		t.Fatalf("update to running: %v", err)
	}
	j.State = "failed"
	j.Message = "vendor said no"
	if j, err = s.UpdateCLIJob(j); err != nil {
		t.Fatalf("update to failed: %v", err)
	}
	k5, _, err := s.BeginCLIJob(testCLIJob("codex", "update", "k5"))
	if err != nil {
		t.Fatalf("job after terminal state: %v", err)
	}
	k5.State = "succeeded"
	if _, err := s.UpdateCLIJob(k5); err != nil {
		t.Fatalf("finish k5: %v", err)
	}
	// Completed jobs are immutable.
	if _, err := s.UpdateCLIJob(j); !errors.Is(err, ErrCLILifecycleConflict) {
		t.Fatalf("update on failed job: err=%v", err)
	}
	// Revision guard.
	fresh, _, _ := s.BeginCLIJob(testCLIJob("grok", "reinstall", "k6"))
	fresh.State = "running"
	if _, err := s.UpdateCLIJob(fresh); err != nil {
		t.Fatalf("fresh to running: %v", err)
	}
	stale := fresh
	stale.Revision--
	stale.State = "failed"
	if _, err := s.UpdateCLIJob(stale); !errors.Is(err, ErrCLILifecycleConflict) {
		t.Fatalf("stale revision: err=%v", err)
	}
}

func TestCLIJobsListKeepsActiveAndRecent(t *testing.T) {
	s := testStore(t)
	first, _, err := s.BeginCLIJob(testCLIJob("pi", "update", "old"))
	if err != nil {
		t.Fatal(err)
	}
	first.State = "succeeded"
	if _, err := s.UpdateCLIJob(first); err != nil {
		t.Fatal(err)
	}
	// The single-lane rule frees after each job reaches a terminal state.
	for i := range 3 {
		j, created, err := s.BeginCLIJob(testCLIJob("codex", "update", "key"+string(rune('a'+i))))
		if err != nil || !created {
			t.Fatalf("job %d: created=%v err=%v", i, created, err)
		}
		j.State = "succeeded"
		if _, err := s.UpdateCLIJob(j); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := s.CLIJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 4 {
		t.Fatalf("want 4 jobs, got %d", len(jobs))
	}
	if _, err := s.CLIJob(first.ID); err != nil {
		t.Fatalf("read one job: %v", err)
	}
}
