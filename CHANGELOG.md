# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

## [Unreleased]

### Changed (owner feedback: agents must be IDE tabs; dock must not auto-open)

- **Agents now open as tabs in the editor area** (IDE convention): clicking
  a workspace in the sidebar opens/selects a tab; tabs carry a status dot
  and a closer; closing a tab detaches the terminal attach but the agent
  keeps running. Multiple agents coexist as tabs.
- **Terminal dock never auto-opens** (bug + UX): previously selecting an
  interactive agent always re-opened the dock, defeating the closer — the
  #1 "unusable" report. Dock now opens ONLY via the explicit Terminal
  action and stays closed across sidebar clicks and reloads.
- Sidebar active state syncs with tab selection.
- `agent-browser` skill gains an **exploratory QA recipe** (clickability
  sweep, open/close persistence cycles, reload-state checks, IDE
  conventions) — the lesson from this round's QA gap.

### Fixed (exploratory QA round — agent-browser skill dogfooding)

- **Ghost overlay blocked the chat after Run**: `.run-cta`'s explicit
  `display:grid` defeated the `hidden` attribute, leaving an invisible
  full-surface overlay eating every click (Send/Terminal/Stop dead).
  Fixed with `.run-cta[hidden]{display:none}`.
- **Dead zone after reload**: with all agents stopped, no workspace was
  selected and the main area rendered empty. Boot now selects the first
  workspace when none is running.
- **Interactive mode orphaned**: the anatomy redesign dropped the only UI
  path to the terminal (tmux TUI) mode — `/open` had no caller. Restored:
  the stopped-agent CTA now offers **Run agent** and **Open terminal**
  (door, not cage).
- **Raw backend errors in the form**: `store: path…: stat…` leaked Go
  internals to users; `humanizeError` maps common cases to plain language.
- **tmux warning jargon** in the sidebar humanized (raw detail stays in
  Settings → System); managed panel now greets with a connected line.

### Added

- QA evidence: `docs/screenshots/qa-m2-final-dark.png`, `qa-interactive-tui.png`.

### Added

- **`agent-browser` skill** (ported from the agentdeck-proven skill, adapted):
  stateful Chromium automation via bash — agents can now open PiCode,
  snapshot the accessibility tree, click through flows, eval JS and take
  live screenshots against the running server. Includes PiCode specifics:
  port discovery via `server.json`, HTTPS ignore-flag, the go:embed
  rebuild rule, the observed `close`+`open` session-reset gotcha, and the
  UI map. `visual-review` skill now points to it for interactive checks.

### Fixed

- **Theme buttons inside the user-menu popover were dead**: the popover's
  `stopPropagation` blocked clicks from reaching the document-level theme
  handler. Reworked outside-click close logic; picking a theme now applies,
  persists and closes the popover (Vercel-style). `cmd/uicheck` extended to
  three functional assertions: popover opens, theme click applies+persists
  (dataset + localStorage + closed), Settings link navigates to `#/settings`.

### Fixed

- **User-menu popover not opening**: a `func`-instead-of-`function` typo
  in app.js (SyntaxError at boot) shipped inside the last binary; JS
  syntax-check was not run after the final edit (process slip, noted in
  handoff). Fixed, plus two systemic guards: `Cache-Control: no-cache` on
  the app shell (vendor caches 1h) so browsers never serve stale UI after
  binary upgrades, and a new `go run ./cmd/uicheck <url>` debug tool that
  loads the app headlessly, captures console exceptions, and clicks the
  user-menu trigger to verify the popover opens.

### Added

- **HTTPS by default with mkcert trust (ADR-0007)**: serves TLS on
  `0.0.0.0` with a zero-config self-signed bootstrap; `make cert`
  (`scripts/setup-cert.sh`) upgrades to a mkcert-issued cert (installs
  mkcert if missing, SANs = localhost + LAN + tailscale, CA exported to
  the Windows trust store on WSL, optional iOS import `--ios`); weekly
  renewal via systemd timer (`scripts/install-systemd.sh`).
- **Runtime port configuration**: Settings UI (`#/settings` → Server) now
  changes the port — validated, probe-bound (busy port → clear error), and
  applied via graceful rebind (new listener up before the old one drops;
  automatic revert on failure). Precedence: UI/DB > `PICODE_PORT` env >
  default range `8445-8455` (first free port wins). Discovery file
  `~/.picode/server.json` (url/port/pid) for scripts and tooling.

### Changed

- **UI redesigned to agent-IDE anatomy** (owner feedback: "looks nothing
  like Cursor/t3code/paseo"): conversation is now the centered hero surface
  (~760px column, flat blocks with actor labels — Cursor composer style),
  tool calls render as pills inside the conversation (28px, expandable),
  composer is a rounded elevated box (textarea + Prompt/Steer/Follow-up
  chip + send), terminal became a bottom **dock** with compact tabs —
  fixing the stacked-surfaces bug (panel residue above tabs).
- New visual spec documented (`docs/design/benchmark-visual-anatomy.md`)
  with provenance note (trained knowledge, not live capture);
  `docs/design/references/` created for owner screenshots — vision sessions
  must verify/refine against them. Tokens updated: `#0e0e11` base,
  `#131317` panel, `#7c8cf8` accent.

### Added

- **M2 core — managed agents (ADR-0006)**: agents run in exactly one mode
  (interactive tmux TUI **or** managed rpc). `internal/rpc`: JSONL client for
  `pi --mode rpc` (strict `\n` framing, response correlation, event fan-out)
  plus the managed runtime — **task delivery engine** claims queued tasks,
  maps kind→command (`prompt`/`steer`/`follow_up`), gates on `agent_settled`,
  finishes delivered/failed with audit events. Verified against real pi.
- **Agent panel UI**: live streaming view (text/thinking deltas), compact
  tool-call rows (≤32px collapsed, click to expand args/result — Cursor bar),
  task composer with kind selector; state indicators (streaming/idle).
- **Mode-switch API**: `POST /api/agents/{id}/managed/start|stop` (stops the
  other mode first); `GET /ws/agent?agent=` streams events + accepts
  `enqueue` commands; workspace views now expose `agent.mode`.

### Added

- **Database layer (ADR-0005)**: SQLite store via pure-Go `modernc.org/sqlite`
  at `~/.picode/picode.db` — orchestration overlay only (sessions, creds,
  MCP and skills stay in pi's own files; never duplicated). Schema v1:
  `workspaces`, `agents` (default agent per workspace), `tasks` (prompt/
  steer/follow_up queue, delivery state machine, claim/finish for the M2
  engine), `messages` (reserved M4), `events` (audit), `settings`.
  Embedded sequential migrations; one-time import of the M1 JSON registry
  (retired to `workspaces.json.migrated`).
- **Task queue API** (M2 prep): `POST/GET /api/agents/{id}/tasks`; tasks
  persist as `queued` until the RPC delivery engine lands.
- `/api/workspaces` responses now embed the workspace's default agent
  (`agent.id`, `agent.running`, `lastStatus` cache).

### Changed

- Workspace persistence moved from flat JSON (`internal/workspace`, removed)
  to the SQLite store; API surface unchanged except the embedded `agent`
  object; tmux session names now derive from the agent id.

### Added

- **User menu** (Vercel-inspired) in the sidebar footer: local identity
  (hostname from `/api/system`), inline theme switcher
  (Light/System/Dark), Settings and Documentation links, version.
- **Settings view** (`#/settings`, hash routing): Appearance (theme cards,
  applied instantly and persisted), System status (tmux/pi/extended-keys),
  About (version, repo, license).
- **Theme system**: light/dark/system with `localStorage` persistence,
  no-flash bootstrap in `<head>`, OS color-scheme tracking, `?theme=`
  override for testing/links. Applies to app chrome only — agent terminals
  keep their own colors.
- **Live statusbar**: connection state (connected/detached) and terminal
  dimensions (cols×rows) replacing the former static hint.

### Changed

- **UI copy de-documentarized** (owner directive): teaching-steps empty
  state replaced with a minimal one (one line + one action); sidebar
  footer system info moved into Settings; statusbar hint removed —
  documentation lives in README/docs, UI carries state and actions only
  (`docs/benchmarks.md` records the rule).
- `/api/system` now reports the machine hostname.

### Added

- **M1 — Terminal grid** (complete vertical slice):
  - `picode screenshot --url --out [--full --width --height --wait-ms]`
    subcommand (chromedp) powering the visual-review loop end-to-end.
  - tmux session manager (`internal/tmux`): create/kill/list with a
    `picode-` namespace, exact-name matching (`=` prefix), sanitized ids,
    availability/version/extended-keys detection.
  - Terminal bridge (`internal/term`): WebSocket ↔ PTY (`tmux attach`);
    binary frames carry terminal bytes, text frames carry `resize`
    control messages; detaching never kills the agent.
  - Workspace registry (`internal/workspace`): file-backed JSON at
    `~/.picode/workspaces.json`, idempotent add, validation.
  - HTTP API: workspace CRUD + open/close lifecycle + `running` status,
    `GET /api/system` (pi/tmux detection with actionable warnings).
  - UI: dark-first terminal grid (vanilla ES, vendored xterm.js 5.5.0 +
    fit addon — ADR-0004) — sidebar workspaces with status dots, tabs,
    teaching empty state, auto-attach to running agents on load.
- **ADR-0004**: frontend framework decision deferred with explicit
  adoption thresholds.
- **First visual evidence**: `docs/screenshots/m1-termgrid-first-look.png`
  (capture pipeline proven; visual verdict UNVERIFIED — see handoff).

- **Visual validation loop**: new `visual-review` skill (capture → read
  pixels → judge against Cursor bar/benchmarks → verdict + evidence),
  `docs/screenshots/` for committed evidence, gitignored `var/` working
  dir, and an honesty rule in `AGENTS.md`: no visual verdict without an
  actual screenshot.

- **MCP strategy**: documented adoption of `pi-mcp-adapter` (community Pi
  extension — proxy tool, lazy servers, standard configs, Cursor/Claude/Codex
  config import) with PiCode's value-add being a visual MCP Server Manager
  per workspace/agent (M3–M4). No native MCP in Pi by design; PiCode does
  not fork the ecosystem, it orchestrates it (`docs/architecture.md`).

- **Cursor benchmark**: product patterns (agent activity feed, checkpoints,
  diff review, per-agent model picker, background-agent cards, @-mentions,
  rules management, command palette) mapped to milestones, plus an
  aesthetic/density bar with design tokens — enforced via the `uiux-review`
  skill (`docs/benchmark-cursor.md`).

### Fixed

- **Race in terminal bridge shutdown** (caught by CI's `-race` on ubuntu):
  redesign from `sync.Once`-based teardown to single-owner pty access with
  cooperative unblocking; handler returns only after both pumps finish.

### Changed

- Dependencies: chromedp, gorilla/websocket, creack/pty added (each with
  justification; go.mod toolchain follows current Go).
- **Official language set to English**: changelog, docs references and skills
  translated; language policy added to `AGENTS.md` and `CONTRIBUTING.md`.

## [0.1.0] - 2026-08-23

### Added

- **Project bootstrap**: public repository, MIT license, CI (GitHub Actions
  with gofmt/vet/test/build across linux/macos/windows), Makefile.
- **Living documentation system**: `docs/` with architecture, philosophy,
  engineering + UI/UX benchmarks, handoff (`docs/handoff.md`) and ADRs
  (`docs/decisions/`) — with an explicit contract that documentation evolves
  with the code (see `AGENTS.md`).
- **Pi agent harness**: root `AGENTS.md` (operating contract), quality skills
  in `.pi/skills/` (`quality-gate`, `uiux-review`, `handoff-update`) and
  project settings in `.pi/`.
- **Go server skeleton**: `picode` binary with UI embedded via `go:embed`,
  `/api/health` and `/api/version` endpoints, dark-first placeholder page
  with a live health check.
- **Initial decision records (ADRs)**: browser app served by a single Go
  binary (0001), dual-channel tmux+RPC agent control (0002), dependence on
  user-installed `pi` (0003).

[Unreleased]: https://github.com/cfpperche/picode/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
