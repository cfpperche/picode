# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** the Docker project grouping refinement (ADR-0066) is in
`feat/docker-project-groups`, based on local `main` `1ed6708f`. Grouping,
search and saved folds are implemented; v2 project operations/Compose are
specified in `docs/plans/docker-v2.md` and are not implemented. Existing
local commits remain unpushed. Preserve the unrelated `.pi/compact.json`.

**Deployment:** the installed service still runs `0.1.0+cf9aafc`. The
refinement is verified on an isolated daemon; deployment follows the gates.

**Quality:** `make ci` passed (Go, 477 frontend tests, packages, build,
docs parity and Vale). Browser QA covered real project labels, mouse/Enter/Space, refresh/reload/Back,
search and saved folds. Empty/blocked/error, standalone groups, both themes,
phone confirmations and long names were captured and read. A narrow-screen
metadata truncation was fixed and re-captured. visual-review: PASS; overlay
geometry is `ok`. Curated evidence: `docs/screenshots/docker-groups-*.png`.

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
  CLI is never promoted to an Agent. CLI favicons fill the same 22px face slot
  as agent rows, with no chip; the compact text mark is only an asset-load
  fallback.
- Apps now provide literal output, progress, empty lists and generic mobile
  navigation. Docker exposes inventory, details, sampled resources/logs,
  start/stop/restart and durable history. Inventory groups exact Compose
  projects inside the app, with saved folds and search (ADR-0066). Optional
  `pi-sysadmin` tools share the same service, existing authentication and
  local Unix socket access.
- Public docs use VitePress, generated OpenAPI, Vale, committed screenshots,
  and integrity-checked tutorial videos.
- Public release mechanics are tag-driven. The cadence study, proposed
  ADR-0064 and maintainer checklist are documented, but no calendar-triggered
  release or Preview lane is active.

### ADR-0061 compaction policy package (`pi-compact`)

Shipped, deployed, then amended after the first real compaction (owner
directive: **no defaults, ever**). The primary checkout now has an untracked
`.pi/compact.json` from other local work; its configuration and live dogfood
were not evaluated in this session:

- **Dormant until configured.** Without `.pi/compact.json` (or a per-agent
  overlay) Pi's stock compaction and summarizer run untouched; the status
  line reads `compact: not configured · /compact-edit`. Any config file is
  the opt-in.
- **Commands are `/compact-edit|model|on|off` (second amendment).** Pi's
  TUI dispatches `/compact …` to its built-in command before extension
  commands run, so `/compact edit` compacted instead of opening the wizard.
  Bare `/compact` stays native; the summarizer hook still applies to it.
- **Trigger on `agent_settled`, never `turn_end`.** `ctx.compact()` starts
  with `abort()`, so the old mid-run trigger killed active runs ("This
  operation was aborted"); `agent_settled` is Pi's own post-run compaction
  boundary, plus an `isIdle()` guard.
- **The summarizer chain retries link by link** (error stops, throws, empty
  or length-capped summaries fall through; Pi's summarizer is the last
  resort). Auto chain: `gemini-3.6-flash` → `claude-haiku-4-5` — 2.5-flash
  now 404s for newer Google accounts. 54 package tests; `make ci` green.

## In flight

- Docker groups passed the gates and await local deployment. V2 project
  operations, Compose registration/deployment and shared tools are proposed
  in `docs/plans/docker-v2.md`; no project-level action is shipped yet.
- Docker v1 is merged and deployed. The real Pi 0.85.0 runtime loaded all
  four tools and executed inventory/detail against the shared API without a
  model turn. Autonomous model-driven dogfood and Docker Desktop/rootless
  acceptance remain open; the Linux Engine path passed real operation QA.
- **ADR-0061's amendment is deployed in the current service.** A configured
  real-compaction re-dogfood remains pending; without a config it stays dormant
  as designed, and the run must record `fromHook: true` plus gemini-3.6-flash
  pricing with no aborted turns.
- Real CLI dogfood was intentionally not run. The historical Inbox `[Teste 3]`
  and `mobile-6bf740` rows still need deliberate reconciliation before a new
  live question is filed.
- ADR-0054 `picode-act` still needs real model-emitted dogfood before merge;
  the Browser preview emitter/panel remains open.
- Second-account, container, public-OIDC, and other remote-mode acceptance
  runs require owner-controlled infrastructure.
- ADR-0064 is proposed: the owner still needs to choose whether to accept the
  three-release, two-week pilot. Official release dates remain unset.

## Next up

1. Implement Docker v2 project actions from `docs/plans/docker-v2.md`: exact
   target preview, shared locks, parent/child results and partial failure QA.
   Compose deployment follows in a separate ADR extending ADR-0065.
2. Review current local `main` and decide when to push/promote it.
3. Review ADR-0064 and choose the official cadence/pilot window; no release
   date is committed yet.
4. Verify the local `pi-compact` configuration and re-dogfood its policy.
5. Inspect the exact historical Inbox rows before any real TUI reply test.
6. Run the owner-controlled remote-mode acceptance matrix.
7. Continue the Browser preview panel and ADR-0054 dogfood.
8. Decide whether selective docs-video capture/render should be scheduled;
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

- **2026-09-04 — Docker project groups and v2 proposal (ADR-0066).** Native
  disclosures stay in the app; exact Compose labels, standalone fallback,
  state summaries, saved folds and search work together. Long phone rows
  wrap. V2 separates detected groups from registered Compose deployments.
  `make ci` passed. visual-review: PASS (screenshots read, overlay audit ok).
- **2026-09-04 — terminal favicons merged and deployed as `0.1.0+cf9aafc`.**
  The owner's `v0.1.0+a2e377e` screenshot still had the leftover chip because
  the first worktree preview was overwritten by the Docker deploy and the
  branch had not been merged. Rebased onto Docker `main`, merged, deployed.
  Live sidebar: Claude/Codex/Grok/Pi fill the 22px slot with no plate.
  visual-review: PASS (`/tmp/picode-term-favicon-deployed.png` read).

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
