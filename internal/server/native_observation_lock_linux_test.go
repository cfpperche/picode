package server

import (
	"errors"
	"github.com/cfpperche/picode/internal/store"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestObservationFenceBusyIsTemporary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "native-observations")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(path, "busy.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	if _, err = readNativeObservation(dir, "busy"); !errors.Is(err, errNativeObservationUpdating) {
		t.Fatalf("in-flight write treated as corruption: %v", err)
	}
	before := TermRuntime{CLI: "codex", RunID: "run", SessionID: "native", SessionSeq: 1, PID: os.Getpid(), ProcStart: processStartToken(os.Getpid()), Observation: true}
	deps := Deps{DataDir: dir, Store: &store.Store{}, TermRuntimes: NewTermRuntimes(), TermStates: NewTermStates()}
	deps.TermRuntimes.m["busy"] = before
	deps.TermStates.m["busy"] = TermState{CLI: "codex", RunID: "run", SessionID: "native", SessionSeq: 1, State: TermIdle}
	reconcileNativeObservation(t.Context(), deps, "busy")
	if after, _ := deps.TermRuntimes.Get("busy"); after != before {
		t.Fatal("busy writer erased runtime")
	}
	if state, ok := deps.TermStates.Get("busy"); !ok || state.State != TermIdle {
		t.Fatal("busy writer erased activity")
	}
	if _, ok := peerLiveTerminal(deps, store.PeerConnection{PeerOwner: store.PeerOwner{OwnerID: "busy", CLI: "codex", SessionKey: "native"}}); ok {
		t.Fatal("delivery allowed while writer is busy")
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if _, err = readNativeObservation(dir, "busy"); !errors.Is(err, errNativeObservationBlocked) {
		t.Fatalf("permanent empty fence not actionable: %v", err)
	}
}

func TestObservationFenceRejectsNonRegularWithoutBlocking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := openNativeObservationFence(path); !errors.Is(err, errNativeObservationBlocked) {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := openNativeObservationFence(link); !errors.Is(err, errNativeObservationBlocked) {
		t.Fatal(err)
	}
}
