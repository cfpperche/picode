package server

// ADR-0112: the registries remain live projections. A native reporter's local
// observation is only a candidate until the exact process and pane are proved.
import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// Keep legacy live hooks independent of Linux-only recovery, including when
// stale observation files were copied from another host.
var nativeObservationSupported = func() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := os.Stat("/proc/sys/kernel/random/boot_id")
	return err == nil
}

var errNativeObservationBlocked = errors.New("native observation requires a new runtime")
var errNativeObservationUpdating = errors.New("native observation is being written")

type nativeObservation struct {
	Version     int    `json:"version"`
	TermID      string `json:"termId"`
	CLI         string `json:"cli"`
	RunID       string `json:"runId"`
	PID         int    `json:"pid"`
	ProcStart   string `json:"procStart"`
	BootID      string `json:"bootId"`
	State       string `json:"state"`
	SessionID   string `json:"sessionId"`
	SessionPath string `json:"sessionPath"`
	SessionSeq  int64  `json:"sessionSeq"`
	Source      string `json:"source"`
	CodexHooks  bool   `json:"codexHooks"`
	Action      string `json:"action"`
}

func readNativeObservation(dataDir, id string) (nativeObservation, error) {
	var o nativeObservation
	if !nativeObservationSupported() {
		return o, os.ErrNotExist
	}
	if dataDir == "" || id == "" || strings.ContainsAny(id, `/\\.`) {
		return o, errors.New("no native observation")
	}
	base := filepath.Join(dataDir, "native-observations", id)
	lock, err := openNativeObservationFence(base + ".lock")
	if err != nil {
		if os.IsNotExist(err) {
			if _, checkpointErr := os.Lstat(base + ".json"); !os.IsNotExist(checkpointErr) {
				return o, errors.New("native observation fence is missing")
			}
		}
		return o, err
	}
	defer lock.Close()
	raw, err := io.ReadAll(io.LimitReader(lock, 16*1024+1))
	var fence nativeObservation
	if err != nil || len(raw) > 16*1024 || json.Unmarshal(raw, &fence) != nil || fence.Version != 1 || fence.TermID != id || fence.RunID == "" || fence.ProcStart == "" || fence.BootID == "" || fence.PID <= 0 || fence.SessionSeq <= 0 {
		return o, errNativeObservationBlocked
	}
	o, err = readNativeObservationFile(base + ".json")
	if err != nil || o != fence {
		return nativeObservation{}, errors.New("native observation publication is incomplete")
	}
	if o.TermID != id || normalizeTerminalCLI(o.CLI) != o.CLI || o.RunID == "" || len(o.RunID) > runtimeRunIDCap || o.PID <= 0 || o.ProcStart == "" || o.SessionSeq <= 0 || o.SessionSeq > time.Now().Add(5*time.Second).UnixNano() {
		return o, errors.New("invalid native observation")
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil || strings.TrimSpace(string(boot)) != o.BootID || processStartToken(o.PID) != o.ProcStart {
		return o, errors.New("native process changed")
	}
	return o, nil
}

func readNativeObservationFile(path string) (nativeObservation, error) {
	var o nativeObservation
	info, err := os.Lstat(path)
	if err != nil {
		return o, err
	}
	if !info.Mode().IsRegular() || info.Size() > 16*1024 || info.Mode().Perm() != 0o600 {
		return o, errors.New("invalid native observation file")
	}
	// A recorder unable to invalidate this candidate must not leave Idle usable.
	parent, parentErr := os.Stat(filepath.Dir(path))
	if parentErr != nil || parent.Mode().Perm()&0o700 != 0o700 {
		return o, errors.New("native observation directory is unavailable")
	}
	writable, writeErr := os.OpenFile(path, os.O_WRONLY, 0)
	if writeErr != nil {
		return o, writeErr
	}
	_ = writable.Close()
	raw, err := os.ReadFile(path)
	if err != nil {
		return o, err
	}
	if json.Unmarshal(raw, &o) != nil || o.Version != 1 {
		return o, errors.New("invalid native observation")
	}
	return o, nil
}

func observationMatchesRuntime(o nativeObservation, rt TermRuntime) bool {
	return o.Action == "" && o.RunID == rt.RunID && o.PID == rt.PID && o.ProcStart == rt.ProcStart && o.CLI == rt.CLI && o.SessionID == rt.SessionID && o.SessionSeq == rt.SessionSeq && validTermState(o.State)
}

// Called by the existing presence watcher, including while a wrapper is alive:
// native events may have reached disk while the daemon was unavailable.
func reconcileNativeObservation(ctx context.Context, deps Deps, id string) {
	if !nativeObservationSupported() || deps.TermRuntimes == nil || deps.TermStates == nil || deps.Store == nil {
		return
	}
	// Snapshot before disk I/O so an obsolete failed read cannot erase a newer
	// observation installed by another reconciler while these files are read.
	rt, had := deps.TermRuntimes.Get(id)
	o, err := readNativeObservation(deps.DataDir, id)
	if errors.Is(err, errNativeObservationUpdating) {
		return
	}
	if err != nil {
		if had && (rt.Observation || (deps.DataDir != "" && !os.IsNotExist(err))) {
			forgetNativeObservation(deps, id, rt)
		}
		return
	}
	if had && rt.Source == "wrapper" && rt.RunID != o.RunID {
		return
	}
	if o.Action == "end" {
		finishTermRuntime(deps, id, o.RunID)
		return
	}
	launch, err := deps.Store.TerminalLaunch(id)
	if err != nil || launch == nil || launch.CLI != o.CLI {
		return
	}
	if had && rt.RunID == o.RunID && rt.SessionSeq >= o.SessionSeq {
		return
	}
	recoverNativeRuntime(ctx, deps, id, o.CLI, o.RunID, o.PID)
	rt, ok := deps.TermRuntimes.Get(id)
	if !ok || rt.RunID != o.RunID || rt.PID != o.PID || rt.ProcStart != o.ProcStart {
		return
	}
	if o.Action == "start" || o.SessionID == "" || !validTermState(o.State) {
		if rt.Observation {
			forgetNativeObservation(deps, id, rt)
		}
		return // Presence alone is not a conversation or an idle observation.
	}
	source := o.Source
	if o.CodexHooks {
		source = "codex-hook"
	}
	if recordNativeTerminalObservation(deps, id, o.CLI, o.RunID, o.SessionID, o.SessionPath, o.SessionSeq, o.State, source, true) != nil {
		return
	}
	// Preserve the actual event age; restarting must not extend workingTTL.
	deps.TermStates.mu.Lock()
	state := deps.TermStates.m[id]
	if state.RunID == o.RunID && state.SessionSeq == o.SessionSeq {
		state.At = time.Unix(0, o.SessionSeq)
		if state.State == TermWorking && time.Since(state.At) > workingTTL {
			delete(deps.TermStates.m, id)
		} else {
			deps.TermStates.m[id] = state
		}
	}
	deps.TermStates.mu.Unlock()
	if deps.Feed != nil {
		data := map[string]any{"termId": id, "runId": o.RunID, "state": nil, "cli": o.CLI}
		if current, ok := deps.TermStates.Get(id); ok {
			data["state"] = current.State
			data["at"] = current.At
		}
		deps.Feed.Ephemeral("terminal.state", data)
	}
}

func forgetNativeObservation(deps Deps, id string, rt TermRuntime) {
	deps.TermRuntimes.mu.Lock()
	current := deps.TermRuntimes.m[id]
	if current.RunID != rt.RunID || current.SessionSeq != rt.SessionSeq {
		deps.TermRuntimes.mu.Unlock()
		return
	}
	current.SessionID, current.SessionPath = "", ""
	// Keep the sequence fence: only a newer native event can recover trust.
	deps.TermRuntimes.m[id] = current
	deps.TermStates.mu.Lock()
	delete(deps.TermStates.m, id)
	deps.TermStates.mu.Unlock()
	deps.TermRuntimes.mu.Unlock()
	if deps.Feed != nil {
		deps.Feed.Ephemeral("terminal.state", map[string]any{"termId": id, "runId": rt.RunID, "state": nil})
	}
}

var observationFileName = regexp.MustCompile(`^[A-Za-z0-9_-]+\.(json|lock)$`)

// Deleted owners no longer need their bounded, private observation files.
func sweepNativeObservations(deps Deps, owners map[string]bool) {
	if deps.DataDir == "" {
		return
	}
	dir := filepath.Join(deps.DataDir, "native-observations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !observationFileName.MatchString(name) {
			continue
		}
		id := strings.TrimSuffix(strings.TrimSuffix(name, ".json"), ".lock")
		if !owners[id] {
			if _, err := deps.Store.GetTerminal(id); !errors.Is(err, store.ErrNotFound) {
				continue
			}
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
