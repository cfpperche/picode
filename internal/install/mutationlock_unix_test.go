//go:build unix

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lockPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "mutate.lock")
}

func TestMutationLockHeldSkipsEntirely(t *testing.T) {
	path := lockPath(t)
	release, err := MutationLock(path, true)
	if err != nil {
		t.Fatalf("held lock must be a no-op: %v", err)
	}
	release()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("held lock must not touch the file, stat says %v", err)
	}
}

func TestMutationLockBlocksWhileHeldAndComesFree(t *testing.T) {
	path := lockPath(t)
	first, err := MutationLock(path, false)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	acquired := make(chan struct{})
	var second func()
	go func() {
		second, err = MutationLock(path, false)
		close(acquired)
	}()
	select {
	case <-acquired:
		t.Fatal("second acquire must block while the first holds the lock")
	case <-time.After(150 * time.Millisecond):
	}
	first()
	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire never completed after release")
	}
	second()
}

func TestMakefileNamesTheSameLock(t *testing.T) {
	makefile, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Skipf("no Makefile to check: %v", err)
	}
	text := string(makefile)
	if !strings.Contains(text, "/tmp/picode-mutate.lock") {
		t.Fatal("Makefile no longer names /tmp/picode-mutate.lock — the CLI and the make targets have drifted apart")
	}
	if !strings.Contains(text, "PICODE_MUTATION_LOCK_HELD") {
		t.Fatal("Makefile no longer hands the lock down via PICODE_MUTATION_LOCK_HELD — a child `picode deploy` would deadlock on its parent's flock")
	}
}
