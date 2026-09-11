# ADR-0120: Desktop v2 shell — Tauri 2 + WebView2

- **Status**: accepted
- **Date**: 2026-09-11
- **Boundary**: process + security — a new Windows runtime (Rust/Tauri, WebView2) becomes the desktop shell and the browser actuator for agents; the Go daemon inside WSL stays the only backend, and CDP replaces the Chrome-extension bridge (supersedes ADR-0043's mechanism, keeps its goal).

## Context

PiCode Desktop v1 (`cmd/picode-desktop`, Go + fyne.io/systray) is a tray-only
process: no window, menu and MessageBox as its whole UI. It grew real
responsibilities — disk facts, the compact action, keepalive — and its UI shape
stopped fitting them. The web UI, meanwhile, already runs in the user's
personal browser, mixed with personal tabs and profiles.

The owner's direction (2026-09-11): a work application with its own window and
its own browsing context, separate from the personal browser, keeping the
Windows + WSL architecture (daemon in WSL is the source of truth). Rust was
chosen for the shell over Electron/CEF to keep the binary small and reuse the
platform WebView; WebView2 is Chromium (Blink + V8), already present on
Windows 10/11, Evergreen-updated by Microsoft, and honours the mkcert CA the
provision flow installs.

Browser access for agents moves to **CDP directly**. The Chrome-extension +
native-messaging bridge (ADR-0043) existed because CDP is blocked on a user's
default Chrome profile; in a browser the product launches and owns
(own user-data-dir, `--remote-debugging-pipe`, no open port) that limitation
does not exist. The extension's trust model — scoped, visible, revocable — is
re-implemented as shell policy (domains × actions per agent), not as a
browser extension.

## Decision

- The desktop shell is a **Tauri 2** application (`desktop-shell/`, crate
  `picode-shell`): a window rendering the PiCode UI served by the daemon in
  WSL, plus a tray (status, open, quit) and single-instance behaviour.
- The daemon in WSL remains the **only backend**. The shell is a client and a
  supervisor: it launches and watches processes, reads machine facts, and
  executes the Windows-side actions; it never stores state beyond preferences.
- Agents reach pages through **CDP** exposed by the shell to the daemon over
  the existing authenticated channel — in-process/pipe, never an open
  authenticated-everyone port — gated by an explicit permission surface
  (domains × actions per agent) in the UI.
- `ext/` + `internal/browserhost` are deprecated when the CDP path ships and
  removed once the managed work browser replaces personal-Chrome workflows.
- Cross-builds happen from WSL with `cargo-xwin` targeting
  `x86_64-pc-windows-msvc`; no Rust in the default CI gates (a dedicated
  `make desktop-shell` target, guarded by toolchain presence).

## Consequences

- **Easier**: one work window with app identity (taskbar, Alt-Tab,
  notifications later); personal browsing fully separated; the tray's actions
  get a real UI (plan → confirm → progress → measured result) instead of
  MessageBox; remote/mobile access untouched.
- **Harder**: a new toolchain and a new runtime to maintain; WebView2 is
  Evergreen (Microsoft controls the engine version — Fixed Version exists if
  determinism is ever required); Google login inside embedded views can be
  blocked and needs a spike; the keepalive/watchdog must migrate before the Go
  tray retires, or the two coexist with duplicated duty.
- **If we're wrong**: the shell is a client — deleting it leaves the daemon,
  the CLI and the browser path working exactly as v1 did.
