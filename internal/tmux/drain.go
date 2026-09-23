package tmux

// The drain (ADR-0139): managed sessions moved to a per-instance socket, but
// sessions created before the move still live on the default server. tmux
// has no live migration, so the Manager keeps a second, legacy Manager and
// falls back to it whenever a session is not on the primary server. Every
// session-scoped method below is a wrapper around the private primary-only
// implementation (the same names in lower case); the public API and its
// callers are unchanged, and the wrappers disappear when the last legacy
// session ends.

import (
	"context"
	"time"
)

// WithLegacy gives this Manager a second server to consult for sessions the
// primary one does not have. It returns the receiver so wiring reads as one
// expression.
func (m *Manager) WithLegacy(legacy *Manager) *Manager {
	m.legacy = legacy
	return m
}

// SocketPath is the path a client must name (-S) to reach this Manager's
// server; empty means the caller's default socket.
func (m *Manager) SocketPath() string { return m.socket }

// SocketFor is the socket that can reach the named session right now: the
// primary's path in the normal case, the legacy's (empty, meaning the
// default socket) while a drain-bound session has not moved yet, and the
// primary's when the session is on neither — the caller's usual absent
// handling decides what that means.
func (m *Manager) SocketFor(ctx context.Context, name string) string {
	if m.legacy != nil {
		if has, err := m.hasSession(ctx, name); err == nil && !has {
			if lhas, lerr := m.legacy.hasSession(ctx, name); lerr == nil && lhas {
				return m.legacy.socket
			}
		}
	}
	return m.socket
}

// legacyFor reports whether session-scoped work for name belongs to the
// legacy server: one exists, the primary does not have the session, and the
// legacy one does. One extra has-session per operation is the price of a
// drain that needs no bookkeeping; it disappears with the drain.
func (m *Manager) legacyFor(ctx context.Context, name string) bool {
	if m.legacy == nil {
		return false
	}
	if has, err := m.hasSession(ctx, name); err == nil && has {
		return false
	}
	has, err := m.legacy.hasSession(ctx, name)
	return err == nil && has
}

// legacyPane retries a pane-id-addressed command on the legacy server. Pane
// ids are server-scoped and the caller only has the id, so this is trial:
// the primary is tried first and a failure falls through to the legacy one.
func (m *Manager) legacyPane(retry func() error, err error) error {
	if err == nil || m.legacy == nil {
		return err
	}
	if lerr := retry(); lerr == nil {
		return nil
	}
	return err
}

func (m *Manager) HasSession(ctx context.Context, name string) (bool, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.hasSession(ctx, name)
	}
	return m.hasSession(ctx, name)
}

func (m *Manager) SessionReceipt(ctx context.Context, name string) (SessionReceipt, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.sessionReceipt(ctx, name)
	}
	return m.sessionReceipt(ctx, name)
}

func (m *Manager) KillSession(ctx context.Context, name string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.killSession(ctx, name)
	}
	return m.killSession(ctx, name)
}

func (m *Manager) SendKeys(ctx context.Context, name string, keys ...string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.sendKeys(ctx, name, keys...)
	}
	return m.sendKeys(ctx, name, keys...)
}

func (m *Manager) TypeText(ctx context.Context, name, text string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.typeText(ctx, name, text)
	}
	return m.typeText(ctx, name, text)
}

func (m *Manager) ClearLine(ctx context.Context, name string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.clearLine(ctx, name)
	}
	return m.clearLine(ctx, name)
}

func (m *Manager) PasteText(ctx context.Context, name, text string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.pasteText(ctx, name, text)
	}
	return m.pasteText(ctx, name, text)
}

func (m *Manager) PaneCommand(ctx context.Context, name string) (string, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.paneCommand(ctx, name)
	}
	return m.paneCommand(ctx, name)
}

func (m *Manager) SessionActivity(ctx context.Context, name string) (time.Time, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.sessionActivity(ctx, name)
	}
	return m.sessionActivity(ctx, name)
}

func (m *Manager) PanePID(ctx context.Context, name string) (int, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.panePID(ctx, name)
	}
	return m.panePID(ctx, name)
}

func (m *Manager) PaneSessionID(ctx context.Context, name string) (string, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.paneSessionID(ctx, name)
	}
	return m.paneSessionID(ctx, name)
}

func (m *Manager) PaneCwd(ctx context.Context, name string) (string, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.paneCwd(ctx, name)
	}
	return m.paneCwd(ctx, name)
}

func (m *Manager) CaptureTail(ctx context.Context, name string, n int) (string, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.captureTail(ctx, name, n)
	}
	return m.captureTail(ctx, name, n)
}

func (m *Manager) SetEnv(ctx context.Context, name, key, value string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.setEnv(ctx, name, key, value)
	}
	return m.setEnv(ctx, name, key, value)
}

func (m *Manager) SetOption(ctx context.Context, name, key, value string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.setOption(ctx, name, key, value)
	}
	return m.setOption(ctx, name, key, value)
}

func (m *Manager) RespawnPaneEnv(ctx context.Context, name, cwd string, extraEnv []string, command string, args ...string) error {
	if m.legacyFor(ctx, name) {
		return m.legacy.respawnPaneEnv(ctx, name, cwd, extraEnv, command, args...)
	}
	return m.respawnPaneEnv(ctx, name, cwd, extraEnv, command, args...)
}

func (m *Manager) SetScopedOption(ctx context.Context, scope, session, key, value string) error {
	if m.legacyFor(ctx, session) {
		return m.legacy.setScopedOption(ctx, scope, session, key, value)
	}
	return m.setScopedOption(ctx, scope, session, key, value)
}

func (m *Manager) UnsetScopedOption(ctx context.Context, scope, session, key string) error {
	if m.legacyFor(ctx, session) {
		return m.legacy.unsetScopedOption(ctx, scope, session, key)
	}
	return m.unsetScopedOption(ctx, scope, session, key)
}

func (m *Manager) SetArrayOption(ctx context.Context, scope, session, key string, values []string) error {
	if m.legacyFor(ctx, session) {
		return m.legacy.setArrayOption(ctx, scope, session, key, values)
	}
	return m.setArrayOption(ctx, scope, session, key, values)
}

func (m *Manager) ApplyValue(ctx context.Context, session string, sv ScopedValue) error {
	if m.legacyFor(ctx, session) {
		return m.legacy.applyValue(ctx, session, sv)
	}
	return m.applyValue(ctx, session, sv)
}

func (m *Manager) InputSnapshot(ctx context.Context, name string) (InputSnapshot, error) {
	if m.legacyFor(ctx, name) {
		return m.legacy.inputSnapshot(ctx, name)
	}
	return m.inputSnapshot(ctx, name)
}

func (m *Manager) PasteOnly(ctx context.Context, paneID, text string) error {
	err := m.pasteOnly(ctx, paneID, text)
	return m.legacyPane(func() error { return m.legacy.pasteOnly(ctx, paneID, text) }, err)
}

func (m *Manager) SubmitPane(ctx context.Context, paneID string) error {
	err := m.submitPane(ctx, paneID)
	return m.legacyPane(func() error { return m.legacy.submitPane(ctx, paneID) }, err)
}

// The server-wide reads merge both servers during the drain: the tmux app,
// the sidebar and forensics see one fleet while sessions are moving.

func (m *Manager) ServerSessions(ctx context.Context) ([]ServerSession, error) {
	cur, err := m.serverSessions(ctx)
	if err != nil || m.legacy == nil {
		return cur, err
	}
	old, lerr := m.legacy.serverSessions(ctx)
	if lerr != nil {
		return cur, nil // drain visibility is not an error
	}
	return append(cur, old...), nil
}

func (m *Manager) ListSessions(ctx context.Context) ([]Session, error) {
	cur, err := m.listSessions(ctx)
	if err != nil || m.legacy == nil {
		return cur, err
	}
	old, lerr := m.legacy.listSessions(ctx)
	if lerr != nil {
		return cur, nil
	}
	return append(cur, old...), nil
}

func (m *Manager) ListOwned(ctx context.Context) ([]OwnedSession, error) {
	cur, err := m.listOwned(ctx)
	if err != nil || m.legacy == nil {
		return cur, err
	}
	old, lerr := m.legacy.listOwned(ctx)
	if lerr != nil {
		return cur, nil
	}
	return append(cur, old...), nil
}

// ServerInfo reports the primary server, but a drain that has not opened the
// primary yet is not "no server": a legacy session answers as running so the
// tmux app draws the fleet instead of an empty machine.
func (m *Manager) ServerInfo(ctx context.Context) ServerInfo {
	info := m.serverInfo(ctx)
	if !info.Running && m.legacy != nil {
		if old := m.legacy.serverInfo(ctx); old.Running {
			info.Running = true
			if info.SocketPath == "" {
				info.SocketPath = old.SocketPath
			}
			// The legacy server is the one answering, so its version and
			// keyboard mode are the ones in use — not the primary's, which
			// has no server behind it yet.
			info.Version = old.Version
			info.ExtendedKeysFormat = old.ExtendedKeysFormat
		}
	}
	return info
}
