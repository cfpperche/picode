package server

// Terminal runtime presence (ADR-0062). Lifecycle state tells whether a CLI
// is working; this registry tells whether a supported CLI still owns the
// terminal pane. Both are ephemeral and keyed by the wrapper's runId.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

const runtimeRunIDCap = 160

// TermRuntime identifies the live CLI process in a project terminal.
type TermRuntime struct {
	CLI       string    `json:"cli"`
	Source    string    `json:"source"`
	RunID     string    `json:"runId"`
	PID       int       `json:"pid,omitempty"`
	ProcStart string    `json:"-"`
	StartedAt time.Time `json:"startedAt"`
}

// TermRuntimes holds one active CLI lease per terminal. It is intentionally
// not persisted: after a daemon restart the wrapper or the tmux reconciler
// must prove the process again before the UI calls it a CLI.
type TermRuntimes struct {
	mu sync.Mutex
	m  map[string]TermRuntime
}

// NewTermRuntimes builds an empty runtime registry.
func NewTermRuntimes() *TermRuntimes {
	return &TermRuntimes{m: map[string]TermRuntime{}}
}

// Get returns the current lease for a terminal.
func (r *TermRuntimes) Get(termID string) (TermRuntime, bool) {
	if r == nil {
		return TermRuntime{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rt, ok := r.m[termID]
	return rt, ok
}

// Start installs or refreshes a lease. Repeating the exact same lease is not
// a change and therefore does not need another feed event.
func (r *TermRuntimes) Start(termID string, runtime TermRuntime) (TermRuntime, bool) {
	if r == nil {
		return runtime, false
	}
	if len(runtime.RunID) > runtimeRunIDCap {
		runtime.RunID = runtime.RunID[:runtimeRunIDCap]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = map[string]TermRuntime{}
	}
	prev, had := r.m[termID]
	if had && prev.CLI == runtime.CLI && prev.Source == runtime.Source && prev.RunID == runtime.RunID && prev.PID == runtime.PID && prev.ProcStart == runtime.ProcStart {
		return prev, false
	}
	r.m[termID] = runtime
	return runtime, true
}

// End removes a lease only when runID owns it. An empty runID is accepted for
// the legacy fallback, but a non-empty stale end can never erase a newer run.
func (r *TermRuntimes) End(termID, runID string) (TermRuntime, bool) {
	if r == nil {
		return TermRuntime{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	prev, ok := r.m[termID]
	if !ok || (runID != "" && prev.RunID != "" && prev.RunID != runID) {
		return prev, false
	}
	delete(r.m, termID)
	return prev, true
}

// Drop forgets a terminal without checking its run identity.
func (r *TermRuntimes) Drop(termID string) (TermRuntime, bool) {
	return r.End(termID, "")
}

// Snapshot returns a copy for the reconciliation watcher.
func (r *TermRuntimes) Snapshot() map[string]TermRuntime {
	out := map[string]TermRuntime{}
	if r == nil {
		return out
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, runtime := range r.m {
		out[id] = runtime
	}
	return out
}

func normalizeTerminalCLI(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "claude", "claude-code":
		return "claude-code"
	case "codex":
		return "codex"
	case "grok":
		return "grok"
	case "pi":
		return "pi"
	default:
		return ""
	}
}

func terminalCLIFromCommand(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return ""
	}
	return normalizeTerminalCLI(filepath.Base(fields[0]))
}

// procSnapshot maps every readable process id to its first two argv
// tokens and parent id. Reading /proc is inherently Linux-specific; on
// platforms without it the snapshot stays empty and reconciliation keeps
// the exact tmux command match as the only fallback.
type procSnapshot struct {
	argv map[int][]string
	ppid map[int]int
}

func readProcSnapshot() *procSnapshot {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return &procSnapshot{argv: map[int][]string{}, ppid: map[int]int{}}
	}
	snap := &procSnapshot{argv: make(map[int][]string, len(entries)), ppid: make(map[int]int, len(entries))}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		if raw, err := os.ReadFile("/proc/" + entry.Name() + "/cmdline"); err == nil {
			parts := strings.Split(string(raw), "\x00")
			argv := make([]string, 0, 2)
			for _, part := range parts {
				if part = strings.TrimSpace(part); part != "" {
					argv = append(argv, part)
				}
				if len(argv) == 2 {
					break
				}
			}
			if len(argv) > 0 {
				snap.argv[pid] = argv
			}
		}
		if raw, err := os.ReadFile("/proc/" + entry.Name() + "/stat"); err == nil {
			// Field 4 (ppid) sits after the comm field, which may contain
			// spaces and ')'. The final ')' is the safe delimiter.
			end := strings.LastIndexByte(string(raw), ')')
			if end >= 0 {
				fields := strings.Fields(string(raw)[end+2:])
				if len(fields) >= 2 {
					if ppid, err := strconv.Atoi(fields[1]); err == nil {
						snap.ppid[pid] = ppid
					}
				}
			}
		}
	}
	return snap
}

// identifyPaneCLIProcs walks the pane's process tree breadth-first and
// returns the first known CLI with its process id. Wrapper shells stay
// alive as the CLI's parent on purpose, so the pane command alone reads
// "sh" while a supported CLI still owns the pane. A process matches by
// its own name (native binaries) or, when it is an interpreter driving a
// script, by the script's basename — which also matches the wrapper shell
// itself, and the wrapper only lives while the CLI runs.
func identifyPaneCLIProcs(panePID int, snap *procSnapshot) (string, int) {
	if panePID <= 0 || snap == nil || len(snap.argv) == 0 {
		return "", 0
	}
	children := make(map[int][]int, len(snap.ppid))
	for pid, ppid := range snap.ppid {
		children[ppid] = append(children[ppid], pid)
	}
	for _, kids := range children {
		sort.Ints(kids)
	}
	queue := []int{panePID}
	visited := 0
	for len(queue) > 0 && visited < 64 {
		pid := queue[0]
		queue = queue[1:]
		visited++
		argv := snap.argv[pid]
		if len(argv) == 0 {
			continue
		}
		candidates := []string{filepath.Base(argv[0])}
		if len(argv) > 1 && interpreters[strings.TrimSuffix(filepath.Base(argv[0]), ".exe")] {
			candidates = append(candidates, filepath.Base(argv[1]))
		}
		for _, candidate := range candidates {
			if cli := normalizeTerminalCLI(candidate); cli != "" {
				return cli, pid
			}
		}
		queue = append(queue, children[pid]...)
	}
	return "", 0
}

// interpreters re-identify a CLI from the script it drives: the kernel
// records the interpreter as argv[0] for shebang scripts, and node-based
// CLIs run as `node <path>/claude|codex|grok`.
var interpreters = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true,
	"node": true, "bun": true, "deno": true,
	"python": true, "python3": true,
}

func processStartToken(pid int) string {
	if pid <= 0 {
		return ""
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return ""
	}
	// comm may contain spaces and ')' characters. The final ')' before the
	// state field is the safe delimiter; field 22 is index 19 after it.
	end := strings.LastIndexByte(string(raw), ')')
	if end < 0 || end+2 >= len(raw) {
		return ""
	}
	fields := strings.Fields(string(raw)[end+2:])
	if len(fields) <= 19 {
		return ""
	}
	return fields[19]
}

func processAlive(runtime TermRuntime) bool {
	if runtime.PID <= 0 {
		return false
	}
	if token := processStartToken(runtime.PID); token != "" {
		return runtime.ProcStart == "" || token == runtime.ProcStart
	}
	process, err := os.FindProcess(runtime.PID)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func applyTermRuntime(deps Deps, view map[string]any, termID string) {
	if deps.TermRuntimes == nil {
		return
	}
	if runtime, ok := deps.TermRuntimes.Get(termID); ok {
		view["cli"] = runtime.CLI
		view["tui"] = map[string]any{
			"cli":       runtime.CLI,
			"source":    runtime.Source,
			"runId":     runtime.RunID,
			"startedAt": runtime.StartedAt,
		}
	}
}

func publishTermRuntime(deps Deps, action string, runtime TermRuntime, termID string) {
	if deps.Feed == nil {
		return
	}
	data := map[string]any{
		"termId": termID,
		"action": action,
		"cli":    runtime.CLI,
		"source": runtime.Source,
		"runId":  runtime.RunID,
	}
	if !runtime.StartedAt.IsZero() {
		data["startedAt"] = runtime.StartedAt
	}
	deps.Feed.Ephemeral("terminal.runtime", data)
}

func clearRuntimeState(deps Deps, termID, runID string) {
	if deps.TermStates == nil {
		return
	}
	state, ok := deps.TermStates.Get(termID)
	if !ok || (runID != "" && state.RunID != "" && state.RunID != runID) {
		return
	}
	if !deps.TermStates.DropForRun(termID, runID) || deps.Feed == nil {
		return
	}
	deps.Feed.Ephemeral("terminal.state", map[string]any{"termId": termID, "state": nil, "runId": runID})
}

func registerTermRuntime(deps Deps, termID string, runtime TermRuntime) (TermRuntime, bool) {
	previous, had := deps.TermRuntimes.Get(termID)
	current, changed := deps.TermRuntimes.Start(termID, runtime)
	if !changed {
		return current, false
	}
	// A new wrapper run supersedes any unscoped legacy activity. A tmux
	// fallback, however, proves presence only and must not erase a valid
	// lifecycle report that arrived before it was discovered.
	if (had && previous.RunID != current.RunID) || (!had && current.Source == "wrapper") {
		clearRuntimeState(deps, termID, previous.RunID)
	}
	publishTermRuntime(deps, "started", current, termID)
	return current, true
}

func finishTermRuntime(deps Deps, termID, runID string) (TermRuntime, bool) {
	current, removed := deps.TermRuntimes.End(termID, runID)
	if !removed {
		return current, false
	}
	clearRuntimeState(deps, termID, current.RunID)
	publishTermRuntime(deps, "ended", current, termID)
	return current, true
}

func handleSetTerminalRuntime(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.TermRuntimes == nil {
			writeErr(w, http.StatusServiceUnavailable, "Terminal runtime presence is not available.")
			return
		}
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "Terminal runtime presence is not available.")
			return
		}
		id := r.PathValue("id")
		if _, err := deps.Store.GetTerminal(id); err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			Action string `json:"action"`
			CLI    string `json:"cli"`
			RunID  string `json:"runId"`
			PID    int    `json:"pid"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		runID := strings.TrimSpace(req.RunID)
		switch strings.ToLower(strings.TrimSpace(req.Action)) {
		case "start":
			cli := normalizeTerminalCLI(req.CLI)
			if cli == "" {
				writeErr(w, http.StatusBadRequest, "cli must be claude-code, codex, grok or pi")
				return
			}
			if runID == "" || len(runID) > runtimeRunIDCap {
				writeErr(w, http.StatusBadRequest, "runId is required")
				return
			}
			if req.PID <= 0 {
				writeErr(w, http.StatusBadRequest, "pid must be positive")
				return
			}
			runtime, _ := registerTermRuntime(deps, id, TermRuntime{
				CLI: cli, Source: "wrapper", RunID: runID,
				PID: req.PID, ProcStart: processStartToken(req.PID), StartedAt: time.Now(),
			})
			writeJSON(w, http.StatusOK, runtimeView(id, runtime))
		case "end":
			if runID == "" {
				writeErr(w, http.StatusBadRequest, "runId is required")
				return
			}
			runtime, _ := finishTermRuntime(deps, id, runID)
			writeJSON(w, http.StatusOK, runtimeView(id, runtime))
		default:
			writeErr(w, http.StatusBadRequest, "action must be start or end")
		}
	}
}

func runtimeView(termID string, runtime TermRuntime) map[string]any {
	return map[string]any{
		"termId": termID,
		"cli":    runtime.CLI,
		"source": runtime.Source,
		"runId":  runtime.RunID,
	}
}

// StartTermRuntimeWatch reconciles wrapper leases with the actual pane. It
// also recovers sessions created before the runId protocol: exact tmux
// command + PID can establish presence, but never an activity state.
func StartTermRuntimeWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.TermRuntimes == nil || deps.Store == nil || deps.Tmux == nil || !deps.Tmux.Available() {
		return
	}
	reconcileTermRuntimes(ctx, deps)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			reconcileTermRuntimes(ctx, deps)
		}
	}
}

func reconcileTermRuntimes(ctx context.Context, deps Deps) {
	terminals, err := deps.Store.ListTerminals()
	if err != nil {
		return
	}
	seen := map[string]bool{}
	var procSnap *procSnapshot
	for _, terminal := range terminals {
		id := terminal.ID
		name := tmux.ShellSessionName(id)
		has, err := deps.Tmux.HasSession(ctx, name)
		if err != nil || !has {
			if runtime, ok := deps.TermRuntimes.Get(id); ok {
				finishTermRuntime(deps, id, runtime.RunID)
			}
			continue
		}
		runtime, ok := deps.TermRuntimes.Get(id)
		if ok {
			// A legacy lease uses the pane shell PID, not the short-lived
			// wrapper PID. Re-check the pane still hosts this CLI — by exact
			// foreground command or, when the pane is held by a wrapper shell
			// ("sh"), by its process tree — so a fallback cannot survive the
			// CLI returning to bash or exiting.
			if runtime.Source == "tmux-fallback" {
				command, commandErr := deps.Tmux.PaneCommand(ctx, name)
				stillCLI := commandErr == nil && terminalCLIFromCommand(command) == runtime.CLI
				if !stillCLI && commandErr == nil {
					if pid, pidErr := deps.Tmux.PanePID(ctx, name); pidErr == nil && pid > 0 {
						if procSnap == nil {
							procSnap = readProcSnapshot()
						}
						foundCLI, _ := identifyPaneCLIProcs(pid, procSnap)
						stillCLI = foundCLI == runtime.CLI
					}
				}
				if !stillCLI {
					finishTermRuntime(deps, id, runtime.RunID)
					continue
				}
			}
			if !processAlive(runtime) {
				finishTermRuntime(deps, id, runtime.RunID)
				continue
			}
			seen[id] = true
			continue
		}
		command, err := deps.Tmux.PaneCommand(ctx, name)
		if err != nil {
			continue
		}
		cli := terminalCLIFromCommand(command)
		pid := 0
		if cli == "" {
			// A wrapper script keeps the pane command at "sh" while the CLI
			// it wrapped is still running (ADR-0062). Identify the CLI from
			// the pane's process tree — this also revives presence after a
			// daemon restart, when the in-memory wrapper announcement is
			// gone but the CLI still owns the pane.
			panePID, pidErr := deps.Tmux.PanePID(ctx, name)
			if pidErr != nil || panePID <= 0 {
				continue
			}
			if procSnap == nil {
				procSnap = readProcSnapshot()
			}
			cli, pid = identifyPaneCLIProcs(panePID, procSnap)
			if cli == "" {
				continue
			}
		} else {
			pid, err = deps.Tmux.PanePID(ctx, name)
			if err != nil || pid <= 0 {
				continue
			}
		}
		registerTermRuntime(deps, id, TermRuntime{
			CLI: cli, Source: "tmux-fallback", RunID: "tmux-" + id + "-" + strconv.Itoa(pid),
			PID: pid, ProcStart: processStartToken(pid), StartedAt: time.Now(),
		})
		seen[id] = true
	}
	for id, runtime := range deps.TermRuntimes.Snapshot() {
		if !seen[id] {
			if _, err := deps.Store.GetTerminal(id); errors.Is(err, store.ErrNotFound) {
				finishTermRuntime(deps, id, runtime.RunID)
			}
		}
	}
}
