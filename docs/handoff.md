# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** `feat/sidebar-tui` is an isolated worktree based on
`06f45c84`. The sidebar refactor and terminal-presence work are committed on
this branch. Nothing was merged to `main` and nothing was (de)ployed; owner
authorization is still required for either action.

**Product:** desktop and mobile use compact supervision rows. Agent rows lead
with identity and activity; terminal rows show a CLI mark, truthful
presence/activity, subdued path/branch context, and Radix secondary actions.
The layout was checked at desktop `1280×633` and mobile `390×844`, including
populated, empty, blocked, error, and menu states.

**Terminal presence:** ADR-0056 remains the activity sensor contract and
ADR-0060 adds authoritative, ephemeral presence. Opt-in session-only wrappers
for Claude Code, Codex, Grok, and Pi announce a run id/PID to
`POST /api/terminals/{id}/runtime`; lifecycle-aware wrappers report end and
maintenance commands preserve their original argv behavior. The server
revalidates leases against tmux and process identity, reconciles legacy
sessions only from exact pane command/PID data, and publishes
`terminal.runtime` feed events. Run IDs protect a newer process from stale
state/end events. `tui` is authoritative in the browser; legacy `cli` remains
only as a compatibility projection. TUI is still a terminal, never a guest
Agent.

**Documentation:** `docs/decisions/0060-terminal-cli-presence.md`,
`docs/architecture.md`, `www/guide/terminal-status.md`, generated public docs
artifacts, and `[Unreleased]` in `CHANGELOG.md` describe the feature. No
SQLite migration or new React polling loop was added.

**Validation:** `visual-review: PASS` was completed from read screenshots for
populated, empty, blocked, error, and menu states on desktop and mobile.
`window.__picodeOverlayAudit()` returned `ok: true` for the audited menus;
the clean browser audit had no console/page errors or unexpected 4xx/5xx
requests. `make fmt-check`, `make vet`, `go test ./...`,
`npm --prefix web test` (461), `make build`, `make vale`, and the final
`make ci` passed. `make docs-shots` regenerated 10/10 stills and
`make docs-videos` regenerated the three tutorial videos. QA servers, tmux
sessions, and temporary directories were removed; ports 18445, 18446, and
18740 are free.

## In flight

- No implementation work is in flight on this branch. Merge and deployment
  remain blocked until the owner explicitly authorizes them.
- Separate pre-existing follow-ups remain deliberate: reconcile the historical
  Inbox QA item before any new live question, and keep the owner-controlled
  dogfood and remote-mode work below separate from this branch.

## Next up

1. Owner reviews the committed branch and authorizes or declines the merge.
2. If authorized, merge only through the repository worktree policy; remove
   the worktree and feature branch after the merge.
3. If separately authorized, deploy and perform the production health check.

## Known debts / open questions

- Presence is authoritative only for terminals launched through an enabled
  PiCode wrapper. Older sessions have exact command/PID presence fallback but
  must be recreated for wrapper lifecycle instrumentation.
- `/proc` process-start tokens are strongest on Linux; other platforms retain
  liveness checks but have weaker PID-reuse protection.
- Vendor wrapper and hook schemas can change. Unsupported or ambiguous
  commands deliberately degrade to `Terminal open` or no CLI identity; pixels,
  titles, and screen scraping remain refused.
- Ephemeral feed events can be missed across reconnects; the initial terminal
  list and tmux reconciliation repair the projection, but a daemon restart
  forgets in-memory leases until a wrapper/fallback reports again.
- Legacy agent-only `role-state` and `slash` requests are still used by other
  terminal-detail paths; desktop terminal tabs no longer request them.
- Guest CLI promotion, composer control, Inbox/Web Push integration, and ACP
  remain deferred under ADR-0056. Any change to that boundary needs a new ADR
  and owner approval.
- Historical Inbox QA still needs deliberate reconciliation before dogfood:
  inspect the exact `[Teste 3]` item/task and captured `mobile-6bf740` session;
  the earlier cleanup targeted `qa-switch-058577`, so do not create another
  live question first.
- The ADR-0054 extension actuator still needs a real model-emitted
  `picode-act` dogfood before integration.
- Browser-preview follow-through (package-side emitter and dedicated Browser
  panel) remains open; generic conversation rendering is shipped.
- Owner-controlled second-account, container, and public-OIDC acceptance runs
  remain open for remote modes.
- Real vendor CLI dogfood was not authorized in this session. Wrapper tests
  use deterministic fake binaries; live user configuration was not modified.
- One standalone `go test ./internal/server` run exceeded its 120-second
  exploratory timeout; the complete final `make ci` passed.

## Recent activity

- **2026-09-04 — sidebar supervision refactor and CLI presence
  (visual-review: PASS; make ci: PASS).** Added compact desktop/mobile rows,
  shared CLI/status language, authoritative wrapper leases, run-id-safe feed
  reducers, tmux/PID fallback, tests, ADR-0060, architecture, guide,
  changelog, generated docs evidence, and final browser QA.
- **2026-09-04 — systemd stop fix deployed (`6cf705dd`, `0.1.0+6cf705d`).**
  This branch starts after the verified stop/drain fix; no deployment was done
  for the current sidebar work.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
