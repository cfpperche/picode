# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** local `main` is clean, with the sidebar merge
`9bfd1f01` in its history. While this session was completing the merge, the
workspace-favicon merge (`4e9cedbf`) and its handoff (`66bee74f`) also landed
on local main. The resulting tree preserves the Inbox-to-TUI receiver
(ADR-0060), pi-compact (ADR-0061), compact supervision rows, authoritative
terminal CLI presence (ADR-0062), the favicon fix, and the What’s New release
surface (ADR-0063). Release cadence research, proposed ADR-0064 and the
maintainer runbook are also in the tree; no official cadence or date is set.
Local commits have not been pushed to the remote.

**Deployment:** this session's `make deploy` completed successfully after the
What’s New merge. The installed service is active and serves semver `0.1.0`;
`GET /api/health` returned status `ok`, and `GET /api/version` confirmed the
source build identity (`release: false`). There are currently 101 PiCode-owned
tmux sessions after the restarts.

**Quality:** post-merge `make ci` passes, including Go tests, 464 frontend
tests, package tests, docs/OpenAPI/llms parity, Vale, and the embedded build.
`make docs-shots` captured all three public app surfaces and `make docs-check`
passed.

**UI evidence:** post-deploy desktop and mobile screenshots were read:
`/tmp/picode-postdeploy-desktop.png`, `/tmp/picode-postdeploy-menu-open.png`,
`/tmp/picode-postdeploy-mobile-work.png`, and
`/tmp/picode-postdeploy-mobile-menu2.png`. The populated rows, empty workspace,
CLI labels, menus, reload state, and mobile navigation are legible. Open menus
were inside the viewport and `window.__picodeOverlayAudit()` returned `ok: true`;
Escape closed them. No new browser console/page/network error was observed.
visual-review: PASS.

The What’s New dialog was also read in desktop light/dark, mobile-sheet and
empty-note states; the overlay audit returned `ok`, and the primary Got it
action stayed closed after reload.

### Product and platform

- One Go binary serves the React/Vite desktop and mobile ADE. HTTPS defaults
  to `:8445`; hashed assets may cache, while HTML and APIs do not.
- Workspaces contain multiple agents; free agents and first-class tmux
  terminals are supported. Agent sessions are privately scoped and recorded
  in `agent_sessions` (ADRs 0039/0040/0053).
- Agents have interactive Pi TUI and managed Pi RPC run modes. Inbox replies
  use the injected receiver extension with a tmux paste fallback (ADR-0060).
- Desktop/mobile consume the ADR-0048 change feed. Store mutations append their
  events in the same transaction; ephemeral terminal runtime/state signals are
  deliberately in memory and reconcile through the terminal list.
- Terminal CLI presence and activity remain separate. Wrappers identify
  Claude Code, Codex, Grok, or Pi with a run id; exact tmux command/PID data is
  only a weaker legacy presence fallback. Pixels are never scraped and a guest
  CLI is never promoted to an Agent.
- Public docs use VitePress, generated OpenAPI, Vale, committed screenshots,
  and integrity-checked tutorial videos.
- Public release mechanics are tag-driven. The cadence study, proposed
  ADR-0064 and maintainer checklist are documented, but no calendar-triggered
  release or Preview lane is active.

### ADR-0061 compaction policy package (`pi-compact`)

Shipped, deployed, then amended after the first real compaction (owner
directive: **no defaults, ever**):

- **Dormant until configured.** Without `.pi/compact.json` (or a per-agent
  overlay) Pi's stock compaction and summarizer run untouched; the status
  line reads `compact: not configured · /compact edit`; bare `/compact` and
  `/compact on|off` report "not configured". Any config file is the opt-in.
- **Trigger on `agent_settled`, never `turn_end`.** `ctx.compact()` starts
  with `abort()`, so the old mid-run trigger killed active runs ("This
  operation was aborted"); `agent_settled` is Pi's own post-run compaction
  boundary, plus an `isIdle()` guard.
- **The summarizer chain retries link by link** (error stops, throws, empty
  or length-capped summaries fall through; Pi's summarizer is the last
  resort). Auto chain: `gemini-3.6-flash` → `claude-haiku-4-5` — 2.5-flash
  now 404s for newer Google accounts. 59 package tests; `make ci` green.

## In flight

- **ADR-0061 amendment is deployed (`0.1.0+18e6788`); it needs one more
  owner restart, then a configured re-dogfood.** After restarting PiCode; sessions stay dormant (status
  "not configured") until `/compact edit` writes a config — or opt in, and a
  real compaction must record `fromHook: true` + gemini-3.6-flash pricing
  with no aborted runs.
- Real CLI dogfood was intentionally not run. The historical Inbox `[Teste 3]`
  and `mobile-6bf740` rows still need deliberate reconciliation before a new
  live question is filed.
- ADR-0054 `picode-act` still needs real model-emitted dogfood before merge.
  The Browser preview emitter/panel remains open.
- Second-account, container, public-OIDC, and other remote-mode acceptance
  runs require owner-controlled infrastructure.
- ADR-0064 is proposed: the owner still needs to choose whether to accept the
  three-release, two-week pilot. Official release dates remain unset.

## Next up

1. Owner restarts PiCode (new binary), then chooses: leave agents dormant or
   write a config (`/compact edit`); re-dogfood compaction expecting
   `fromHook: true` + 3.6-flash pricing and no aborted runs.
2. Review ADR-0064 and choose the official cadence/pilot window; no release
   date is committed yet.
3. Review the current local main and decide when to push/promote it; the
   What’s New merge and deploy are complete locally.
4. Inspect the exact historical Inbox rows before any real TUI reply test.
5. Run the owner-controlled remote-mode acceptance matrix.
6. Continue the Browser preview panel and ADR-0054 dogfood.
7. Decide whether selective docs-video capture/render should be scheduled;
   current explicit capture and integrity gates already pass.

## Known debts / open questions

- Wrapper presence is strongest for instrumented sessions. Legacy sessions
  get only exact command/PID fallback; Linux process identity has stronger
  `/proc` protection than platforms without start tokens.
- ADR-0048 ephemeral events can be missed across reconnects until a later
  state change or explicit reconciliation.
- Cross-platform acceptance of the ADR-0060 paste fallback has not been
  live-proved. A receiver ack also leaves a small window before Pi processes
  the queued reply.
- Pi still exposes one active credential slot; per-agent OAuth isolation and
  proactive quota switching need an owner decision and measurement.
- The terminal Shift+Enter shim remains until a stable xterm/tmux protocol
  combination replaces it. Some non-terminal screens still carry legacy
  `role-state`/`slash` assumptions.
- Tutorial CI checks committed integrity without rendering every changed
  tutorial; `make docs-videos-fresh` remains the explicit drift audit.
- Branch protection and CODEOWNERS still require owner action on GitHub.

## Recent activity

- **2026-09-04 — release cadence process documented (proposed ADR-0064).**
  Added a benchmark study covering VS Code, Linear, Zed, Cursor and Go; a
  proposed source/dogfood versus Stable release-lane decision; and a
  maintainer runbook covering scope freeze, quality gates, tagging, artifact
  verification, observation and hotfixes. Linked the documents from the
  contributor, README, benchmark and ADR indexes. No official cadence, date,
  Preview channel or scheduled workflow was activated. `make ci` passed.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
