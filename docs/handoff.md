# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** local `main` includes Docker App/sysadmin delivery
`a2e377ef` and all prior work through `c117fa46`. Tracked files are clean.
The pre-existing untracked `.pi/compact.json` was preserved; commits remain
local and unpushed. ADR-0064 remains proposed.

**Deployment:** the installed service runs `0.1.0+a2e377e` (`release: false`).
Health returned `ok`, boot `baddf1edaa26e732`; all 123 tmux sessions survived.
Installed binary and served UI asset hashes match the validated build.
The deployed Docker inventory and container details work. Scratch daemons,
owned browser sessions and the disposable QA container were cleaned up.

**Quality:** post-integration `make ci` passes: Go tests, 472 frontend tests,
all Pi package suites (including sysadmin confirmation/connection coverage),
embedded build, docs/OpenAPI/llms parity and Vale. Targeted Docker/store race
tests passed. The ADR-0065 decision table links each outcome to coverage.

**UI evidence:** reviewed desktop light/dark, mobile, empty, blocked, failed
operation and confirmation states. Curated evidence is in
`docs/screenshots/docker-*.png`. The mobile title/state/ID fit the viewport;
confirmation overlays return `window.__picodeOverlayAudit().ok: true`.
visual-review: PASS. Deployed mobile details/confirmation were also read;
audit stayed `ok`, Cancel worked, and no page errors were reported. Generated
docs captures were refreshed and read.

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
  CLI is never promoted to an Agent. CLI badges wear the runtime's official
  favicon (first working link wins), rendered bare like workspace favicons;
  the compact text mark is only an asset-load fallback.
- Apps now provide literal output, progress, empty lists and generic mobile
  navigation. Docker exposes inventory, details, sampled resources/logs,
  start/stop/restart and durable history. Optional `pi-sysadmin` tools share
  the same service, existing authentication and local Unix socket access.
- Public docs use VitePress, generated OpenAPI, Vale, committed screenshots,
  and integrity-checked tutorial videos.
- Public release mechanics are tag-driven. The cadence study, proposed
  ADR-0064 and maintainer checklist are documented, but no calendar-triggered
  release or Preview lane is active.

### ADR-0061 compaction policy package (`pi-compact`)

Shipped, deployed, then amended after the first real compaction (owner
directive: **no defaults, ever**). The primary checkout now has an untracked
`.pi/compact.json` from other local work; its configuration and live dogfood
were not evaluated in this Docker session:

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

1. Review current local `main` and decide when to push/promote it; merge and
   deploy are complete locally.
2. Review ADR-0064 and choose the official cadence/pilot window; no release
   date is committed yet.
3. Verify the local `pi-compact` configuration and re-dogfood its policy.
4. Inspect the exact historical Inbox rows before any real TUI reply test.
5. Run the owner-controlled remote-mode acceptance matrix.
6. Continue the Browser preview panel and ADR-0054 dogfood.
7. Extend Docker with Compose deployment after defining target ownership,
   secret handling and recovery semantics in an ADR; v1 operations are covered.
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

- **2026-09-04 — Docker App deployed; pi-sysadmin package ready (ADR-0065).**
  Extended existing Apps primitives and mobile navigation; added bounded
  Engine API operations, idempotent background jobs, verified outcomes and
  durable history. Real disposable-container start/stop/restart passed.
  Event bursts initially starved detail reads; container-state event filters
  and a serialized refresh queue fixed the observed failure, with regression
  coverage. Plain-text logs, empty/blocked/error and mobile confirmations
  passed screenshot review. `make ci` passed. visual-review: PASS.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
