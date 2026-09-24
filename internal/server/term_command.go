package server

// Pane command detection (ADR-0212). A coding CLI's hooks report its turn,
// but a shell command the person types with "!" fires no start hook in
// Claude Code, Codex, Grok or Hermes, so a minutes-long `!make deploy` sat
// under a "Ready" pill. The process tree does see it: every measured CLI
// starts a command as its own child, and those children differ from the
// CLI's long-lived helpers (MCP servers, LSPs, code-mode sidecars) in one
// way the kernel records — a command is detached into a session of its own
// (Claude, Codex, Grok, OpenCode, Pi, Omp) or reads the terminal on stdin
// (Hermes runs `sh -c` in the pane), while a helper stays in the pane's
// session behind a pipe or socket. This is an observation beside the hook
// state, never a hook state: it cannot start a turn or end one.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// commandMinAge hides the short children a CLI spawns on its own between
// keystrokes — hook reporters, the status line — which finish well under a
// second and would otherwise flash the pill.
const commandMinAge = 2 * time.Second

// clockTicks is USER_HZ, fixed at 100 on every Linux ABI PiCode runs on.
const clockTicks = 100

// TermCommand is the shell command a terminal's CLI is running.
type TermCommand struct {
	Name  string    `json:"name"`
	PID   int       `json:"-"`
	Start uint64    `json:"-"`
	Since time.Time `json:"since"`
}

func readUptimeTicks() uint64 {
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return uint64(secs * clockTicks)
}

// stdinIsTerminal says whether pid reads a terminal on stdin.
func (s *procSnapshot) stdinIsTerminal(pid int) bool {
	if s.ttyIn == nil {
		s.ttyIn = map[int]bool{}
	}
	if v, ok := s.ttyIn[pid]; ok {
		return v
	}
	target, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/fd/0")
	v := err == nil && (strings.HasPrefix(target, "/dev/pts/") || strings.HasPrefix(target, "/dev/tty"))
	s.ttyIn[pid] = v
	return v
}

func (s *procSnapshot) children() map[int][]int {
	out := make(map[int][]int, len(s.ppid))
	for pid, ppid := range s.ppid {
		out[ppid] = append(out[ppid], pid)
	}
	for _, kids := range out {
		sort.Ints(kids)
	}
	return out
}

// isCLIProc reports whether argv is one of the supported CLIs (or the
// wrapper shell driving one): the chain a command hangs from.
func isCLIProc(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	if normalizeTerminalCLI(filepath.Base(argv[0])) != "" {
		return true
	}
	return len(argv) > 1 && interpreters[strings.TrimSuffix(filepath.Base(argv[0]), ".exe")] &&
		normalizeTerminalCLI(filepath.Base(argv[1])) != ""
}

// commandWrappers are launchers a command runs through; the label names
// what they run instead (`bash -c make deploy` reads "make").
var commandWrappers = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true, "fish": true,
	"env": true, "timeout": true, "nice": true, "nohup": true, "stdbuf": true,
	"systemd-inhibit": true, "setsid": true,
}

// shellCommand is `sh -c …`: the only shape a command takes when it stays
// in the pane's session (Hermes). Asking for the shell, not just a terminal
// on stdin, keeps a CLI process under an unfamiliar name from reading as its
// own command.
func shellCommand(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	switch filepath.Base(argv[0]) {
	case "sh", "bash", "dash", "zsh", "fish":
		return strings.HasPrefix(argv[1], "-") && strings.Contains(argv[1], "c")
	}
	return false
}

func commandLabel(root int, snap *procSnapshot, kids map[int][]int) string {
	queue := []int{root}
	for visited := 0; len(queue) > 0 && visited < 32; visited++ {
		pid := queue[0]
		queue = queue[1:]
		if argv := snap.argv[pid]; len(argv) > 0 {
			if name := filepath.Base(argv[0]); !commandWrappers[name] {
				return name
			}
		}
		queue = append(queue, kids[pid]...)
	}
	if argv := snap.argv[root]; len(argv) > 0 {
		return filepath.Base(argv[0])
	}
	return ""
}

// runningCommand finds the oldest command running under the CLI rooted at
// root (the wrapper, the CLI itself, or the pane shell for a fallback
// lease). The CLI chain is walked through processes that are the CLI; the
// first process off the chain is a helper or a command, and the walk never
// descends into either — a helper's own detached child (gopls forking a
// daemon) is not the CLI's command.
func runningCommand(root int, snap *procSnapshot) (TermCommand, bool) {
	if root <= 0 || snap == nil || len(snap.argv) == 0 {
		return TermCommand{}, false
	}
	paneSID, ok := snap.sid[root]
	if !ok {
		return TermCommand{}, false
	}
	kids := snap.children()
	minAge := uint64(commandMinAge.Seconds() * clockTicks)
	var best TermCommand
	found := false
	chain := []int{root}
	for visited := 0; len(chain) > 0 && visited < 64; visited++ {
		pid := chain[0]
		chain = chain[1:]
		for _, kid := range kids[pid] {
			argv := snap.argv[kid]
			if len(argv) == 0 {
				continue
			}
			if isCLIProc(argv) {
				chain = append(chain, kid)
				continue
			}
			start, ok := snap.start[kid]
			if !ok || snap.uptime < start || snap.uptime-start < minAge {
				continue
			}
			if snap.sid[kid] == paneSID && !(shellCommand(argv) && snap.stdinIsTerminal(kid)) {
				continue // a helper: MCP server, LSP, sidecar — or the CLI under a name we do not know
			}
			if found && start >= best.Start {
				continue
			}
			best = TermCommand{Name: commandLabel(kid, snap, kids), PID: kid, Start: start}
			found = true
		}
	}
	if found {
		best.Since = time.Now().Add(-time.Duration(snap.uptime-best.Start) * time.Second / clockTicks)
	}
	return best, found
}

// Command returns the command a terminal's CLI is running.
func (r *TermRuntimes) Command(termID string) (TermCommand, bool) {
	if r == nil {
		return TermCommand{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd, ok := r.cmds[termID]
	return cmd, ok
}

// setCommand records the observation and says whether it changed. The same
// process observed again is no change, so its since stays the first sighting.
func (r *TermRuntimes) setCommand(termID string, cmd TermCommand, present bool) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	prev, had := r.cmds[termID]
	if !present {
		if !had {
			return false
		}
		delete(r.cmds, termID)
		return true
	}
	if had && prev.PID == cmd.PID && prev.Start == cmd.Start {
		return false
	}
	if r.cmds == nil {
		r.cmds = map[string]TermCommand{}
	}
	r.cmds[termID] = cmd
	return true
}

// observeTermCommand folds one reading into the registry and announces a
// change as the ephemeral terminal.command event (ADR-0048).
func observeTermCommand(deps Deps, termID string, cmd TermCommand, present bool) {
	if !deps.TermRuntimes.setCommand(termID, cmd, present) {
		return
	}
	if deps.Feed == nil {
		return
	}
	data := map[string]any{"termId": termID, "command": nil}
	if present {
		data["command"] = map[string]any{"name": cmd.Name, "since": cmd.Since}
	}
	deps.Feed.Ephemeral("terminal.command", data)
}

func applyTermCommand(deps Deps, view map[string]any, termID string) {
	if cmd, ok := deps.TermRuntimes.Command(termID); ok {
		view["command"] = map[string]any{"name": cmd.Name, "since": cmd.Since}
	}
}
