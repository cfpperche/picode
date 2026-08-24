# Handoff — living project state

> **This file is the heartbeat of PiCode.** Any session (human or agent)
> that changes state MUST update this file before ending.
> Ritual defined in `/skill:handoff-update`; contract in [AGENTS.md](/AGENTS.md).
>
> Format rules: newest entries at the top of "Recent activity"; keep it
> short — this is a handoff, not a diary. Archive old blocks to
> `docs/handoff-archive.md` when the file exceeds ~150 lines.

## Current state (read this first)

**Phase: store shipped (ADR-0005). M2 — Agent panel (next, queue API ready).**

What exists right now:
- Repo public at `cfpperche/picode`, MIT, CI green (linux/macos/windows).
- **M1 shipped**: `picode screenshot` (chromedp); tmux manager (`internal/tmux`);
  WS↔PTY terminal bridge (`internal/term`); dark-first UI (vanilla ES +
  xterm.js, ADR-0004) — sidebar cards, tabs, live statusbar, Vercel-style
  user menu, settings route, theme light/dark/system.
- **Store shipped**: SQLite `~/.picode/picode.db` (pure-Go driver), embedded
  migrations, legacy JSON imported+retired. Tables: workspaces, agents
  (default per workspace), tasks (claim/finish state machine), messages
  (M4), events (audit), settings. API: workspace CRUD + open/close +
  `POST/GET /api/agents/{id}/tasks` (tasks persist `queued`).
- Docs: philosophy, architecture (data section), benchmarks (+Cursor bar),
  ADRs 0001–0005, this handoff. Harness: AGENTS.md + 4 skills.
- 23 tests across 5 packages (store CRUD/lifecycle/legacy; tmux/term
  integration skip gracefully where tmux is absent).

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

## Next up (M2 — Agent panel, in order)

1. **RPC agent manager** (`internal/rpc/`): spawn `pi --mode rpc` per
   workspace, JSONL client with strict `\n` framing, event fan-out to
   WebSocket. Wire the **task delivery engine**: `ClaimNextTask` → send →
   `FinishTask` (queue API + state machine already live).
2. **Agent panel UI**: live status, streaming output, tool-call rows
   (Cursor bar: ≤32px collapsed, expandable), task input with
   steer/follow_up semantics (buttons already backed by the API).
3. **Diff view**: surface agent edits from tool-call events.
4. **Command palette** (`Ctrl+K`).

## Known debts / open questions

- **Visual verdicts require a vision-capable model** (see finding above).
- tmux-gated tests skip on windows/macos runners (tmux absent); ubuntu CI
  covers them. Accepted; alternative (CI tmux on all OSes) not worth it now.
- Token auth: still localhost-only bind; `--listen` beyond localhost must
  ship with token auth (contract in architecture.md security model).
- Vendored xterm.js 5.5.0 + fit addon need a manual upgrade story
  (note in ADR-0004); track upstream releases occasionally.
- Branch protection + CODEOWNERS on GitHub — needs owner action (manual).

- **Task delivery engine pending**: `/api/agents/{id}/tasks` persists
  `queued` tasks; the RPC consumer ships with M2 step 1.

## Recent activity

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
