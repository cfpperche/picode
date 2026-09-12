package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func TestUnsupportedObservationPreservesLegacyState(t *testing.T) {
	previous := nativeObservationSupported
	nativeObservationSupported = func() bool { return false }
	t.Cleanup(func() { nativeObservationSupported = previous })
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	deps := Deps{DataDir: data, Store: st, TermRuntimes: NewTermRuntimes(), TermStates: NewTermStates()}
	dir := filepath.Join(data, "native-observations")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "legacy.lock"), []byte("blocked\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, observed := range []bool{false, true} {
		before := TermRuntime{CLI: "codex", RunID: "legacy", SessionID: "native", SessionSeq: 1, Observation: observed}
		deps.TermRuntimes.m["legacy"] = before
		deps.TermStates.m["legacy"] = TermState{CLI: "codex", RunID: "legacy", State: TermIdle}
		reconcileNativeObservation(t.Context(), deps, "legacy")
		after, _ := deps.TermRuntimes.Get("legacy")
		if after != before {
			t.Fatalf("legacy identity changed: %+v", after)
		}
		if state, ok := deps.TermStates.Get("legacy"); !ok || state.State != TermIdle {
			t.Fatal("legacy activity lost")
		}
		if _, err := readNativeObservation(data, "legacy"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unsupported observation: %v", err)
		}
	}
}

func TestRecorderUnsupportedDoesNotCreateFence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix recorder uses fcntl")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	dir := t.TempDir()
	script := `import sys,os
from unittest.mock import patch
module={"__name__":"test_recorder"};exec(sys.argv[1],module)
root=sys.argv[2];report={'action':'start','cli':'codex','pid':os.getpid(),'runId':'test','sessionSeq':1}
with patch.object(sys,'platform','darwin'):
 assert not module['save_observation'](root,'term',report)
assert not os.path.exists(os.path.join(root,'native-observations'))
with patch.object(os.path,'exists',return_value=False):
 assert not module['save_observation'](root,'term',report)
assert not os.path.exists(os.path.join(root,'native-observations'))
`
	if out, err := exec.Command("python3", "-c", script, string(nativeObservationPy), dir).CombinedOutput(); err != nil {
		t.Fatalf("unsupported recorder: %v %s", err, out)
	}
}

func TestBlockedObservationAPIHasRepairReason(t *testing.T) {
	deps, id, emit := observationFixture(t, "codex")
	emit(TermIdle, "native", time.Now().UnixNano())
	reconcileNativeObservation(t.Context(), deps, id)
	fence := filepath.Join(deps.DataDir, "native-observations", id+".lock")
	for _, content := range []string{"blocked\n", "{", `{ "version": 999 }`, ""} {
		if err := os.WriteFile(fence, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		reconcileNativeObservation(t.Context(), deps, id)
		mux := http.NewServeMux()
		registerPeerOnboarding(mux, deps)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/communication/workspaces", nil))
		var got struct{ Recovery, Identity map[string]string }
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 200 || got.Recovery["terminal:"+id] != "restart-required" || got.Identity["terminal:"+id] != "unobserved" {
			t.Fatalf("missing recovery action: %s", rec.Body.String())
		}
	}
	// A fresh invocation can replace the block; a regular event cannot.
	rt, _ := deps.TermRuntimes.Get(id)
	if rt.SessionID != "" {
		t.Fatal("blocked observation still trusted")
	}
}
