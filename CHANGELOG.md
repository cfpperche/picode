# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

### Changed

- Folder picker on WSL lists Windows drives (`C:`, `D:`) and accepts `C:\\…` paths. Place chips use home/drive icons; current path is a labeled card.
- Removing an agent or workspace now offers to delete that folder's pi sessions (and a free-agent work folder) when nobody else uses it. Project folders stay. All agents in a workspace are stopped first.

### Added

- Local backup in Preferences: folder, interval, retention, explicit **Schedule**, Backup now, Restore. Snapshots are inspectable directories (`VACUUM INTO` + hardlinks). Choosing a folder does not start the schedule.
- Preferences uses section tabs (Appearance / Notifications / Server / Backup).
- Backup now shows a step overlay (same pattern as package install/remove).
- Snapshot rows: Restore, Reveal in Explorer, Remove. Reveal on WSL uses the Windows Explorer binary (not PATH).
- Backup job card animates each step (spinner + check) instead of jumping to done.
- Package install/remove uses the same stepped motion.
- Restore uses the same stepped overlay and asks to reload when done.
- Pin sketches V3: Excalidraw (blank or annotate an image). Scene + preview on disk.
- Pin file cards: type badge (PDF/ZIP/…) + always-on remove. No artifact preview.
- Pin body is a TipTap editor (saves markdown). Toolbar: bold, italic, heading, lists, code, quote.
- Pin attachments V2: paste/drop/import images and files. Bytes on disk, not in the list.
- Pin form is a route (`#/pins/new`, `#/pins/:id`). Sidebar Pins is the list only.
- Console easter egg. Free-agent status no longer 404s (was using agent id as workspace).
- Browser QA: `npm:pi-agent-browser-native` (project + machine). Skill shrunk to PiCode-only map.
- Pins: flat list + form (title, tags, markdown body). `+` on the Pins title bar.
- Sidebar tabs: Agents / Pins (title left, tabs right). QR moved to user menu → Open on phone.
- Conversation blockquotes and ```diff``` fences (same hunks as tool diffs).
- Conversation images (`![alt](https://…)`) and Mermaid fences.
- Conversation math (KaTeX: `$…$` inline, `$$…$$` display).
- Conversation markdown tables (GFM), plus strikethrough and task lists.
- Conversation source blocks: language label, copy, highlight.js tokens on PiCode colors.
- **Run** on bash / python / javascript / go blocks — executes in the agent's folder.
- Sidebar shows the pi braille spinner on an agent card while that agent is working.
- Chat expands `web_search` / `url_context` into source cards (title, url).

### Fixed

- Closed agent tabs stay closed across reload (no longer reopen the first running agent).
- Model errors (quota, 400) show in chat and as a toast; empty assistant replies are no longer silent.
- Chat applies `message_end` / `turn_end` text (Codex often does not stream `text_delta`). Sending no longer reloads the transcript (that wiped the new message).
- One assistant reply per turn — `agent_end` no longer replays every message into the chat.
- Live `web_search` opens source cards in the tool pill (same as after reload); tool dumps stay out of the assistant text.
- Packages **This agent**: only that agent, every session (`pi -e` on start). Optional isolate skips machine and folder extras.
- `/llama` dialog: URL, load/unload/download. Setup is docs, not an in-app installer.
- `/share` creates a secret GitHub gist (`gh`). Not the phone QR.
- Composer `/` lists skills and prompt templates (insert; pi expands on send).
- `/export` JSONL download, `/import` file picker, `/hotkeys`, `/changelog` (installed pi).
- Multiple logins per provider (**Add account** / **Use**). pi still has one active slot.
  Re-login on the same OAuth account updates tokens (no duplicate). Click the name to rename.
- Claude and Codex **account** login via the same loopback ports as pi TUI.
- kimi-coding is account-or-api-key (TUI parity).
- `#/providers` Add-provider wizard (full loginable set, account vs API key).
  `/login` opens Add.
- Folder field: type a path or Browse (list + create directory on this machine).

### Changed

- Providers page: no docs copy in chrome (auth.json / TUI notes live in [public docs](https://cfpperche.github.io/picode/guide/providers)).
- Sidebar tree: one indent step (12px) per level; chevron / icon / label share a grid so names and paths line up.
- Controls share `--ctl-h` 36px (shadcn h-9). Input + button in a row match.
- Create forms validate with Zod (`noValidate` — no native browser bubbles).
- New agent / workspace forms are a dialog (desktop) or Vaul drawer (mobile),
  not inline in the sidebar.
- Palette, session picker, kind chip, slash menu, and user menu use cmdk / Radix.
  Icons are lucide-react.
- Toasts use Sonner. Position and options live in Preferences.
- `/quit` stops the agent and closes its tab.
- `/session` dialog: name, file, git, model, usage, plus copy/rename/new/compact/tree.
- Sidebar agent row: provider icon, name, and model (`Grok - grok-4.6`).
- Slash hints: docs icon, italic + dotted underline, slightly more context.
- Collapsed sidebar groups show stacked provider favicons (max 5, then +N)
  instead of a count.
- License: MIT → PolyForm Noncommercial 1.0.0 for personal/noncommercial
  use; commercial/enterprise needs a signed license
  ([LICENSING.md](LICENSING.md)). Prior MIT tags stay MIT.

### Added

- Composer `/copy` `/quit` `/reload` `/logout` `/session` `/trust` open PiCode UI
  (clipboard, stop agent, restart, providers sign-out, session dialog, trust.json).
- Public docs are VitePress (Markdown in `www/` → GitHub Pages). Slash hints
  open `/commands#{id}` in a new tab. No in-app docs viewer.
- Session tree dots sit mid-card; the spine runs through the last card.
- Session tree is a chain of prompt cards; replies/tools sit on the card.
- Composer `/tree` `/fork` `/clone`: session tree dialog. Fork is a new
  session from a user prompt; clone duplicates this branch (RPC in chat).
- `/scoped-models` opens Settings; `enabledModels` patterns + default
  tools (native checkboxes) on global/workspace cards.
- Settings layers resolve by depth (workspace beats global, like skills).
  No "Pi default" empty option.
- Pi Settings S3: agent card writes provider/model/thinking and Full/Read-only
  (existing PATCH, all sessions of that pi).
- Pi Settings S2: workspace `.pi/settings.json` when the folder is in
  `trust.json`. Untrusted → 409, run `/trust` in the terminal.
- Pi Settings S1: read/write global `settings.json` (auto-compact, steering,
  follow-up, defaults). Radix switch + native selects.

### Changed

- Packages (and other pane cards) have inner padding.
- Rail tick preview uses Radix Tooltip (no longer misplaced by the rail transform).
- Settings and Packages sit in the same gray panel card as System.
- Conversation rail is compact and centered; it grows with the thread up to 360px.
- Sidebar type is JetBrains Mono (same face Tachyon ships as Tachyon Mono).
- Sidebar section + aligns right. Empty groups show "— empty" collapsed
  and a one-line placeholder expanded (never a 0 badge).
- `#/settings` is pi (composer `/settings`). PiCode theme/port moved to
  `#/preferences` (ADR-0012). Settings page is a scoped shell (S0).
- One chrome gray (`--bg-panel`); `--bg-elevated` aliases it. Hover/focus
  still use `--bg-hover`. Canvas stays `--bg-base`.
- Compact 8px overlay scrollbar on every overflow (no Windows arrows).
  VS Code / Cursor / Linear pattern.

### Added

- Plan: `#/settings` becomes pi GUI; PiCode chrome → `#/preferences`
  (ADR-0012 proposed, `docs/design/pi-settings.md`).
- Slash parity matrix: TUI `/` vs PiCode composer
  (`docs/design/slash-parity.md`). 7 of 24 have UI.
- Workspace groups collapse (agent count), hover-only action icons, and a
  real git branch/worktree line (or "local").
- Sidebar splits **Agents** (`~/.picode/work/<name>/`, optional folder) and **Workspaces** with many
  agents per folder (own model/config). ADR-0011.
- Package install scope: **This machine** or **This workspace** (`pi install -l`).
  Source field accepts npm, git, and path.
- Package install/remove freezes the page behind a blur overlay and lists
  the real `pi install` / `pi remove` steps (no more every-button Working…).
- Packages gallery shows **skeleton cards** while npm search is in flight.
  Optimistic UI is now a project bar (`docs/philosophy.md` §7).
- Composer **dictation** (mic, Ctrl+D) and **voice mode** (waveform,
  Ctrl+Shift+O). Voice replaces the composer like Grok on x.ai; silence
  sends through pi. Dictation stays in the textarea until Send.

### Fixed

- Chat column and composer share one width (`--chat-col`).
- Composer bar hides scrolled chat text underneath (no leak at the bottom).
- Voice composer mic actually requests the microphone (and retries).
  Speaker toggles spoken replies (browser TTS) instead of a dead control.
- Dictation swaps the send cluster for a live waveform + cancel/confirm
  (ChatGPT pattern) so the recording state is obvious.

### Changed

- Agent cockpit lives in the **composer**: searchable provider, model,
  per-model thinking, and **Full / Read-only** tool-set. Read-only starts
  pi with `--tools read,grep,find,ls` and restarts a live agent.
- UI split into **desktop** and **mobile** shells (one Vite app). Phone
  gets a bottom-nav ADE; desktop is unchanged. `?desktop=1` / `?mobile=1`.
- Product benchmarks now include **t3code** and **paseo** alongside
  Cursor (`docs/benchmarks/`). Studies required before substantial UI.
- Sidebar is resizable (drag the right edge; 180–480px; remembered).
- **Settings is PiCode-only.** Providers and MCPs are their own routes
  (`#/providers`, `#/mcps`), listed in the user menu next to Settings and
  Documentation. Documented in `docs/architecture.md`.

### Added

- Chat body type is **16px / 1.7** (Claude/Grok reading size); chrome stays 13px.
- Date marks between turns (Today / Yesterday / weekday), ChatGPT-style.
- Conversation **section rail** (Grok-style): ticks per message, hover
  preview, jump. Native scrollbar hides while the rail is on.
- Agent work (thinking + tools) collapses to **Worked for Xm Ys**;
  expand for the step list. Duration from pi timestamps.

- **Packages** gallery is a **2-column card grid** (adapted from
  [pi.dev/packages](https://pi.dev/packages)). Preview frame matches the
  official catalog: real `pi.image` when pi.dev has one, graph-paper
  placeholder otherwise. No invented charts.
- App icon is the official **pi favicon**.
- Assistant replies render **markdown**. Sessions can be **renamed**
  (picker or `/name`).
- Confirms use a **Radix dialog** (compact, remove workspace), not `window.confirm`.
- Click the context bar (or `/compact`) to **compact** the session via pi RPC.
- **Copy** on assistant replies.
- Composer **floats** over the conversation (Claude/ChatGPT); the
  scrollbar runs the full pane height.
- **Send** from the composer works when the agent is stopped or in the
  TUI: the prompt is queued and the agent runs in managed mode (chat).
- Composer **status line**: cwd, git (dirty *), context bar +(auto), ↑↓
  tokens, cache hit, cost, session name.
  (same facts as the pi TUI footer; no invented provider quotas).
- Stopped state uses the **composer chrome** (Run / Open terminal), not a
  floating card over the chat.
- Session **replay** paints the JSONL conversation in the chat (view-only).
- Visual QA is a **gate**: overlay geometry audit + 5-question card.
  Clipped menus fail the build ritual (see `/skill:visual-review`).
- **Sessions** in the tab strip: list / resume / new from pi JSONL files.
  `/new` and `/resume` hit that UI, not the TUI.
- Host, network, deps and About live on **`#/system`**. Settings is only
  theme and port.
- In-app **toasts** replace browser `alert()` for errors (slash commands,
  start/stop, login).
- **Install app** button (Chrome `beforeinstallprompt`). iPhone still
  uses Share → Add to Home Screen — Apple has no install API.
- Mobile **PWA**: installable home-screen app (`standalone`, Apple meta,
  service worker). start_url forces the mobile shell.
- **Devices** (`#/devices`): connected browsers. Host vs other machines
  (iPhone/Android + IP). Online if pinged in the last 45s.
- Phone trust page is a 3-step wizard, iOS vs Android from User-Agent
  (profile download / CA install / enable trust / open) — not a text dump.
- Phone QR follows the selected path: Tailscale row → `http://<tailnet>:8470`,
  LAN row → Wi-Fi IP. Mixing them caused infinite load on the phone.
- Phone trust: QR opens an HTTP page (`:8470`) to install the mkcert CA
  (iOS profile / Android .cer) before the HTTPS app — fixes “Not Secure”.
- Cert SANs refresh automatically when the LAN/tailnet IP changes
  (`mkcert` rewrite + TLS reload on the next handshake — no rebuild).
  Only this machine's Tailscale IP is offered (not the Windows host node).
- **Open on phone**: QR button in the sidebar. Drawer diagnoses HTTPS,
  bind, reachable IP, cert SAN and mkcert CA (`GET /api/share`). QR only
  when ready; otherwise a how-to of the missing actions.
- Chrome no longer scrolls with Settings: sidebar stays pinned; only the
  main pane scrolls (`100vh` lock + `#settings-view` overflow).
- Agent provider/model/thinking lives on the **agent tab bar**, not in
  system Settings. Change saves immediately (applies on next start).
- **M3 lifecycle (ADR-0009)**: catalog from `pi --list-models` (no curated
  subset); workspace form + Settings → Agents set provider/model/thinking
  (empty = inherit). Settings → Providers shows signed-in state from
  `auth.json` keys; Sign in opens the TUI and types `/login`. Settings →
  MCP is status-only (not in the wizard). Startup exclusive lock on
  `picode.lock` (overlapping-restart incident).
- **Inline diffs** for `edit`/`write` tool pills (`+N −M`, expandable hunks).
  Uses `result.details.patch`/`diff` when pi sends them; otherwise the
  replacement args. View-only — no accept/reject (not an editor).
- **N files changed** summary at the end of a turn.
- **Command palette** (`Ctrl+K` / `⌘K`): open/run/stop/terminal per agent,
  Settings. Esc / click-outside closes.

### Changed

- **UI is React + Vite + Tailwind** (ADR-0008, supersedes ADR-0004).
  Same design tokens and component classes; vanilla `app.js` is gone.
  `make build` / CI run `npm ci && npm run build` into `internal/web/public`.
  Terminal dock is resizable (drag the top edge) and can maximize to fill
  the agent tab.

### Removed

- Yellow tmux-config tip from the sidebar. UI chrome carries state and
  actions only (owner rule); the check still lives in Settings → System.

### Fixed (the dock closer was a lie)

- **`.dock { display:flex }` beat the `hidden` attribute** — `hideDock()`
  set `hidden=true` but the panel stayed painted. Same class of bug as
  the Run-CTA ghost overlay; previous QA checked the attribute, not
  `getComputedStyle`. Fixed with a global `[hidden]{display:none !important}`.
- **Terminal dock no longer has its own tabs.** The panel belongs to the
  active agent tab (title = `Terminal · <agent>`). Closing `×` hides it
  (`display:none` verified) and it stays hidden across sidebar clicks.
  Preference is per agent tab.

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
