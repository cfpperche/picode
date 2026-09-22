package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func observationFixture(t *testing.T, cli string) (Deps, string, func(string, string, int64)) {
	// Native observation is Linux-only by construction: recovery compares
	// the host's boot id from /proc/sys/kernel/random/boot_id and the
	// process start token beside it, and nativeObservationSupported already
	// says so. Without the skip these tests ran on macOS, found every
	// recovery refused with "native process changed", and one of them then
	// wrote into a map the recovery would have built — a nil-map panic that
	// reads like a product fault and is a fixture running where the feature
	// does not exist.
	if !nativeObservationSupported() {
		t.Skip("native observation needs Linux (/proc/sys/kernel/random/boot_id)")
	}
	t.Helper()
	if processStartToken(os.Getpid()) == "" || !tmux.New().Available() {
		t.Skip("requires Linux process metadata and tmux")
	}
	data := t.TempDir()
	s, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	w, _ := s.AddWorkspace("observation", data)
	term, _ := s.CreateTerminalIn(w.ID, cli, data)
	if err := s.SetTerminalLaunch(term.ID, cli, clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	m := tmux.New()
	name := tmux.ShellSessionName(term.ID)
	if err := m.NewSession(t.Context(), name, data, "sleep", "180"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.KillSession(context.Background(), name) })
	pid, err := m.PanePID(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	hook, err := ensureHookScript(data)
	if err != nil {
		t.Fatal(err)
	}
	// Network is deliberately unavailable. The real hook must save first.
	emit := func(state, id string, seq int64) {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"state": state, "session_id": id, "transcript_path": filepath.Join(data, "session.jsonl")})
		cmd := exec.Command(hook, "auto", cli, string(payload))
		cmd.Env = append(os.Environ(), "PICODE_TERM_ID="+term.ID, "PICODE_TERM_URL=http://127.0.0.1:1", "PICODE_TUI_RUN_ID=observation-run", fmt.Sprintf("PICODE_TUI_PID=%d", pid), fmt.Sprintf("PICODE_NATIVE_SESSION_SEQ=%d", seq))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hook: %v %s", err, out)
		}
	}
	return Deps{Store: s, DataDir: data, Tmux: m, TermRuntimes: NewTermRuntimes(), TermStates: NewTermStates()}, term.ID, emit
}

func TestNativeObservationRestartRecovery(t *testing.T) {
	for _, cli := range []string{"pi", "grok", "opencode", "hermes", "codex", "claude-code"} {
		t.Run(cli, func(t *testing.T) {
			deps, id, emit := observationFixture(t, cli)
			seq := time.Now().UnixNano()
			emit(TermIdle, "native-A", seq)
			reconcileNativeObservation(t.Context(), deps, id)
			first, _ := deps.TermRuntimes.Get(id)
			if first.SessionID != "native-A" || !first.Observation {
				t.Fatalf("not recovered: %+v", first)
			}
			connection, token, err := deps.Store.EnablePeer("terminal", id, "native-A")
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := peerLiveTerminal(deps, connection); !ok {
				t.Fatal("idle not available")
			}
			// A state change saved before the HTTP handler runs blocks automatic input.
			emit(TermNeedsYou, "native-A", seq+1)
			if _, ok := peerLiveTerminal(deps, connection); ok {
				t.Fatal("old idle trusted over pending approval")
			}
			deps.TermRuntimes = NewTermRuntimes()
			deps.TermStates = NewTermStates()
			reconcileNativeObservation(t.Context(), deps, id)
			state, _ := deps.TermStates.Get(id)
			if state.State != TermNeedsYou {
				t.Fatalf("approval lost: %+v", state)
			}
			emit(TermWorking, "native-B", seq+2)
			emit(TermIdle, "native-A", seq) // delayed completion cannot roll back identity
			deps.TermRuntimes = NewTermRuntimes()
			deps.TermStates = NewTermStates()
			reconcileNativeObservation(t.Context(), deps, id)
			rt, _ := deps.TermRuntimes.Get(id)
			if rt.SessionID != "native-B" {
				t.Fatalf("wrong session: %+v", rt)
			}
			if _, err := deps.Store.AuthorizePeer(token); err == nil {
				t.Fatal("previous conversation remained authorized")
			}
		})
	}
}

func TestNativeObservationRejectsInvalidRecovery(t *testing.T) {
	for _, scenario := range []string{"missing", "corrupt", "public", "missing-fence", "corrupt-fence", "blocked-fence", "read-only-file", "read-only-directory", "pid-reuse", "reboot", "different-pane", "ended", "working-expired"} {
		t.Run(scenario, func(t *testing.T) {
			deps, id, emit := observationFixture(t, "grok")
			emit(TermIdle, "native", time.Now().UnixNano())
			path := filepath.Join(deps.DataDir, "native-observations", id+".json")
			raw, _ := os.ReadFile(path)
			var record map[string]any
			json.Unmarshal(raw, &record)
			switch scenario {
			case "missing":
				os.Remove(path)
			case "corrupt":
				os.WriteFile(path, []byte("{"), 0o600)
			case "public":
				os.Chmod(path, 0o644)
			case "missing-fence":
				os.Remove(filepath.Join(filepath.Dir(path), id+".lock"))
			case "corrupt-fence":
				os.WriteFile(filepath.Join(filepath.Dir(path), id+".lock"), []byte("blocked"), 0o600)
			case "blocked-fence":
				os.Chmod(filepath.Join(filepath.Dir(path), id+".lock"), 0o000)
			case "read-only-file":
				os.Chmod(path, 0o400)
			case "read-only-directory":
				os.Chmod(filepath.Dir(path), 0o500)
				t.Cleanup(func() { os.Chmod(filepath.Dir(path), 0o700) })
			default:
				switch scenario {
				case "pid-reuse":
					record["procStart"] = "0"
				case "reboot":
					record["bootId"] = "previous-boot"
				case "different-pane":
					record["pid"] = os.Getpid()
					record["procStart"] = processStartToken(os.Getpid())
				case "ended":
					record["action"] = "end"
				case "working-expired":
					record["state"] = TermWorking
					record["sessionSeq"] = time.Now().Add(-workingTTL - time.Minute).UnixNano()
				}
				raw, _ = json.Marshal(record)
				os.WriteFile(path, raw, 0o600)
				os.WriteFile(filepath.Join(filepath.Dir(path), id+".lock"), raw, 0o600)
			}
			reconcileNativeObservation(t.Context(), deps, id)
			if _, ok := deps.TermStates.Get(id); ok {
				t.Fatal("invalid observation became live")
			}
			if rt, _ := deps.TermRuntimes.Get(id); scenario != "working-expired" && rt.SessionID != "" {
				t.Fatal("invalid identity restored")
			}
		})
	}
}

func TestNativeObservationFirstCheckpointBlocksOldIdle(t *testing.T) {
	deps, id, emit := observationFixture(t, "grok")
	pid, _ := deps.Tmux.PanePID(t.Context(), tmux.ShellSessionName(id))
	deps.TermRuntimes.Start(id, TermRuntime{CLI: "grok", Source: "wrapper", RunID: "observation-run", PID: pid, ProcStart: processStartToken(pid)})
	seq := time.Now().UnixNano()
	if err := recordNativeTerminalSession(deps, id, "grok", "observation-run", "native", "", seq, TermIdle); err != nil {
		t.Fatal(err)
	}
	connection, _, err := deps.Store.EnablePeer("terminal", id, "native")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := peerLiveTerminal(deps, connection); !ok {
		t.Fatal("legacy fixture was not idle")
	}
	emit(TermWorking, "native", seq+1)
	if _, ok := peerLiveTerminal(deps, connection); ok {
		t.Fatal("first checkpoint was ignored before HTTP")
	}
	os.Remove(filepath.Join(deps.DataDir, "native-observations", id+".json"))
	if _, ok := peerLiveTerminal(deps, connection); ok {
		t.Fatal("first failed publication restored legacy Idle")
	}
	reconcileNativeObservation(t.Context(), deps, id)
	if rt, _ := deps.TermRuntimes.Get(id); rt.SessionID != "" {
		t.Fatal("failed first publication retained identity")
	}
}

func TestNativeObservationObsoleteInvalidationRetainsNewerState(t *testing.T) {
	deps, id, emit := observationFixture(t, "grok")
	seq := time.Now().UnixNano()
	emit(TermIdle, "first", seq)
	reconcileNativeObservation(t.Context(), deps, id)
	old, _ := deps.TermRuntimes.Get(id)
	emit(TermWorking, "second", seq+1)
	reconcileNativeObservation(t.Context(), deps, id)
	forgetNativeObservation(deps, id, old)
	state, ok := deps.TermStates.Get(id)
	if !ok || state.SessionID != "second" || state.State != TermWorking {
		t.Fatal("old invalidation cleared newer state")
	}
}

func TestRecoveredPiWaitsForReceiverWithoutRestart(t *testing.T) {
	deps, id, emit := observationFixture(t, "pi")
	deps.Replies = NewTuiReplies()
	emit(TermIdle, "pi-native", time.Now().UnixNano())
	reconcileNativeObservation(t.Context(), deps, id)
	term, _ := deps.Store.GetTerminal(id)
	if err := deps.Store.SetPeerParticipants(term.WorkspaceID, []store.PeerSelection{{Kind: "terminal", OwnerID: id, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	pref, _ := deps.Store.PeerParticipant("terminal", id)
	connection, _, err := deps.Store.EnsureParticipantPeer(pref, "pi-native")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := deps.TermRuntimes.Get(id)
	phase, _ := applyPeerParticipant(t.Context(), deps, pref, connection)
	after, _ := deps.TermRuntimes.Get(id)
	if phase != "waiting-receiver" || after.PID != before.PID || after.RunID != before.RunID || !processAlive(before) {
		t.Fatalf("Pi was restarted: phase=%s before=%+v after=%+v", phase, before, after)
	}
	deps.Replies.helloConnectionProcess(termReplyKey(id), before.SessionPath, connection.ID, before.RunID)
	if phase, _ := applyPeerParticipant(t.Context(), deps, pref, connection); phase != "connected" {
		t.Fatalf("hello did not restore connection: %s", phase)
	}
}

func TestNativeObservationRecorderFailureAndCodexPrecedence(t *testing.T) {
	// The recorder refuses on any host without /proc/sys/kernel/random/boot_id
	// — its first two lines say so, because an unsupported host must not write
	// a fence that poisons its otherwise valid HTTP reports. This test asserts
	// the writes succeed, so it belongs where they can.
	if !nativeObservationSupported() {
		t.Skip("the observation recorder is Linux-only by construction")
	}
	if processStartToken(os.Getpid()) == "" {
		t.Skip("Linux process metadata required")
	}
	dir := t.TempDir()
	ensureHookScript(dir)
	script := `import importlib.util,os,json,sys,time
from unittest.mock import patch
spec=importlib.util.spec_from_file_location("recorder",sys.argv[1]);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
root=os.path.dirname(sys.argv[1]);path=os.path.join(root,"native-observations","fixture.json")
r={"cli":"codex","runId":"run","pid":os.getpid(),"state":"idle","sessionId":"native","sessionSeq":time.time_ns(),"source":"codex-hook"}
for n,target in enumerate(["fcntl.flock","os.fsync","os.replace"]):
 r.update(runId="run"+str(n),sessionSeq=r["sessionSeq"]+10)
 assert m.save_observation(root,"fixture",dict(r,action="start"))
 r["sessionSeq"]+=1
 assert m.save_observation(root,"fixture",r)
 with patch(target,side_effect=OSError("fixture failure")):
  assert not m.save_observation(root,"fixture",dict(r,state="working",sessionSeq=r["sessionSeq"]+2))
 assert not os.path.exists(path),target
 assert not m.save_observation(root,"fixture",dict(r,sessionSeq=r["sessionSeq"]+1)),target
 assert not m.save_observation(root,"fixture",dict(r,source="codex-notify",sessionSeq=r["sessionSeq"]+3)),target
 if target=="os.replace":
  assert m.save_observation(root,"fixture",dict(r,sessionSeq=r["sessionSeq"]+3))
 else:
  assert not m.save_observation(root,"fixture",dict(r,sessionSeq=r["sessionSeq"]+3)),target
  assert not m.save_observation(root,"fixture",dict(r,action="start",sessionSeq=r["sessionSeq"]+4)),target
r.update(runId="fence-fsync-success",sessionSeq=r["sessionSeq"]+10)
assert m.save_observation(root,"fixture",dict(r,action="start"))
r["sessionSeq"]+=1
assert m.save_observation(root,"fixture",r)
with patch("os.fsync",side_effect=[None,OSError("checkpoint fsync failure")]):
 assert not m.save_observation(root,"fixture",dict(r,state="working",sessionSeq=r["sessionSeq"]+2))
assert not m.save_observation(root,"fixture",dict(r,sessionSeq=r["sessionSeq"]+1))
r["sessionSeq"]+=3
assert m.save_observation(root,"fixture",dict(r,state="working"))
assert not m.save_observation(root,"fixture",dict(r,source="codex-notify",sessionId="auxiliary",sessionSeq=r["sessionSeq"]+1))
with open(path) as f:record=json.load(f)
assert record["sessionId"]=="native" and record["state"]=="working" and record["codexHooks"]
new=dict(r,runId="replacement",action="start",sessionSeq=r["sessionSeq"]+2)
assert m.save_observation(root,"fixture",new)
assert not m.save_observation(root,"fixture",dict(r,sessionSeq=r["sessionSeq"]+3))
assert m.save_observation(root,"fixture",dict(new,action="end",sessionSeq=r["sessionSeq"]+4))
assert not m.save_observation(root,"fixture",dict(new,action="",state="idle",sessionSeq=r["sessionSeq"]+5))
with patch("fcntl.flock",side_effect=OSError("lock failure")),patch("os.chmod",side_effect=OSError("chmod failure")):
 assert not m.save_observation(root,"fixture",dict(new,sessionSeq=r["sessionSeq"]+6))
assert not m.save_observation(root,"fixture",dict(new,action="",sessionSeq=r["sessionSeq"]+7))
`
	cmd := exec.Command("python3", "-c", script, filepath.Join(dir, "picode-hook-observation.py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("recorder regression: %v %s", err, out)
	}
}

func TestPiLegacyHeartbeatRecoversOnlyItsWrapper(t *testing.T) {
	deps, id, _ := observationFixture(t, "pi")
	name := tmux.ShellSessionName(id)
	deps.Tmux.KillSession(t.Context(), name)
	script := fmt.Sprintf("export PICODE_TERM_ID=%s PICODE_TUI_RUN_ID=legacy-run PICODE_TUI_PID=$$; sleep 120 & wait", id)
	if err := deps.Tmux.NewSession(t.Context(), name, deps.DataDir, "sh", "-c", script); err != nil {
		t.Fatal(err)
	}
	pid, _ := deps.Tmux.PanePID(t.Context(), name)
	child := 0
	for n := 0; n < 50 && child == 0; n++ {
		for candidate, parent := range readProcSnapshot().ppid {
			if parent == pid {
				child = candidate
				break
			}
		}
		if child == 0 {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if child == 0 {
		t.Fatal("fixture child missing")
	}
	deps.TermRuntimes.Start(id, TermRuntime{CLI: "pi", Source: "tmux-fallback", RunID: "fallback", PID: pid, ProcStart: processStartToken(pid)})
	recoverPiReceiverRuntime(t.Context(), deps, id, "wrong-run", child, 0)
	if rt, _ := deps.TermRuntimes.Get(id); rt.RunID != "fallback" {
		t.Fatal("wrong heartbeat replaced wrapper")
	}
	recoverPiReceiverRuntime(t.Context(), deps, id, "legacy-run", child, 0)
	rt, _ := deps.TermRuntimes.Get(id)
	if rt.RunID != "legacy-run" || rt.PID != pid || rt.SessionID != "" {
		t.Fatalf("invalid heartbeat recovery: %+v", rt)
	}
	if _, ok := deps.TermStates.Get(id); ok {
		t.Fatal("heartbeat invented activity")
	}
}

func TestNativeObservationCleanupPreservesLiveAndNewOwners(t *testing.T) {
	deps, id, emit := observationFixture(t, "grok")
	emit(TermIdle, "native", time.Now().UnixNano())
	dir := filepath.Join(deps.DataDir, "native-observations")
	os.WriteFile(filepath.Join(dir, "deleted.json"), []byte("{}"), 0o600)
	os.WriteFile(filepath.Join(dir, "deleted.lock"), nil, 0o600)
	// Empty list simulates an owner created after the watcher took its snapshot.
	sweepNativeObservations(deps, map[string]bool{})
	if _, err := os.Stat(filepath.Join(dir, id+".json")); err != nil {
		t.Fatal("new live owner lost record")
	}
	for _, name := range []string{"deleted.json", "deleted.lock"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatal("deleted owner record retained")
		}
	}
}
