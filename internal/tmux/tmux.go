// Package tmux manages tmux sessions for interactive Pi agents
// (ADR-0002: dual-channel control). Sessions created by PiCode carry a
// `picode-` name prefix — we only ever list and kill our own sessions.
package tmux

import (
	"context"
	"fmt"
	"os/exec"
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

// Manager wraps tmux CLI operations. All methods are safe for concurrent
// use (each spawns its own tmux invocation).
type Manager struct{}

// New returns a Manager.
func New() *Manager { return &Manager{} }

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
	cmd := exec.CommandContext(ctx, "tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// HasSession reports whether a tmux session with the given name exists.
// The "=" prefix forces exact-name matching so dots in names can't be
// parsed as session.window targets.
func (m *Manager) HasSession(ctx context.Context, name string) (bool, error) {
	out, err := m.run(ctx, "has-session", "-t", "="+name)
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
	if exists, err := m.HasSession(ctx, name); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("tmux session %q already exists", name)
	}
	full := []string{"new-session", "-d", "-s", name, "-c", cwd,
		"-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor"}
	for _, e := range extraEnv {
		if e == "" || !strings.Contains(e, "=") || strings.ContainsAny(e, "\n\x00") {
			continue
		}
		full = append(full, "-e", e)
	}
	full = append(full, "--", command)
	full = append(full, args...)
	if _, err := m.run(ctx, full...); err != nil {
		return err
	}
	_ = m.EnsureExtendedKeys(ctx)
	// PiCode owns the surface: the tmux status line would render as a green
	// bar at the bottom of the web terminal. Terminals turn it off per
	// session; do it here so agent TUI sessions match from the first paint.
	_ = m.SetOption(ctx, name, "status", "off")
	return nil
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
func (m *Manager) SetEnv(ctx context.Context, name, key, value string) error {
	_, err := m.run(ctx, "set-environment", "-t", name+":", key, value)
	return err
}

// SetOption sets a session option (e.g. status off on project shells).
func (m *Manager) SetOption(ctx context.Context, name, key, value string) error {
	// set-option treats "=" as a pane name (same as send-keys); use "name:".
	_, err := m.run(ctx, "set-option", "-t", name+":", key, value)
	return err
}

// KillSession terminates the session. Killing a missing session is a no-op.
func (m *Manager) KillSession(ctx context.Context, name string) error {
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
func (m *Manager) ListSessions(ctx context.Context) ([]Session, error) {
	out, err := m.run(ctx, "list-sessions", "-F",
		"#{session_name}\t#{session_created}\t#{session_attached}\t#{session_windows}")
	if err != nil {
		// No server running means no sessions at all.
		if strings.Contains(out, "no server running") || strings.Contains(out, "error connecting") {
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
func (m *Manager) SendKeys(ctx context.Context, name string, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	args := append([]string{"send-keys", "-t", name + ":"}, keys...)
	_, err := m.run(ctx, args...)
	return err
}

// PaneCwd returns the current pane's working directory (#{pane_current_path}).
func (m *Manager) PaneCwd(ctx context.Context, name string) (string, error) {
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
func (m *Manager) CaptureTail(ctx context.Context, name string, n int) (string, error) {
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
