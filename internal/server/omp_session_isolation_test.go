package server

import (
	"path/filepath"
	"testing"
)

func TestOmpAgentSessionDirIsDurableAndScoped(t *testing.T) {
	dataDir := t.TempDir()
	got := ompAgentSessionDir(dataDir, "agent-123")
	want := filepath.Join(dataDir, "omp-sessions", "agent-123")
	if got != want {
		t.Fatalf("ompAgentSessionDir = %q, want %q", got, want)
	}
	if other := ompAgentSessionDir(dataDir, "agent-456"); other == got {
		t.Fatal("different agents must not share the Omp session directory")
	}
}
