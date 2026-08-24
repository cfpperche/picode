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

- **Visual verdicts require a vision-capable model** (see M1 note); M2
  panel evidence: `docs/screenshots/m2-agentpanel-first-look.png` (pixel
  checks pass; agent verdict UNVERIFIED).
- Panel shows only events observed while connected — no session replay yet;
  replay via session JSONL reader is an M3 candidate.
- tmux-gated tests skip on windows/macos runners (tmux absent); ubuntu CI
  covers them (accepted).
- Token auth: still localhost-only bind; `--listen` beyond localhost must
  ship with token auth (contract in architecture.md security model).
- Vendored xterm.js 5.5.0 + fit addon need a manual upgrade story
  (note in ADR-0004); track upstream releases occasionally.
- Branch protection + CODEOWNERS on GitHub — needs owner action (manual).

## Recent activity

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
