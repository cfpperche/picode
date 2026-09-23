package desktop

import (
	"os"
	"testing"
	"time"
)

func TestHoldDistroNestsAndReleases(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	if HoldActive() {
		t.Fatal("held before any hold")
	}
	r1 := HoldDistro("move")
	r2 := HoldDistro("update")
	if !HoldActive() {
		t.Fatal("not held after HoldDistro")
	}
	r1()
	if !HoldActive() {
		t.Error("the second hold was released by the first")
	}
	r2()
	r2() // idempotent
	if HoldActive() {
		t.Error("still held after every release")
	}
}

func TestStaleHoldDoesNotHold(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	release := HoldDistro("move")
	defer release()
	old := time.Now().Add(-HoldFresh - time.Second)
	if err := os.Chtimes(HoldPath(), old, old); err != nil {
		t.Fatal(err)
	}
	if HoldActive() {
		t.Error("a hold nobody refreshed must expire")
	}
}
