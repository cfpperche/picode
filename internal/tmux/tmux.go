// Package tmux manages tmux sessions for interactive Pi agents
// (ADR-0002: dual-channel control). Sessions created by PiCode carry a
// `picode-` name prefix — we only ever list and kill our own sessions.
package tmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Prefix marks tmux sessions owned by PiCode.
const Prefix = "picode-"

// ShellPrefix marks project shells (ADR-0016). Still under Prefix so
// OwnedSessionName and /ws/term accept them. Distinct from the interactive
// agent session (SessionName).
const ShellPrefix = "picode-sh-"

// Session describes a tmux session owned by PiCode.
type Session struct {
	Name     string    `json:"name"`
	Created  time.Time `json:"created"`
	Attached bool      `json:"attached"`
	Windows  int       `json:"windows"`
}

// OwnedSession is a PiCode-owned session with its pane facts — the flight-
// recorder view (ADR-0085): enough to tell, after a restart, what was
// running, through which root command, and whether it survived.
type OwnedSession struct {
	Name       string `json:"name"`
	PanePID    int    `json:"panePid"`
	RootCmd    string `json:"rootCmd"`
	Attached   bool   `json:"attached"`
	WorkingDir string `json:"workingDir,omitempty"`
}

// SocketDirEnv is tmux's own TMUX_TMPDIR: the directory tmux keeps its socket
// in, as `default` inside a `tmux-<uid>` subdirectory. A daemon leaves it
// alone — the user's real terminals and agents live on their own server, the
// one their own `tmux ls` shows. A harness that seeds its own store sets it,
// so **every** tmux call that process and its children make reaches a private
// server instead: the tests that create sessions, the helper scripts that
// shell out to `tmux` themselves, anything they spawn.
//
// Chosen over `tmux -L <name>` (a first attempt, 2026-09-13) because -L has to
// ride every argv: a fake tmux on PATH parses $1, a test that calls
// exec.Command("tmux", …) misses it entirely, and the resulting failures look
// like product bugs. TMUX_TMPDIR is inherited by everything, including
// subprocesses the harness never sees.
const SocketDirEnv = "TMUX_TMPDIR"

// DefaultSocketDir is where tmux puts the user's own server: tmux's docs
// order is $TMUX_TMPDIR, then the OS temp dir, then /tmp, always inside
// tmux-<uid>. KillIsolatedServer refuses this path.
func DefaultSocketDir() string {
	base := os.Getenv(SocketDirEnv)
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "tmux-"+strconv.Itoa(os.Getuid()))
}

// IsolatedEnv is the environment a tmux child needs to reach the server in
// dir and nowhere else. Measured 2026-09-15: $TMUX outranks TMUX_TMPDIR — a
// child that inherits the caller's TMUX talks to the caller's server no
// matter what TMUX_TMPDIR says (a guard fixture "isolated" this way ran
// kill-server on the live server twice). A pre-existing TMUX_TMPDIR is also
// dropped: tmux reads the first match, so an inherited one would win over
// the value appended here.
func IsolatedEnv(dir string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		switch {
		case strings.HasPrefix(kv, "TMUX="), strings.HasPrefix(kv, "TMUX_PANE="), strings.HasPrefix(kv, SocketDirEnv+"="):
			continue
		}
		env = append(env, kv)
	}
	return append(env, SocketDirEnv+"="+dir)
}

// KillIsolatedServer ends the tmux server whose socket lives in dir — the
// harness's own — and every session on it. It refuses an empty dir and the
// user's default socket directory, because that is where real work lives: a
// harness that kills it takes the human's terminals with it (it happened: an
// unguarded `kill-server` during a 2026-09-13 test run killed 140 sessions
// across someone else's working tree). The refusal is the point of this
// function existing at all; a harness that needs a server to kill must have
// made the directory itself.
func KillIsolatedServer(ctx context.Context, dir string) error {
	if dir == "" {
		return errors.New("tmux: refusing kill-server without a socket directory — that is the user's own server")
	}
	if filepath.Clean(dir) == filepath.Clean(DefaultSocketDir()) {
		return fmt.Errorf("tmux: refusing kill-server in %s — that is the user's own server", dir)
	}
	cmd := exec.CommandContext(ctx, "tmux", "kill-server")
	cmd.Env = IsolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil && (strings.Contains(string(out), "no server running") || strings.Contains(string(out), "error connecting")) {
		return nil // nothing to kill is the state a harness wants
	}
	if err != nil {
		return fmt.Errorf("tmux kill-server in %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Manager wraps tmux CLI operations. All methods are safe for concurrent
// use (each spawns its own tmux invocation).
type Manager struct {
	// exec runs the tmux binary. Tests script it to reproduce what a live
	// server cannot on demand: the client that lost the first-server race.
	exec func(ctx context.Context, stdin string, args ...string) ([]byte, error)
	// socket is the -S path every command carries (ADR-0139); empty keeps
	// tmux's default socket, which is what every pre-migration caller and
	// test uses.
	socket string
	// instance is this daemon's own identity (ADR-0140): the data directory
	// its socket lives in. Every session this Manager creates carries it as
	// MarkerInstanceEnv, so another PiCode reading the same server can tell
	// "a leftover" from "someone else's live work". Derived from the socket
	// path because that is the one place the two are already tied together.
	instance string
	// legacy is the drain's second server (ADR-0139): sessions created
	// before the socket move live there until they end. Nil outside a drain.
	legacy *Manager
}

// New returns a Manager on tmux's default socket.
func New() *Manager { return &Manager{exec: execTmux} }

// NewWithSocket returns a Manager whose every command carries -S path, so
// this instance's sessions live on their own server (ADR-0139). The socket's
// directory is this instance's identity (ADR-0140).
func NewWithSocket(path string) *Manager {
	return &Manager{exec: execTmux, socket: path, instance: filepath.Dir(path)}
}

// Instance is this daemon's own identity as stamped into the sessions it
// creates — empty on a Manager with no dedicated socket (tmux's default
// server is shared, so no single data directory owns it).
func (m *Manager) Instance() string { return m.instance }

func execTmux(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tmux", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return cmd.CombinedOutput()
}

// Available reports whether a tmux binary is on PATH.
func (m *Manager) Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// Version returns the tmux version string (e.g. "3.6").
func (m *Manager) Version() (string, error) {
	out, err := exec.Command("tmux", "-V").Output()
	if err != nil {
		return "", fmt.Errorf("tmux -V: %w", err)
	}
	v := strings.TrimSpace(string(out))
	return strings.TrimPrefix(v, "tmux "), nil
}

// OwnedSessionName reports whether name is a syntactically valid,
// PiCode-owned session name (prefix + non-empty sanitized id).
func OwnedSessionName(name string) bool {
	return strings.HasPrefix(name, Prefix) && len(name) > len(Prefix) && !strings.ContainsAny(name, ".:")
}

// SessionName derives the tmux session name for a workspace/agent id.
// Ids are sanitized to [a-z0-9-]: dots and colons are tmux target
// separators and would corrupt lookups (verified against tmux 3.6 —
// even the "=" exact-match prefix does not protect dotted names).
func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_', r == '-', r == '.', r == ' ', r == '/', r == ':':
			b.WriteRune('-')
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func clipName(name string) string {
	if len(name) > 60 { // tmux session name length cap (with margin)
		return name[:60]
	}
	return name
}

func SessionName(id string) string {
	return clipName(Prefix + sanitizeID(id))
}

// ShellSessionName is the one project shell for this agent (F1).
func ShellSessionName(agentID string) string {
	return clipName(ShellPrefix + sanitizeID(agentID))
}

// IsShellSession reports whether name is a project shell, not the Pi TUI.
func IsShellSession(name string) bool {
	return strings.HasPrefix(name, ShellPrefix) && OwnedSessionName(name)
}

func (m *Manager) run(ctx context.Context, args ...string) (string, error) {
	return m.runStdin(ctx, "", args...)
}

// runStdin feeds stdin to tmux (load-buffer reads the buffer content from it).
// The socket flag lives here, the one path every command takes.
func (m *Manager) runStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	execFn := m.exec
	if execFn == nil {
		execFn = execTmux
	}
	if m.socket != "" {
		args = append([]string{"-S", m.socket}, args...)
	}
	out, err := execFn(ctx, stdin, args...)
	if err != nil {
		return string(out), fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// serverStartRace is what a tmux client prints when it lost the race to
// start the first server: two clients on a socket with no server behind it,
// one server binds, the other exits, and the losing client sees its
// connection close with no reply. The command never ran, so the startup
// commands (has-session, new-session) retry it. Seen on a fresh CI runner
// where several test packages started tmux at once (2026-09-09); on a
// workstation it is two terminals created together after a reboot.
const serverStartRace = "server exited unexpectedly"

func (m *Manager) runStartup(ctx context.Context, args ...string) (string, error) {
	var out string
	var err error
	for attempt := 1; ; attempt++ {
		out, err = m.run(ctx, args...)
		if err == nil || attempt >= 3 || !strings.Contains(out, serverStartRace) {
			return out, err
		}
		select {
		case <-ctx.Done():
			return out, err
		case <-time.After(time.Duration(attempt) * 50 * time.Millisecond):
		}
	}
}

// HasSession reports whether a tmux session with the given name exists.
// The "=" prefix forces exact-name matching so dots in names can't be
// parsed as session.window targets.
func (m *Manager) hasSession(ctx context.Context, name string) (bool, error) {
	out, err := m.runStartup(ctx, "has-session", "-t", "="+name)
	if err == nil {
		return true, nil
	}
	// tmux exits 1 for both "no such session" and real failures; the
	// message distinguishes them. A stopped server also means "no".
	notThere := []string{
		"can't find session", "no such session",
		"can't find window", "can't find pane", // dotted-name lookups miss here
		"no server running", "error connecting to",
	}
	for _, msg := range notThere {
		if strings.Contains(out, msg) {
			return false, nil
		}
	}
	return false, fmt.Errorf("tmux has-session %q: %s", name, out)
}

// NewSession creates a detached session named name, rooted at cwd, running
// the given command. It errors if the session already exists.
func (m *Manager) NewSession(ctx context.Context, name, cwd string, command string, args ...string) error {
	return m.NewSessionEnv(ctx, name, cwd, nil, command, args...)
}

// NewSessionEnv is NewSession plus extra KEY=VALUE entries (tmux -e).
func (m *Manager) NewSessionEnv(ctx context.Context, name, cwd string, extraEnv []string, command string, args ...string) error {
	return m.NewSessionEnvSize(ctx, name, cwd, 0, 0, extraEnv, command, args...)
}

// NewSessionEnvSize preserves the previous pane geometry when replacing an
// idle CLI. A detached replacement otherwise starts at tmux's 80x24 default.
func (m *Manager) NewSessionEnvSize(ctx context.Context, name, cwd string, width, height int, extraEnv []string, command string, args ...string) error {
	if exists, err := m.HasSession(ctx, name); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("tmux session %q already exists", name)
	}
	full := []string{"new-session", "-d", "-s", name, "-c", cwd,
		"-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor"}
	if width > 0 && height > 0 {
		full = append(full, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height))
	}
	for _, e := range extraEnv {
		if e == "" || !strings.Contains(e, "=") || strings.ContainsAny(e, "\n\x00") {
			continue
		}
		full = append(full, "-e", e)
	}
	// The instance stamp (ADR-0140) rides the same -e: a caller cannot
	// supply its own, because "whose session is this" is not a caller's claim
	// to make.
	if m.instance != "" {
		full = append(full, "-e", MarkerInstanceEnv+"="+m.instance)
	}
	full = append(full, "--", command)
	full = append(full, args...)
	if _, err := m.runStartup(ctx, full...); err != nil {
		return err
	}
	_ = m.EnsureExtendedKeys(ctx)
	// PiCode owns the surface: the tmux status line would render as a green
	// bar at the bottom of the web terminal. Terminals turn it off per
	// session; do it here so agent TUI sessions match from the first paint.
	_ = m.SetOption(ctx, name, "status", "off")
	return nil
}

// RespawnPaneEnv replaces the process in an existing session's active pane
// without destroying the tmux session or detaching its browser client. tmux's
// respawn-pane accepts one shell-command string, so every argv element is
// POSIX-shell quoted before it crosses that boundary.
func (m *Manager) respawnPaneEnv(ctx context.Context, name, cwd string, extraEnv []string, command string, args ...string) error {
	if exists, err := m.HasSession(ctx, name); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("tmux session %q does not exist", name)
	}
	full := []string{"respawn-pane", "-k", "-t", name + ":", "-c", cwd}
	for _, e := range extraEnv {
		if e == "" || !strings.Contains(e, "=") || strings.ContainsAny(e, "\n\x00") {
			continue
		}
		full = append(full, "-e", e)
	}
	argv := append([]string{command}, args...)
	quoted := make([]string, 0, len(argv))
	for _, arg := range argv {
		quoted = append(quoted, shellQuote(arg))
	}
	full = append(full, "--", strings.Join(quoted, " "))
	_, err := m.run(ctx, full...)
	return err
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// EnsureExtendedKeys turns on tmux extended keys so Shift+Enter survives
// attach. Format is **xterm** (modifyOtherKeys), not csi-u: probed live —
// tmux 3.6 answers only DA1 to a pane's Kitty query (`CSI ? u`), so pi
// falls back to modifyOtherKeys (`CSI > 4;2m`) and expects
// `ESC [27;2;13~`. tmux re-encodes client keys per this server option.
// Best-effort: older tmux without extended-keys-format is ignored.
func (m *Manager) EnsureExtendedKeys(ctx context.Context) error {
	if _, err := m.run(ctx, "set-option", "-s", "extended-keys", "on"); err != nil {
		return err
	}
	_, err := m.run(ctx, "set-option", "-s", "extended-keys-format", "xterm")
	return err
}

// SetEnv sets a session environment variable (tmux set-environment).
func (m *Manager) setEnv(ctx context.Context, name, key, value string) error {
	_, err := m.run(ctx, "set-environment", "-t", name+":", key, value)
	return err
}

// SetOption sets a session option (e.g. status off on project shells).
func (m *Manager) setOption(ctx context.Context, name, key, value string) error {
	// set-option treats "=" as a pane name (same as send-keys); use "name:".
	_, err := m.run(ctx, "set-option", "-t", name+":", key, value)
	return err
}

// KillSession terminates the session. Killing a missing session is a no-op.
func (m *Manager) killSession(ctx context.Context, name string) error {
	exists, err := m.HasSession(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = m.run(ctx, "kill-session", "-t", "="+name)
	return err
}

// ListSessions returns PiCode-owned tmux sessions (prefix filter).
func (m *Manager) listSessions(ctx context.Context) ([]Session, error) {
	out, err := m.run(ctx, "list-sessions", "-F",
		"#{session_name}\t#{session_created}\t#{session_attached}\t#{session_windows}")
	if err != nil {
		// No server running means no sessions at all (serverAbsent also
		// covers the client-started-a-just-dead-server race).
		if serverAbsent(out) {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions: %s", out)
	}
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 4 || !strings.HasPrefix(parts[0], Prefix) {
			continue
		}
		createdUnix, _ := strconv.ParseInt(parts[1], 10, 64)
		windows, _ := strconv.Atoi(parts[3])
		sessions = append(sessions, Session{
			Name:     parts[0],
			Created:  time.Unix(createdUnix, 0).UTC(),
			Attached: parts[2] == "1",
			Windows:  windows,
		})
	}
	return sessions, nil
}

// SendKeys types into the session's current pane (used to drive `/login` — ADR-0009).
// Target is "name:" (session), not "=name" — send-keys treats "=" as an exact
// pane name, which is why `/login` was alerting "can't find pane".
func (m *Manager) sendKeys(ctx context.Context, name string, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	args := append([]string{"send-keys", "-t", name + ":"}, keys...)
	_, err := m.run(ctx, args...)
	return err
}

// TypeText types text into the session's pane as literal keystrokes — no key
// names, no Enter (ADR-0078): the rail pre-fills a command that the human
// reads and submits in a terminal they can see.
func (m *Manager) typeText(ctx context.Context, name, text string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("tmux type: empty session name")
	}
	if text == "" {
		return nil
	}
	_, err := m.run(ctx, "send-keys", "-l", "-t", name+":", text)
	return err
}

// ClearLine empties whatever is already typed at a shell prompt before
// something else is typed into it.
//
// Without it a prepared command that the human never submitted stays on the
// line and the next one lands glued to its end — measured: a prepared
// `git switch -c …17210e9` and a later `git worktree add …` arrived as one
// word and the shell answered "fatal: only one reference expected"
// (ADR-0096). End-of-line then kill-to-start: verified in bash and zsh
// (readline/zle, emacs and vi insert mode), fish, and a shell with no line
// editor at all, where C-u is the tty's own kill character. nu and pwsh are
// in isShell's list and were not verified; there the worst case is the old
// behaviour, a line that keeps what it held.
//
// The cost, stated: anything the human was typing at that prompt and had not
// submitted is discarded. Before this it was corrupted instead — the command
// landed on its end and the shell ran the concatenation.
//
// Callers must already have established that the pane sits at a shell prompt;
// this sends no signal and interrupts nothing.
func (m *Manager) clearLine(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("tmux clear-line: empty session name")
	}
	_, err := m.run(ctx, "send-keys", "-t", name+":", "C-e", "C-u")
	return err
}

// PasteText inserts text into the session's pane as a bracketed paste
// (ADR-0060 reply fallback): the target editor inserts it wholesale, so no
// keybinding fires and newlines stay literal, then presses Enter to submit.
// A named buffer keeps the user's own copy buffer untouched.
func (m *Manager) pasteText(ctx context.Context, name, text string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("tmux paste: empty session name")
	}
	if _, err := m.runStdin(ctx, text, "load-buffer", "-b", "picode-reply", "-"); err != nil {
		return err
	}
	if _, err := m.run(ctx, "paste-buffer", "-p", "-b", "picode-reply", "-t", name+":"); err != nil {
		return err
	}
	_, err := m.run(ctx, "delete-buffer", "-b", "picode-reply")
	if err != nil {
		return err
	}
	_, err = m.run(ctx, "send-keys", "-t", name+":", "Enter")
	return err
}

// PaneCommand returns tmux's current command name for the active pane.
func (m *Manager) paneCommand(ctx context.Context, name string) (string, error) {
	out, err := m.run(ctx, "display-message", "-p", "-t", name+":", "#{pane_current_command}")
	if err != nil {
		return "", err
	}
	command := strings.TrimSpace(out)
	if command == "" {
		return "", fmt.Errorf("tmux pane command empty")
	}
	return command, nil
}

// PanePID returns the PID of the first process in the active pane. It is a
// reconciliation hint only; a runtime wrapper supplies the stronger process
// start token used to reject PID reuse.
func (m *Manager) panePID(ctx context.Context, name string) (int, error) {
	out, err := m.run(ctx, "display-message", "-p", "-t", name+":", "#{pane_pid}")
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("tmux pane pid invalid: %q", strings.TrimSpace(out))
	}
	return pid, nil
}

// PaneSessionID returns tmux's immutable session identity (for example $12).
// Unlike the name, it proves a pane respawn did not kill and recreate the
// terminal container.
func (m *Manager) paneSessionID(ctx context.Context, name string) (string, error) {
	out, err := m.run(ctx, "display-message", "-p", "-t", name+":", "#{session_id}")
	return strings.TrimSpace(out), err
}

// PaneCwd returns the current pane's working directory (#{pane_current_path}).
func (m *Manager) paneCwd(ctx context.Context, name string) (string, error) {
	out, err := m.run(ctx, "display-message", "-p", "-t", name+":", "#{pane_current_path}")
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(out)
	if p == "" {
		return "", fmt.Errorf("tmux pane cwd empty")
	}
	return p, nil
}

// CaptureTail returns the last n lines of the session's active pane
// (tmux capture-pane -p). Used to read the pi TUI's own state, e.g. its
// "Working…" spinner when the agent is driven from the terminal.
func (m *Manager) captureTail(ctx context.Context, name string, n int) (string, error) {
	if n <= 0 {
		n = 8
	}
	out, err := m.run(ctx, "capture-pane", "-p", "-t", name+":", "-S", "-"+strconv.Itoa(n))
	if err != nil {
		return "", err
	}
	return out, nil
}

// LooksWorking reports whether a captured TUI tail shows pi's busy state.
//
// pi renders its working indicator as its own pane line —
// "⠹ Working...", cycling the braille frames below (~10x/s) — and
// disposes it when the turn ends. Nothing else in the TUI uses braille,
// so a tail line whose first non-space rune is a frame means working,
// whatever the conversation text says: a bare substring "working" match
// lit up on replies that merely mentioned the word (an agent writing
// about this very feature flagged itself as busy while idle,
// 2026-09-02). Frames mirror pi 0.84.4
// (dist/bundle/chunks/chunk-OMWWHBTG.js: DEFAULT_FRAMES) — re-check them
// when upgrading pi. Not matching the "Working..." message keeps the
// detection immune to a renamed working message; a custom indicator
// without braille degrades to false-negative (benign: the badge stays
// idle, it never inflames).
var piSpinnerFrames = map[rune]bool{
	'⠋': true, '⠙': true, '⠹': true, '⠸': true, '⠼': true,
	'⠴': true, '⠦': true, '⠧': true, '⠇': true, '⠏': true,
}

func LooksWorking(captured string) bool {
	for _, line := range strings.Split(captured, "\n") {
		for _, r := range line {
			if r == ' ' || r == '\t' {
				continue
			}
			if piSpinnerFrames[r] {
				return true
			}
			break // first non-space rune is not a frame — next line
		}
	}
	return false
}

// ExtendedKeysFormat returns the server's `extended-keys-format` option
// value ("csi-u", "xterm", ...). Pi recommends `csi-u` so modifier keys
// (Shift+Enter, Ctrl+Enter) survive the hop (see Pi's tmux docs).
// An error means no tmux server is running yet.
func (m *Manager) ExtendedKeysFormat(ctx context.Context) (string, error) {
	out, err := m.run(ctx, "show-options", "-gv", "extended-keys-format")
	if err != nil {
		return "", fmt.Errorf("tmux show-options: %s", strings.TrimSpace(out))
	}
	return strings.TrimSpace(out), nil
}

// ListOwned returns every PiCode-owned session with its pane facts
// (ADR-0085 flight recorder). A server that is not running yields an empty
// list, never an error.
func (m *Manager) listOwned(ctx context.Context) ([]OwnedSession, error) {
	out, err := m.run(ctx, "list-panes", "-a", "-F",
		"#{session_name}\t#{pane_pid}\t#{pane_current_command}\t#{session_attached}\t#{pane_current_path}")
	if err != nil {
		if strings.Contains(out, "no server running") || strings.Contains(out, "error connecting") {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-panes: %s", out)
	}
	var owned []OwnedSession
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 5 || !OwnedSessionName(parts[0]) || seen[parts[0]] {
			continue
		}
		seen[parts[0]] = true
		pid, _ := strconv.Atoi(parts[1])
		owned = append(owned, OwnedSession{
			Name:       parts[0],
			PanePID:    pid,
			RootCmd:    parts[2],
			Attached:   parts[3] == "1",
			WorkingDir: parts[4],
		})
	}
	return owned, nil
}
