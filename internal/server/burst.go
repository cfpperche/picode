package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

const (
	burstControlConflict = "This agent's terminal is changing; try the reply again when it is ready"
	burstControlBusy     = "This agent's terminal is already changing. Wait for it to finish and try again."

	burstReceiving  = "receiving"
	burstProcessing = "processing"
	burstRestoring  = "restoring"
	burstDone       = "done"
	burstFailed     = "failed"
	burstIdle       = "idle"
)

const burstDeadline = 15 * time.Minute

var burstSeq atomic.Uint64

// BurstState is the compact projection rendered over an agent's parked TUI.
// It is process-local orchestration state, never a persisted run-mode change.
type BurstState struct {
	AgentID    string `json:"agentId"`
	Generation string `json:"generation"`
	TaskID     string `json:"taskId,omitempty"`
	Phase      string `json:"phase"`
	Activity   string `json:"activity,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	// TerminalUnavailable distinguishes a failed reply from a failed pane
	// restoration so Return can offer a real TUI restart instead of a dead tab.
	TerminalUnavailable bool   `json:"terminalUnavailable,omitempty"`
	StartedAt           string `json:"startedAt"`
	UpdatedAt           string `json:"updatedAt"`
}

func burstInFlight(phase string) bool {
	return phase == burstReceiving || phase == burstProcessing || phase == burstRestoring
}

type burstEntry struct {
	state  BurstState
	cancel context.CancelFunc
}

// BurstCoordinator owns one generation per agent. All late updates are
// generation-checked so an older goroutine cannot overwrite a newer burst.
type BurstCoordinator struct {
	mu       sync.Mutex
	entries  map[string]*burstEntry
	controls map[string]int
}

func NewBurstCoordinator() *BurstCoordinator {
	return &BurstCoordinator{entries: map[string]*burstEntry{}, controls: map[string]int{}}
}

func (b *BurstCoordinator) Snapshot(agentID string) *BurstState {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entries[agentID]
	if e == nil {
		return nil
	}
	copy := e.state
	return &copy
}

func (b *BurstCoordinator) Reserve(agentID string) (BurstState, context.Context, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.controls[agentID] > 0 {
		return BurstState{}, nil, errors.New(burstControlConflict)
	}
	if e := b.entries[agentID]; e != nil {
		if burstInFlight(e.state.Phase) {
			return BurstState{}, nil, fmt.Errorf("this agent is already processing an inbox reply")
		}
		if e.cancel != nil {
			e.cancel()
		}
	}
	gen := strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatUint(burstSeq.Add(1), 36)
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().UTC().Format(time.RFC3339Nano)
	st := BurstState{AgentID: agentID, Generation: gen, Phase: burstReceiving, Activity: "Receiving your reply", StartedAt: now, UpdatedAt: now}
	b.entries[agentID] = &burstEntry{state: st, cancel: cancel}
	return st, ctx, nil
}

func (b *BurstCoordinator) CheckControl(agentID string) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.controls[agentID] > 0 {
		return errors.New(burstControlConflict)
	}
	return nil
}

// BeginControl blocks a new burst while one pane/session mutation performs
// its cancel → wait → mutate sequence. The returned release is idempotent.
func (b *BurstCoordinator) BeginControl(agentID string) func() {
	if b == nil {
		return func() {}
	}
	b.mu.Lock()
	b.controls[agentID]++
	b.mu.Unlock()
	return b.controlRelease(agentID)
}

// TryBeginControl is the exclusive form used by forced TUI recovery: a
// duplicate click must not kill the fresh pane started by the first request.
func (b *BurstCoordinator) TryBeginControl(agentID string) (func(), error) {
	if b == nil {
		return func() {}, nil
	}
	b.mu.Lock()
	if b.controls[agentID] > 0 {
		b.mu.Unlock()
		return nil, errors.New(burstControlBusy)
	}
	b.controls[agentID] = 1
	b.mu.Unlock()
	return b.controlRelease(agentID), nil
}

func (b *BurstCoordinator) controlRelease(agentID string) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			if b.controls[agentID] <= 1 {
				delete(b.controls, agentID)
			} else {
				b.controls[agentID]--
			}
			b.mu.Unlock()
		})
	}
}

func (b *BurstCoordinator) RestoreIfAbsent(state BurstState) (BurstState, bool) {
	if b == nil || strings.TrimSpace(state.AgentID) == "" || strings.TrimSpace(state.Generation) == "" {
		return BurstState{}, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries[state.AgentID] != nil {
		return BurstState{}, false
	}
	_, cancel := context.WithCancel(context.Background())
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	b.entries[state.AgentID] = &burstEntry{state: state, cancel: cancel}
	return state, true
}

func (b *BurstCoordinator) Update(agentID, generation string, mutate func(*BurstState)) (BurstState, bool) {
	if b == nil {
		return BurstState{}, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entries[agentID]
	if e == nil || e.state.Generation != generation {
		return BurstState{}, false
	}
	mutate(&e.state)
	e.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return e.state, true
}

func (b *BurstCoordinator) Clear(agentID, generation string) bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	e := b.entries[agentID]
	if e == nil || generation == "" || e.state.Generation != generation {
		b.mu.Unlock()
		return false
	}
	delete(b.entries, agentID)
	cancel := e.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return true
}

// Cancel requests an in-flight generation to unwind. A completed/failed card
// is only dismissed; there is no process left to stop.
func (b *BurstCoordinator) Cancel(agentID, generation string) (requested, cleared bool) {
	if b == nil {
		return false, false
	}
	b.mu.Lock()
	e := b.entries[agentID]
	if e == nil || generation == "" || e.state.Generation != generation {
		b.mu.Unlock()
		return false, false
	}
	if !burstInFlight(e.state.Phase) {
		delete(b.entries, agentID)
		cancel := e.cancel
		b.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return false, true
	}
	cancel := e.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return true, false
}

// CancelAllAndWait lets an orderly daemon shutdown restore every borrowed
// pane before the store closes. A hard re-exec still relies on the holder and
// parent-death signal because process defers cannot run in that path.
func (b *BurstCoordinator) CancelAllAndWait(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(b.entries))
	for _, e := range b.entries {
		if burstInFlight(e.state.Phase) && e.cancel != nil {
			cancels = append(cancels, e.cancel)
		}
	}
	b.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		b.mu.Lock()
		active := false
		for _, e := range b.entries {
			if burstInFlight(e.state.Phase) {
				active = true
				break
			}
		}
		b.mu.Unlock()
		if !active {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func (deps Deps) publishBurst(st BurstState) {
	if deps.Feed != nil {
		deps.Feed.Ephemeral("agent.burst", st)
	}
}

func (deps Deps) updateBurst(agentID, generation string, mutate func(*BurstState)) bool {
	st, ok := deps.Bursts.Update(agentID, generation, mutate)
	if ok {
		deps.publishBurst(st)
	}
	return ok
}

func (deps Deps) clearBurst(agentID, generation string) {
	previous := deps.Bursts.Snapshot(agentID)
	if !deps.Bursts.Clear(agentID, generation) {
		return
	}
	idle := BurstState{AgentID: agentID, Generation: generation, Phase: burstIdle}
	if previous != nil && previous.Generation == generation {
		idle.TerminalUnavailable = previous.TerminalUnavailable
	}
	deps.publishBurst(idle)
}

// burstPreflight is deliberately pure: the decision table in ADR-0059 is
// table-tested without spawning tmux or pi.
type burstPreflight struct {
	Active        bool
	TmuxAvailable bool
	SessionExists bool
	TUIWorking    bool
	PiAvailable   bool
	SessionSafe   bool
	LeaseClear    bool
	Managed       bool
}

func burstRefusal(in burstPreflight) string {
	switch {
	case in.Active:
		return "This agent is already processing an inbox reply."
	case in.Managed:
		return "This agent is no longer in its terminal."
	case !in.LeaseClear:
		return "The terminal is still recovering an earlier reply. Wait for it to return, then try again."
	case !in.TmuxAvailable:
		return "Terminal integration is unavailable on this machine."
	case !in.SessionExists:
		return "The agent terminal is no longer running."
	case in.TUIWorking:
		return "The terminal agent is still working. Wait for it to stop, then try again."
	case !in.PiAvailable:
		return errAgentCmdMissing.Error()
	case !in.SessionSafe:
		return "The terminal session could not be identified safely. Open the TUI and try again."
	default:
		return ""
	}
}

// startReplyBurst validates before mutating the Inbox. After RespondAndPark,
// every failure is represented by a burst card and a durable task state.
func (deps Deps) startReplyBurst(ctx context.Context, itemID, verb, text string) (agentID, generation string, err error) {
	it, err := deps.Store.GetInboxItem(itemID)
	if err != nil {
		return "", "", err
	}
	if it.SourceKind != store.InboxFromAgent || strings.TrimSpace(it.SourceID) == "" {
		return "", "", fmt.Errorf("this item has no agent terminal")
	}
	agentID = it.SourceID
	agent, err := deps.Store.GetAgent(agentID)
	if err != nil {
		return "", "", err
	}
	// Give a reply racing an explicit pane/session mutation a deterministic,
	// retryable refusal before it inspects transitional process state. Reserve
	// repeats this check under lock to close the remainder of the race.
	if err := deps.Bursts.CheckControl(agentID); err != nil {
		return "", "", err
	}
	_, cwd, err := deps.agentHome(agent)
	if err != nil {
		return "", "", err
	}

	active := deps.Bursts.Snapshot(agentID)
	available := deps.Tmux != nil && deps.Tmux.Available()
	hasSession := false
	working := false
	if available {
		hasSession, _ = deps.Tmux.HasSession(ctx, tmux.SessionName(agentID))
		if hasSession {
			if tail, e := deps.Tmux.CaptureTail(ctx, tmux.SessionName(agentID), 8); e == nil {
				working = tmux.LooksWorking(tail)
			}
		}
	}
	_, piErr := exec.LookPath(deps.AgentCmd)
	sessionPath, sessionOK := deps.resolveBurstSession(agent, it, cwd)
	if reason := burstRefusal(burstPreflight{
		Active: active != nil && burstInFlight(active.Phase), TmuxAvailable: available,
		SessionExists: hasSession, TUIWorking: working, PiAvailable: piErr == nil,
		SessionSafe: sessionOK, LeaseClear: !hasBurstMarker(deps.DataDir, agentID),
		Managed: deps.Runtime.Active(agentID),
	}); reason != "" {
		return "", "", errors.New(reason)
	}

	// Reserve before touching the selected-session pointer. Two Inbox requests
	// can pass the observational preflight together; only the winner may choose
	// the session used by the temporary writer and restored TUI.
	st, burstCtx, err := deps.Bursts.Reserve(agentID)
	if err != nil {
		return "", "", err
	}
	generation = st.Generation
	deps.publishBurst(st)

	marker, err := createBurstMarker(deps.DataDir, agentID, generation)
	if err != nil {
		deps.clearBurst(agentID, generation)
		return "", "", err
	}
	_, task, err := deps.Store.RespondAndPark(itemID, verb, text)
	if err != nil {
		_ = os.Remove(marker)
		deps.clearBurst(agentID, generation)
		return "", "", err
	}
	deps.updateBurst(agentID, generation, func(s *BurstState) { s.TaskID = task.ID })

	// Build both process command lines from the item-owned path. Runtime also
	// receives it explicitly, so a stale store pointer can never redirect the
	// burst during its startup window. Persist before the holder starts: if the
	// daemon crashes afterward, the holder and the store still agree on what Pi
	// must resume. A failed pane install rolls the previous pointer back.
	var restoreEnv, restoreArgs []string
	burstAgent, err := deps.installBurstSession(agent, sessionPath, func(exact store.Agent) error {
		if st, statErr := os.Stat(sessionPath); statErr != nil || st.IsDir() || st.Size() == 0 {
			if statErr != nil {
				return fmt.Errorf("%w: exact session disappeared: %v", errSelectBurstSession, statErr)
			}
			return fmt.Errorf("%w: exact session is no longer a non-empty file", errSelectBurstSession)
		}
		restoreArgs = exact.CLIFlagsForSpawn("")
		restoreEnv = append([]string(nil), exact.SpawnEnv()...)
		if deps.DataDir != "" {
			restoreEnv = append(restoreEnv, "PICODE_DATA="+deps.DataDir)
		}
		holderArgs := burstHolderArgs(marker, deps.AgentCmd, restoreArgs)
		return deps.Tmux.RespawnPaneEnv(ctx, tmux.SessionName(agentID), cwd, restoreEnv, "/bin/sh", holderArgs...)
	})
	if err != nil {
		deps.Bursts.Cancel(agentID, generation)
		_ = os.Remove(marker)
		note := "The terminal could not be paused. Send the reply again from this item."
		card := "The terminal could not be paused. Return to the terminal and try again."
		if errors.Is(err, errSelectBurstSession) {
			note = "The terminal session could not be selected. Send the reply again from this item."
			card = "The terminal session could not be selected. Return to the terminal and try again."
		}
		_ = deps.Store.EndReplyBurst(task.ID, store.TaskFailed, err.Error(), note)
		deps.updateBurst(agentID, generation, func(s *BurstState) {
			s.Phase = burstFailed
			s.Activity = "Reply could not start"
			s.Error = card
		})
		return agentID, generation, nil
	}

	go deps.runReplyBurst(burstCtx, burstAgent, cwd, sessionPath, marker, restoreEnv, restoreArgs, task, generation)
	return agentID, generation, nil
}

var errSelectBurstSession = errors.New("select burst session")

func (deps Deps) installBurstSession(agent store.Agent, sessionPath string, install func(store.Agent) error) (store.Agent, error) {
	exact := agent
	exact.SessionPath = &sessionPath
	changed := agent.SessionPath == nil || filepath.Clean(*agent.SessionPath) != filepath.Clean(sessionPath)
	if changed {
		if _, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath}); err != nil {
			return store.Agent{}, fmt.Errorf("%w: %v", errSelectBurstSession, err)
		}
	}
	if err := install(exact); err != nil {
		if changed {
			prior := ""
			if agent.SessionPath != nil {
				prior = *agent.SessionPath
			}
			if _, rollbackErr := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &prior}); rollbackErr != nil {
				return store.Agent{}, fmt.Errorf("%w; restore selected session: %v", err, rollbackErr)
			}
		}
		return store.Agent{}, err
	}
	return exact, nil
}

func (deps Deps) resolveBurstSession(agent store.Agent, it store.InboxItem, cwd string) (string, bool) {
	// A burst must target the session that filed this exact item. Falling back
	// to the agent's current or latest session can deliver an old answer into a
	// different conversation.
	path := strings.TrimSpace(it.SessionPath)
	if path == "" || !safeSessionPath(path, session.AgentDir(agent.ID), session.Dir(cwd)) {
		return "", false
	}
	st, err := os.Stat(path)
	return path, err == nil && !st.IsDir() && st.Size() > 0
}

func burstMarkerDir(dataDir string) string {
	root := dataDir
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(os.TempDir(), "picode")
	}
	return filepath.Join(root, "bursts")
}

func burstMarkerPrefix(agentID string) string {
	return strings.TrimPrefix(tmux.SessionName(agentID), tmux.Prefix) + "-"
}

func hasBurstMarker(dataDir, agentID string) bool {
	matches, _ := filepath.Glob(filepath.Join(burstMarkerDir(dataDir), burstMarkerPrefix(agentID)+"*.hold"))
	return len(matches) > 0
}

func removeBurstMarkers(dataDir, agentID string) {
	matches, _ := filepath.Glob(filepath.Join(burstMarkerDir(dataDir), burstMarkerPrefix(agentID)+"*.hold*"))
	for _, path := range matches {
		_ = os.Remove(path)
	}
}

// BurstRecoveryPending reports a lease startup could not safely clear. The
// daemon must not open its store/start accepting writers while this is true.
func BurstRecoveryPending(dataDir string) bool {
	matches, _ := filepath.Glob(filepath.Join(burstMarkerDir(dataDir), "*.hold"))
	return len(matches) > 0
}

func createBurstMarker(dataDir, agentID, generation string) (string, error) {
	dir := burstMarkerDir(dataDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := burstMarkerPrefix(agentID) + generation + ".hold"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func recordBurstRPCPID(marker string, pid int) error {
	if pid <= 0 {
		return errors.New("invalid RPC process id")
	}
	tmp := marker + ".tmp"
	if err := os.WriteFile(tmp, []byte(fmt.Sprintf("%d\n%d\n", os.Getpid(), pid)), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, marker); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func releaseBurstMarker(path string) error {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	fields := strings.Fields(string(body))
	content := "released\n"
	if len(fields) > 1 {
		if _, err := strconv.Atoi(fields[1]); err == nil {
			content += fields[1] + "\n"
		}
	}
	tmp := path + ".release"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ReleaseInterruptedBurstMarkers hands stale pane leases back to their
// holders once per daemon process start. It must not run on HTTP rebinds.
func ReleaseInterruptedBurstMarkers(dataDir string) error {
	return releaseInterruptedBurstMarkers(dataDir, 6*time.Second)
}

func releaseInterruptedBurstMarkers(dataDir string, wait time.Duration) error {
	if strings.TrimSpace(dataDir) == "" {
		return nil
	}
	matches, _ := filepath.Glob(filepath.Join(dataDir, "bursts", "*.hold"))
	var releaseErrs []error
	released := make([]string, 0, len(matches))
	for _, path := range matches {
		// Preserve the leased RPC pid while signalling release. A Unix re-exec
		// keeps the daemon pid alive, so deleting the marker could otherwise
		// make a holder miss the worker it must kill before restoring Pi. One
		// damaged marker must not prevent every other pane from recovering.
		if err := releaseBurstMarker(path); err != nil {
			releaseErrs = append(releaseErrs, fmt.Errorf("release %s: %w", filepath.Base(path), err))
			// An unreadable/non-file lease cannot be observed by a holder.
			// Quarantine it out of the *.hold namespace so one damaged entry
			// cannot permanently refuse every later reply for that agent.
			damaged := path + ".damaged"
			_ = os.Remove(damaged)
			if renameErr := os.Rename(path, damaged); renameErr != nil {
				_ = os.Remove(path)
			}
			continue
		}
		released = append(released, path)
	}
	temps, _ := filepath.Glob(filepath.Join(dataDir, "bursts", "*.hold.*"))
	for _, path := range temps {
		_ = os.Remove(path)
	}
	if len(released) == 0 {
		return errors.Join(releaseErrs...)
	}

	// Do not accept a new Inbox reply while an old holder may still be
	// terminating its leased writer. Holders remove their marker immediately
	// before exec-restoring Pi; a dead holder times out honestly.
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		remaining := 0
		for _, path := range released {
			if _, err := os.Stat(path); err == nil {
				remaining++
			}
		}
		if remaining == 0 {
			return errors.Join(releaseErrs...)
		}
		time.Sleep(50 * time.Millisecond)
	}
	remaining := 0
	for _, path := range released {
		if _, err := os.Stat(path); err == nil {
			remaining++
			// A daemon can die after creating the marker but before installing
			// a holder or RPC lease. That marker has no child PID and is safe to
			// remove. If a leased process may still exist, retain the marker and
			// refuse new replies rather than risk overlapping writers.
			if !burstMarkerHasLiveRPC(path) {
				_ = os.Remove(path)
			}
		}
	}
	if remaining > 0 {
		releaseErrs = append(releaseErrs, fmt.Errorf("timed out waiting for %d interrupted terminal reply holder(s)", remaining))
	}
	return errors.Join(releaseErrs...)
}

func burstMarkerHasLiveRPC(path string) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	fields := strings.Fields(string(body))
	if len(fields) < 2 {
		return false
	}
	pid, err := strconv.Atoi(fields[1])
	if err != nil || pid <= 0 {
		return false
	}
	if runtime.GOOS != "linux" {
		// Without a portable PID identity check, preserve exclusivity.
		return true
	}
	_, err = os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return err == nil
}

func burstHolderArgs(marker, agentCmd string, restoreArgs []string) []string {
	// If the coordinator releases the marker, the daemon dies, or the safety
	// deadline expires, the holder execs Pi in the same pane. This keeps tmux
	// and its browser attach alive even across a hard deploy.
	script := `pid=$1; marker=$2; deadline=$3; shift 3; rpc_pid=""; while kill -0 "$pid" 2>/dev/null && [ "$(sed -n '1p' "$marker" 2>/dev/null)" = "$pid" ] && [ "$(date +%s)" -lt "$deadline" ]; do candidate=$(sed -n '2p' "$marker" 2>/dev/null); case "$candidate" in ''|*[!0-9]*) ;; *) rpc_pid=$candidate ;; esac; sleep 1; done; if [ -r "$marker" ]; then candidate=$(sed -n '2p' "$marker" 2>/dev/null); case "$candidate" in ''|*[!0-9]*) ;; *) rpc_pid=$candidate ;; esac; fi; if [ -n "$rpc_pid" ]; then kill -TERM "$rpc_pid" 2>/dev/null || true; n=0; while kill -0 "$rpc_pid" 2>/dev/null && [ "$n" -lt 20 ]; do n=$((n+1)); sleep .1; done; kill -KILL "$rpc_pid" 2>/dev/null || true; fi; rm -f "$marker"; exec "$@"`
	args := []string{"-c", script, "picode-burst", strconv.Itoa(os.Getpid()), marker, strconv.FormatInt(time.Now().Add(burstDeadline).Unix(), 10), agentCmd}
	return append(args, restoreArgs...)
}

type burstAttemptSignals struct {
	started chan struct{}
	settled chan string
	exited  chan bool
	blocked chan struct{}
}

func newBurstAttemptSignals() (*burstAttemptSignals, *rpc.RunObserver) {
	s := &burstAttemptSignals{started: make(chan struct{}, 1), settled: make(chan string, 1), exited: make(chan bool, 1), blocked: make(chan struct{}, 1)}
	o := &rpc.RunObserver{
		OnStarted: func() {
			select {
			case s.started <- struct{}{}:
			default:
			}
		},
		OnSettled: func(final string) {
			select {
			case s.settled <- final:
			default:
			}
		},
		OnExit: func(expected bool) {
			select {
			case s.exited <- expected:
			default:
			}
		},
	}
	return s, o
}

func (deps Deps) runReplyBurst(parent context.Context, agent store.Agent, cwd, sessionPath, marker string, restoreEnv, restoreArgs []string, original store.Task, generation string) {
	ctx, cancel := context.WithTimeout(parent, burstDeadline)
	defer cancel()
	var final string
	var runErr error
	cancelled := false
	delivered := false
	markDelivered := func(task store.Task) {
		delivered = true
		if err := deps.Store.FinishTask(task.ID, store.TaskDelivered, ""); err != nil {
			runErr = fmt.Errorf("record delivered reply: %w", err)
			return
		}
		_ = deps.Store.AppendEvent("task.delivered", &agent.ID, nil, map[string]string{"taskId": task.ID, "kind": task.Kind})
	}

	for attempt := 1; attempt <= 3 && !delivered; attempt++ {
		if attempt > 1 {
			deps.updateBurst(agent.ID, generation, func(s *BurstState) {
				s.Phase = burstReceiving
				s.Activity = fmt.Sprintf("Retrying delivery (%d of 3)", attempt)
			})
			select {
			case <-ctx.Done():
				cancelled = errors.Is(ctx.Err(), context.Canceled)
				break
			case <-time.After(time.Second):
			}
		}
		if ctx.Err() != nil {
			break
		}

		sig, observer := newBurstAttemptSignals()
		observer.OnSpawn = func(pid int) error { return recordBurstRPCPID(marker, pid) }
		ma, err := deps.Runtime.StartBurst(agent.ID, cwd, sessionPath, observer)
		if err != nil {
			runErr = err
			continue
		}
		unsub := ma.WatchEvents(func(ev rpc.Event) { deps.projectBurstEvent(agent.ID, generation, ev, sig.blocked) })

		readyCtx, readyCancel := context.WithTimeout(ctx, 30*time.Second)
		state, err := ma.GetState(readyCtx)
		readyCancel()
		if err == nil {
			err = requireBurstSession(state, sessionPath)
		}
		if err != nil {
			runErr = fmt.Errorf("start control channel: %w", err)
			unsub()
			deps.Runtime.Stop(agent.ID)
			continue
		}

		task, err := deps.Store.ClaimTask(agent.ID, original.ID)
		if err != nil {
			runErr = fmt.Errorf("claim reply: %w", err)
			unsub()
			deps.Runtime.Stop(agent.ID)
			break
		}
		baseline := rpc.CaptureDeliveryBaseline(sessionPath)
		deps.updateBurst(agent.ID, generation, func(s *BurstState) {
			s.Phase = burstProcessing
			s.Activity = "Processing your reply"
		})
		sendCtx, sendCancel := context.WithTimeout(ctx, 60*time.Second)
		err = ma.SendPromptCtx(sendCtx, task.Payload)
		sendCancel()
		if err == nil {
			select {
			case <-sig.started:
			case <-sig.exited:
				err = errors.New("the control process exited before the reply started")
			case <-sig.blocked:
				err = errors.New("the reply needs an interactive answer")
			case <-ctx.Done():
				err = ctx.Err()
			case <-time.After(30 * time.Second):
				err = errors.New("the reply did not start")
			}
		}
		if err == nil && !rpc.AwaitUserMessageAfterContext(ctx, baseline, task.Payload, 30*time.Second) {
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = errors.New("the reply did not reach newly appended session bytes")
			}
		}
		if err != nil {
			runErr = err
			unsub()
			deps.Runtime.Stop(agent.ID)
			// Stop first, then probe once more. If cancellation raced the
			// append, the durable row wins: never reopen an already delivered
			// answer and invite a duplicate.
			if rpc.UserMessageAfter(baseline, task.Payload) {
				markDelivered(task)
				break
			}
			if attempt < 3 {
				_ = deps.Store.FinishTask(task.ID, store.TaskQueued, err.Error())
				continue
			}
			break
		}

		runErr = nil
		markDelivered(task)
		select {
		case final = <-sig.settled:
		case <-sig.blocked:
			runErr = errors.New("the reply needs another interactive answer")
		case expected := <-sig.exited:
			if !expected {
				runErr = errors.New("the control process exited before the reply finished")
			}
		case <-ctx.Done():
			runErr = ctx.Err()
		}
		unsub()
		deps.Runtime.Stop(agent.ID)
	}

	if ctx.Err() != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		cancelled = true
	}
	if !delivered {
		status := store.TaskFailed
		message := "Reply could not be processed."
		if cancelled {
			status = store.TaskCancelled
			message = "Reply cancelled by the user."
		}
		detail := message
		if runErr != nil {
			detail = runErr.Error()
		}
		_ = deps.Store.EndReplyBurst(original.ID, status, detail, message+" Send it again from the Inbox.")
	}
	deps.restoreReplyBurst(agent, marker, restoreEnv, restoreArgs, generation, final, runErr, cancelled)
}

func requireBurstSession(res rpc.Response, want string) error {
	var data struct {
		SessionFile string `json:"sessionFile"`
	}
	if err := json.Unmarshal(res.Data, &data); err != nil {
		return fmt.Errorf("read session state: %w", err)
	}
	got, _ := filepath.Abs(data.SessionFile)
	expected, _ := filepath.Abs(want)
	if got == "" || filepath.Clean(got) != filepath.Clean(expected) {
		return fmt.Errorf("control channel resumed %q, want %q", data.SessionFile, want)
	}
	return nil
}

func (deps Deps) projectBurstEvent(agentID, generation string, ev rpc.Event, blocked chan<- struct{}) {
	var raw struct {
		Type                  string `json:"type"`
		ToolName              string `json:"toolName"`
		Method                string `json:"method"`
		AssistantMessageEvent struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		} `json:"assistantMessageEvent"`
		Message struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(ev, &raw) != nil {
		return
	}
	switch raw.Type {
	case "message_update":
		switch raw.AssistantMessageEvent.Type {
		case "text_delta":
			deps.updateBurst(agentID, generation, func(s *BurstState) {
				s.Output = appendBurstOutput(s.Output, raw.AssistantMessageEvent.Delta)
				s.Activity = "Writing a response"
			})
		case "thinking_start", "thinking_delta":
			deps.updateBurst(agentID, generation, func(s *BurstState) { s.Activity = "Thinking" })
		}
	case "message_end", "turn_end":
		if raw.Message.Role == "assistant" {
			text := ""
			for _, c := range raw.Message.Content {
				if c.Type == "text" {
					text += c.Text
				}
			}
			if text != "" {
				deps.updateBurst(agentID, generation, func(s *BurstState) { s.Output = appendBurstOutput("", text) })
			}
		}
	case "tool_execution_start":
		deps.updateBurst(agentID, generation, func(s *BurstState) { s.Activity = burstToolActivity(raw.ToolName) })
	case "tool_execution_end":
		deps.updateBurst(agentID, generation, func(s *BurstState) { s.Activity = "Processing your reply" })
	case "auto_retry_start":
		deps.updateBurst(agentID, generation, func(s *BurstState) { s.Activity = "Retrying the response" })
	case "extension_ui_request":
		// Only dialogs that wait on a human answer can block a transient
		// reply. Passive decoration (status lines, widgets, titles, notify
		// toasts, editor text) is fire-and-forget: sessions whose extensions
		// render those at startup must still receive replies.
		if rpc.IsDialogMethod(raw.Method) {
			select {
			case blocked <- struct{}{}:
			default:
			}
		}
	}
}

func appendBurstOutput(current, delta string) string {
	const maxBytes = 100_000
	out := current + delta
	if len(out) <= maxBytes {
		return out
	}
	start := len(out) - maxBytes
	for start < len(out) && !utf8.RuneStart(out[start]) {
		start++
	}
	return out[start:]
}

func burstToolActivity(name string) string {
	switch name {
	case "read":
		return "Reading files"
	case "write", "edit":
		return "Updating files"
	case "bash":
		return "Running a command"
	case "web_search":
		return "Checking sources"
	default:
		return "Working"
	}
}

func (deps Deps) restoreReplyBurst(agent store.Agent, marker string, restoreEnv, restoreArgs []string, generation, final string, runErr error, cancelled bool) {
	deps.updateBurst(agent.ID, generation, func(s *BurstState) {
		s.Phase = burstRestoring
		s.Activity = "Returning to the terminal"
		if final != "" {
			s.Output = appendBurstOutput("", final)
		}
	})
	deps.Runtime.Stop(agent.ID)
	_ = releaseBurstMarker(marker)

	restoreErr := restoreWithFallback(15*time.Second,
		func(ctx context.Context) error {
			return waitForRestoredPane(ctx, deps.Tmux, tmux.SessionName(agent.ID), marker)
		},
		func(ctx context.Context) error {
			_, cwd, err := deps.agentHome(agent)
			if err != nil {
				return err
			}
			if err := deps.Tmux.RespawnPaneEnv(ctx, tmux.SessionName(agent.ID), cwd, restoreEnv, deps.AgentCmd, restoreArgs...); err != nil {
				return err
			}
			// A dead holder cannot consume the release marker. Direct respawn has
			// now replaced it, so this path owns clearing the obsolete lease.
			if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		},
	)
	if restoreErr == nil {
		_ = os.Remove(marker) // holder normally did this; direct-respawn fallback did not
		_ = deps.Store.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive")
	} else {
		_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusStopped)
	}
	if cancelled && restoreErr == nil {
		deps.clearBurst(agent.ID, generation)
		return
	}
	if runErr != nil || restoreErr != nil {
		message := "The reply could not finish. Your terminal is ready again."
		if restoreErr != nil {
			message = "The terminal could not be restored automatically. Open the TUI to continue."
		}
		deps.updateBurst(agent.ID, generation, func(s *BurstState) {
			s.Phase = burstFailed
			s.Activity = "Reply stopped"
			s.Error = message
			s.TerminalUnavailable = restoreErr != nil
		})
		return
	}
	deps.updateBurst(agent.ID, generation, func(s *BurstState) {
		s.Phase = burstDone
		s.Activity = "Reply complete"
	})
	// Leave enough time to read Done, then let the surface's delayed exit
	// animation finish before the terminal is revealed.
	time.Sleep(1200 * time.Millisecond)
	deps.clearBurst(agent.ID, generation)
}

func restoreWithFallback(timeout time.Duration, wait, fallback func(context.Context) error) error {
	waitCtx, waitCancel := context.WithTimeout(context.Background(), timeout)
	err := wait(waitCtx)
	waitCancel()
	if err == nil {
		return nil
	}
	// The crash-safe holder should normally exec Pi. Give the direct respawn
	// its own deadline: reusing the expired holder-wait context would make this
	// recovery branch a no-op.
	fallbackCtx, fallbackCancel := context.WithTimeout(context.Background(), timeout)
	defer fallbackCancel()
	if err = fallback(fallbackCtx); err != nil {
		return err
	}
	return wait(fallbackCtx)
}

func waitForRestoredPane(ctx context.Context, manager *tmux.Manager, name, marker string) error {
	if manager == nil {
		return errors.New("terminal integration unavailable")
	}
	// pane_current_command reports the interpreter for script-backed CLIs
	// (`node` for the npm pi executable), not the invoked command name. The
	// holder's stronger contract is its lease marker: it removes the marker
	// immediately before exec. Require that transition plus a live pane twice
	// so an exec failure cannot be mistaken for a restored TUI.
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	stable := 0
	var lastErr error
	for {
		has, paneErr := manager.HasSession(ctx, name)
		_, markerErr := os.Stat(marker)
		markerGone := errors.Is(markerErr, os.ErrNotExist)
		if markerErr != nil && !markerGone {
			lastErr = markerErr
		} else if paneErr != nil {
			lastErr = paneErr
		} else {
			lastErr = nil
		}
		if paneErr == nil && has && markerGone {
			stable++
			if stable >= 2 {
				return nil
			}
		} else {
			stable = 0
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		case <-t.C:
		}
	}
}

func (deps Deps) cancelBurstAndWait(ctx context.Context, agentID string) (func(), error) {
	// Acquire before observing the current generation. Reserve checks the same
	// coordinator lock, so no new reply can slip between restoration and the
	// caller's pane/session mutation. The caller releases after that mutation.
	release := deps.Bursts.BeginControl(agentID)
	if err := deps.cancelBurstAndWaitHeld(ctx, agentID); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

// cancelBurstAndWaitHeld performs the handoff while its caller already owns a
// control guard (forced restart uses an exclusive guard to reject double-clicks).
func (deps Deps) cancelBurstAndWaitHeld(ctx context.Context, agentID string) error {
	st := deps.Bursts.Snapshot(agentID)
	if st == nil {
		return nil
	}
	generation := st.Generation
	deps.Bursts.Cancel(agentID, generation)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		st = deps.Bursts.Snapshot(agentID)
		if st == nil || st.Generation != generation {
			return nil
		}
		if st.Phase == burstDone || st.Phase == burstFailed {
			_, cleared := deps.Bursts.Cancel(agentID, generation)
			if cleared {
				deps.publishBurst(BurstState{AgentID: agentID, Generation: generation, Phase: burstIdle, TerminalUnavailable: st.TerminalUnavailable})
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func handleBurstCancel(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Generation string `json:"generation"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		req.Generation = strings.TrimSpace(req.Generation)
		if req.Generation == "" {
			writeErr(w, http.StatusBadRequest, "generation is required")
			return
		}
		agentID := r.PathValue("id")
		previous := deps.Bursts.Snapshot(agentID)
		if previous != nil && previous.Generation == req.Generation && previous.Phase == burstFailed && previous.TerminalUnavailable {
			// Return performs an explicit restart next. Preserve the card until
			// that route succeeds, so a failed recovery still has a real action.
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "cancelling": false, "restartRequired": true})
			return
		}
		requested, cleared := deps.Bursts.Cancel(agentID, req.Generation)
		if cleared {
			idle := BurstState{AgentID: agentID, Generation: req.Generation, Phase: burstIdle}
			if previous != nil && previous.Generation == req.Generation {
				idle.TerminalUnavailable = previous.TerminalUnavailable
			}
			deps.publishBurst(idle)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "cancelling": requested})
	}
}
