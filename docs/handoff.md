# Handoff — living project state

> **This file is the heartbeat of PiCode.** Any session (human or agent)
> that changes state MUST update this file before ending.
> Ritual defined in `/skill:handoff-update`; contract in [AGENTS.md](/AGENTS.md).
>
> Format rules: newest entries at the top of "Recent activity"; keep it
> short — this is a handoff, not a diary. Archive old blocks to
> `docs/handoff-archive.md` when the file exceeds ~150 lines.

## Current state (read this first)

**Phase: M2 core shipped (managed agents + panel + delivery engine). Remaining M2: diff view, command palette.**

What exists right now:
- Repo public at `cfpperche/picode`, MIT, CI green (linux/macos/windows).
- **M1 shipped**: screenshot tooling, tmux manager, WS↔PTY bridge,
  dark-first UI (terminal grid, user menu, settings, themes).
- **Store shipped** (ADR-0005): SQLite orchestration overlay.
- **M2 core shipped** (ADR-0006): `internal/rpc` (JSONL client + managed
  runtime + task delivery engine: claim → send → settle-gate → finish),
  mode-switch API, `/ws/agent` event stream + enqueue, agent panel UI
  (stream, tool rows ≤32px, composer with prompt/steer/follow_up).
  **Verified against real pi 0.84.2**: prompt task delivered, response
  streamed, queued follow-up from a previous session drained on start.
- **HTTPS + runtime port shipped** (ADR-0007): TLS default (mkcert via
  `make cert` / self-signed bootstrap), port editable in Settings UI with
  graceful rebind, `~/.picode/server.json` discovery, systemd units.
  Verified live: mkcert cert issued (SANs incl. 2 tailscale IPs), CA
  already trusted on Windows (shared mkcert CAROOT with agentdeck),
  port rebind 8445→8447→8445 same-pid, busy-port 409.
- Docs: philosophy, architecture, benchmarks (+Cursor bar), ADRs 0001–0006.
- Harness: AGENTS.md + 4 skills. 30 tests across 6 packages.

## Visual review status

- **M1 terminal grid: human visual validation PASS (owner, 2026-08-23)** —
  "acceptable for a first version".
- Post-feedback pass (user menu, settings, de-documentarized copy):
  `visual-review: UNVERIFIED` by agent (session model lacks image input);
  programmatic pixel checks PASS — dark tokens `#0d0f12/#15181d` and light
  `#ffffff/#f8f9fb` render correctly in evidence below.
- Evidence: `docs/screenshots/m1-ui-dark.png` (main view, dark),
  `docs/screenshots/m1-settings-light.png` (settings, light).
→ Next visual verdicts should come from a vision-capable model or human.

## In flight

- Nothing. Tree clean, all gates pass. (Server/tmux test sessions cleaned up.)

## Next up (finish M2, then M3 — Lifecycle)

1. **Diff view**: surface agent edits from `tool_execution_end` (edit/write
   tool calls carry oldText/newText + path) — inline diff rows in the panel.
2. **Command palette** (`Ctrl+K`): switch workspace/agent, start/stop, send task.
3. **M3 — Lifecycle**: agent creation wizard (model/provider/thinking per
   agent — columns already exist), provider auth flows (drive `/login` in
   the terminal), config profiles.

## Known debts / open questions

- **DATA INCIDENT (open)**: the real `~/.picode/picode.db` lost all rows
  created before 2026-08-24 02:25 (workspace + tasks + events) during the
  night's ~5 overlapping rebuild/restart cycles (two servers raced the
  same port at one point). Verified since: add → restart preserves data;
  store tests green. Root cause NOT established — candidates: two picode
  processes on one SQLite file across builds. **Follow-up: add an
  exclusive-lock/ownership guard on startup (e.g. server.json pid check +
  stale detection) and re-test multi-process behavior before M3.**
- **Visual verdicts require a vision-capable model** (see M1 note); QA
  evidence pixel-checked programmatically only.

- **Visual verdicts require a vision-capable model** (see M1 note); M2
  panel evidence: `docs/screenshots/m2-agentpanel-first-look.png` (pixel
  checks pass; agent verdict UNVERIFIED).
- Panel shows only events observed while connected — no session replay yet;
  replay via session JSONL reader is an M3 candidate.
- tmux-gated tests skip on windows/macos runners (tmux absent); ubuntu CI
  covers them (accepted).
- Token auth: ADR-0007 accepts the personal-network trust boundary (same
  as agentdeck); token auth becomes mandatory only if exposed beyond the
  tailnet (recorded debt).
- Vendored xterm.js 5.5.0 + fit addon need a manual upgrade story
  (note in ADR-0004); track upstream releases occasionally.
- Branch protection + CODEOWNERS on GitHub — needs owner action (manual).

## Recent activity

- **2026-08-24** — **Exploratory QA (agent-browser dogfooding)**: full
  click-through of the app — form validation, Run + real-pi prompt →
  stream + tool pills (verified real file listing), follow-up chips,
  stop/run cycles, interactive TUI restored (green pi pixels in dock),
  settings navigation, **real port rebind via UI (8445→8446→8445, page
  followed automatically)**, theme sweep, zero console errors. Fixed 5
  findings (ghost overlay, dead zone, orphaned interactive mode, raw
  errors, jargon warning). Data incident recorded above (open).

- **2026-08-24** — `agent-browser` skill added (agentdeck port, adapted):
  verified live against PiCode (open → snapshot → click user menu → apply
  Light theme → close). Harness now supports interactive UI automation via
  bash; visual-review references it. Next up unchanged: diff view (M2).

- **2026-08-24** — Fixed user-menu popover not opening (SyntaxError from a
  `func`→`function` typo that shipped because the JS syntax gate was
  skipped after the last edit). Systemic guards added: Cache-Control
  no-cache on the app shell + `cmd/uicheck` headless UI smoke tool
  (console capture + popover click). **Lesson recorded: the quality
  gate's JS-syntax step is mandatory after ANY app.js edit.**

- **2026-08-24** — ADR-0007 shipped: HTTPS default (mkcert/self-signed),
  port editable in Settings with graceful rebind + server.json discovery,
  scripts (setup-cert/install-systemd) ported from the agentdeck-proven
  pattern. Live-verified: CA trusted on Windows, rebind same-pid, 409 on
  busy port, screenshots over HTTPS.

- **2026-08-24** — UI redesign after owner feedback ("doesn't look like
  Cursor/t3code/paseo"): conversation-hero anatomy, tool pills, rounded
  composer, terminal dock; stacked-surfaces bug fixed by construction.
  Visual spec written with provenance note; `docs/design/references/`
  awaiting owner screenshots for vision-session verification. Evidence:
  `docs/screenshots/m2-chat-redesign.png` (pixel checks pass; agent visual
  verdict still UNVERIFIED — vision model needed).

- **2026-08-24** — M2 core shipped (ADR-0006): rpc client + managed runtime
  + delivery engine, mode-switch API, /ws/agent, agent panel UI. Verified
  against real pi (prompt delivered + streamed; stale queued follow-up
  drained on start). Remaining M2: diff view, palette.

- **2026-08-24** — ADR-0005 shipped: SQLite store (pure-Go driver), schema
  v1 (workspaces/agents/tasks/messages/events/settings), embedded
  migrations, legacy JSON registry imported+retired on live machine; task
  queue API live (queued until M2 engine); workspace views embed default
  agent; UI follows agent ids for tmux sessions.

- **2026-08-23** — Owner feedback applied: UI copy de-documentarized (rule
  recorded in benchmarks.md), Vercel-style user menu (identity + theme +
  links), settings route `#/settings` with theme cards, live statusbar,
  hostname in `/api/system`. M1 visually validated by owner (PASS).

- **2026-08-23** — CI `-race` caught a shutdown race in the term bridge
  (Close vs Setsize/Read); redesigned to single-owner pty access + cooperative
  unblocking. Verified with `go test -race -count=3` locally.
- **2026-08-23** — M1 complete: screenshot tooling, tmux manager, WS↔PTY
  bridge, workspace registry + API, terminal-grid UI, ADR-0004 (defer
  framework). First screenshot evidence committed; visual verdict honestly
  UNVERIFIED (session model lacks image input — recorded above).
- **2026-08-23** — Language policy: English is now the repository's official
  language. Changelog, README, CONTRIBUTING, benchmarks and skills
  translated; policy recorded in AGENTS.md + CONTRIBUTING.md.
