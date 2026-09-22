# ADR-0180: terminal-browser-handoff

- **Status**: accepted (owner, 2026-09-22 — "aprovado, pode executar", on the
  recommendation alongside the spec)
- **Date**: 2026-09-22
- **Boundary**: protocol (new terminal route + a new ephemeral feed event)
  and process (environment injected into every session PiCode creates).

## Context

When a coding CLI runs its login flow (`/login`, OAuth), the CLI resolves
"open in the browser" inside WSL: `$BROWSER` is unset, so the chain
`xdg-open` → `wslview`/desktop default finds the Chromium installed in the
distro and the login page opens in a WSL window nobody is looking at. The
person is looking at a PiCode client — the desktop app (which has an
integrated work browser, ADR-0128) or the web UI in a Windows browser.

The pieces to route the URL already exist: the ADR-0056 intercept puts a
daemon-owned bin dir first on every managed session's PATH (the tmux guard,
ADR-0138, is a proven fourth wrapper class there); the feed fans ephemeral
notices to every subscribed client; the desktop shell already reroutes
`window.open` of off-origin http(s) links into its integrated tab
(`web/browser/src/lib/externalLinks.js`); and `internal/osopen` knows the
WSL → Windows bridge for file managers. What was missing is the one hop
terminal → daemon → the client the human is looking at.

Constraints: OAuth URLs legitimately carry `&` and long query strings, so
the desktop shell's `cmd /C start` allowlist (`external_target`) cannot be
reused verbatim for the server-side open — the URL must reach PowerShell as
data, never as a command line. And a login URL must not replay on a feed
reconnect hours later.

## Decision

Managed terminals get a **browser hand-off**: a `picode-open` wrapper plus
`xdg-open` and `wslview` shadows in the intercept bin dir, and
`BROWSER=<dataDir>/bin/picode-open` in the session environment. The wrapper
forwards exactly its first `http(s)` argument to
`POST /api/terminals/{id}/open-url` (bearer token from `<dataDir>/token`,
same as the hook reporters); every other argument, missing curl, an absent
marker, or a failed POST falls through to the real opener it shadows.

The endpoint validates the URL (trimmed http(s), ≤ 2048 bytes, no control
characters or quote/redirect metacharacters — `&` stays legal), then routes:

| Condition | Action |
|---|---|
| Feed has ≥ 1 subscriber | `Ephemeral("terminal.open_url", {termId, url})` → `{"delivered":"client"}` |
| No subscriber | `osopen.OpenURL(url)` on the daemon host → `{"delivered":"host"}` |
| Unknown terminal | 404 |
| Refused URL / bad JSON | 400, nothing opened |

The event is ephemeral (id 0, never replayed). A **visible** client
(`document.visibilityState`) answers it through the same path as a
Ctrl+clicked terminal link (`openTermLink`), so the existing link-destination
preference governs: the desktop app opens its integrated tab, a plain
browser opens a tab in itself. Hidden clients stay out; a 1.5 s per
terminal+URL dedupe absorbs CLI retries without queueing stale pages.

The host-side `osopen.OpenURL` opens the WSL machine's **Windows** default
browser: PowerShell `Start-Process` with the URL passed as the
`PICODE_OPEN_URL` environment variable, so no shell ever parses it
(`rundll32 url.dll,FileProtocolHandler` on native Windows, `open` on macOS,
`xdg-open` argv on plain Linux). The wiring row `open-url` defaults on like
the tmux guard; `enabled.json "open-url": false` (wiring enable/disable
verbs) removes the wrappers and the env entry. Boot refreshes the wrapper
files, so upgrades can never leave stale ones.

## Consequences

Easier: `/login` in any CLI opens where the human is looking — integrated
tab in the desktop app, a tab in the browser they run PiCode in, or the
Windows default browser when the app is closed. `osopen.OpenURL` is a
reusable primitive for later "open this link" features.

Harder / accepted cost: inside a managed terminal, *any* command's
`xdg-open`/`wslview`/`BROWSER` open now escapes to the host surface instead
of a WSL browser — the desired behavior, but a change for scripts that
expected a local window. An off-scope consequence is honest: two clients
open at once is possible (both visible, both subscribed); the dedupe is
per-client only. Path shadowing is a guardrail, not a boundary — a CLI that
hardcodes `/usr/bin/xdg-open` bypasses the hand-off and gets today's
behavior.

Who breaks if we're wrong: a login whose CLI passes the URL through a
mechanism we do not shadow keeps opening in WSL chromium (no regression);
a URL we refuse (non-http scheme) is refused everywhere consistently, and
the wrapper's fallthrough covers legitimate `file:`/path opens.

## Alternatives considered

- **Set only `BROWSER`, no shadows**: half the coverage — several CLIs call
  `xdg-open` or `wslview` directly and never read `BROWSER`.
- **Server always opens the Windows browser** (no feed hop): simplest, but
  the desktop app's integrated tab — the owner's explicit ask — is
  unreachable from the daemon, and the browser choice ignores the client
  the human is using.
- **wslview-only wiring** (install wslu and stop): depends on an external
  package's update cadence, cannot reach the desktop app, and cannot
  distinguish "nobody watching" from "WSL chromium".
- **Drive the CLI's own browser preference files**: violates ADR-0056's
  "intercept, not user-file wiring" decision and would need per-CLI
  adapters that rot.
