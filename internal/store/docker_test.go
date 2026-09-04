package store

import (
	"path/filepath"
	"testing"
)

func TestDockerHistoryKeepsActiveOperationsOutsideWindow(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	active, _, err := s.BeginDockerOperation(DockerOperation{RequestKey: "older-active", Endpoint: "unix:///tmp/a", ContainerID: "a", Action: "start"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`UPDATE docker_operations SET created_at = '2000-01-01T00:00:00Z' WHERE id = ?`, active.ID); err != nil {
		t.Fatal(err)
	}
	recent, _, err := s.BeginDockerOperation(DockerOperation{RequestKey: "newer-completed", Endpoint: "unix:///tmp/a", ContainerID: "b", Action: "start"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishDockerOperation(recent.ID, "succeeded", "verified"); err != nil {
		t.Fatal(err)
	}
	ops, err := s.DockerOperations(1)
	if err != nil || len(ops) != 2 || ops[0].ID != recent.ID || ops[1].ID != active.ID {
		t.Fatalf("active job disappeared: %+v %v", ops, err)
	}
	if err = s.FinishDockerOperation(active.ID, "unknown", "interrupted"); err != nil {
		t.Fatal(err)
	}
	ops, err = s.DockerOperations(1)
	if err != nil || len(ops) != 1 || ops[0].ID != recent.ID {
		t.Fatalf("completed history exceeded window: %+v %v", ops, err)
	}
}

func TestDockerRecoveryAndEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	events := []Event{}
	s.OnEvent = func(ev Event) { events = append(events, ev) }
	op, created, err := s.BeginDockerOperation(DockerOperation{RequestKey: "request-123", Endpoint: "unix:///tmp/a", ContainerID: "a", ContainerName: "qa", Action: "start", Actor: "Local user"})
	if err != nil || !created {
		t.Fatalf("begin %v %v", created, err)
	}
	if len(events) != 1 || events[0].Type != "docker.operation" {
		t.Fatalf("events %+v", events)
	}
	_ = s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.RecoverDockerOperations(); err != nil {
		t.Fatal(err)
	}
	got, err := s.DockerOperation(op.ID)
	if err != nil || got.State != "unknown" || got.FinishedAt == "" {
		t.Fatalf("recovery %+v %v", got, err)
	}
	if err = s.FinishDockerOperation(op.ID, "succeeded", "late response"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.DockerOperation(op.ID)
	if got.State != "unknown" {
		t.Fatal("late completion overwrote outcome")
	}
}
