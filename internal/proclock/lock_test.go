//go:build unix

package proclock

import (
	"path/filepath"
	"testing"
)

func TestAcquireExclusive(t *testing.T) {
	p := filepath.Join(t.TempDir(), "picode.lock")
	unlock, err := Acquire(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(p); err == nil {
		t.Fatal("second lock succeeded")
	}
	unlock()
	unlock2, err := Acquire(p)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	unlock2()
}
