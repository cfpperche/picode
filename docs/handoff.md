# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Visual gate:** read `uiux-review` before JSX/CSS, then run
`visual-review`. Empty/blocked/error states and screenshots are required;
`window.__picodeOverlayAudit()` must be `ok`.

**Repository:** this integrated `main` contains the ADR-0060 refactor
(`a9c814a`, build `0.1.0+a9c814a`, deployed and verified), its handoff, the
README refactor and current generated screenshots. Tutorial video integrity
still gates CI, but strict freshness against the global UI tree is now an
explicit maintenance audit rather than a delivery blocker. The owner approved
publishing the accumulated local commits with this merge. The ADR-0059 burst
machinery is removed: Inbox replies now land directly in the running TUI. The
owner's systemd stop bound keeps deploys stopping cleanly (verified: no
SIGKILL, no timeout).

**Feature branch:** `feat/whats-new` carries ADR-0062, the in-product What’s
New release highlights surface. It is implemented for desktop and mobile and
awaits owner review/merge; the primary checkout remains untouched.

### Product and platform

- One Go binary serves the React/Vite desktop and mobile ADE. HTTPS defaults
  to `:8445`; hashed assets may cache, while HTML and APIs do not.
- Workspaces contain multiple agents; free agents and first-class tmux
  terminals are supported. Agent sessions are privately scoped and recorded
  in `agent_sessions` (ADRs 0039/0040/0053).
- Agents have interactive (Pi TUI in tmux) and managed (Pi RPC) run modes.
  The process lock and runtime lifecycle enforce one writer per session.
- Desktop/mobile use the ADR-0048 change feed (polling only for recovery), and
  every state-changing store method emits its event in the same transaction.
- Shipped surfaces include Inbox, automations, sessions/files/git, providers,
  MCP/packages, terminals, Web Push, auth/pairing and truthful coding-CLI/TUI
  status. Public
  docs use VitePress/OpenAPI/Vale. Architecture and remaining track status live
  in their ADRs rather than this handoff.
- The root README now leads with a generated product view, shipped capabilities,
  Pi ownership boundaries and a verified first-agent path. Source setup matches
  `go.mod` and hosted CI (Go 1.26, Node.js 22); the public getting-started page
  and generated `llms.txt` carry the same requirements.
- Browser tool previews (ADR-0057) render generic `details.preview` frames; a
  package-side emitter and dedicated Browser surface remain open.
- Public docs keep blocking parity for screenshots, OpenAPI and `llms.txt`.
  Video CI is hash-only integrity; `make docs-videos-fresh` is the strict
  manual audit and `make docs-videos` performs the expensive refresh.

### ADR-0060 Inbox replies land in the running TUI

ADR-0059's transient burst is superseded and removed (net −1,868 lines). The
reply never leaves the terminal:

- Every spawned agent TUI carries PiCode's generated receiver extension
  (`~/.picode/intercept/pi-inbox-reply.ts`, `-e` at spawn, refreshed at boot).
  It hellos `POST /api/agents/{id}/tui-hello` (5-minute lease) and consumes
  one-shot files under `~/.picode/tui-inbox/<agentID>/`, submitting each reply
  via `pi.sendUserMessage` (queued natively mid-turn — owner decision) and
  acking `POST /api/agents/{id}/tui-ack`.
- Without a fresh hello (legacy TUI), the daemon types the reply into the pane:
  tmux named buffer + `paste-buffer -p` + Enter (owner accepted the
  draft-race tradeoff).
- Durable proof is unchanged: the captured session JSONL must gain the
  full-payload user row; otherwise the task fails and the Inbox item reopens
  with the response prefilled. Boot reconciliation (`ReconcilePendingReplies`)
  settles pending replies with a 2s grace — no holders, leases, Pdeathsig, or
  fail-closed startup remain. A per-agent `AgentControls` guard serializes
  replies with pane/session mutations; `open?restart=1` survives for dead
  panes; migration 023 and the exact-session rule are retained.

Integration evidence: `tui_reply_test.go` covers receiver ack/nack/no-ack,
paste fallback against a real tmux pane, refusals, boot reconciliation, and
receiver injection; store/apps/rpc suites updated. `make ci` green on the
exact deployed commit; installed binary matches `bin/picode`; the receiver
file exists under `~/.picode/intercept/`; burst routes 404; no tmux session
lost across the deploy (additions only, 68 sessions). The passive-UI burst fix
never shipped separately — the burst it fixed no longer exists.

## In flight

- **ADR-0062 What’s New release highlights is implemented on `feat/whats-new`,
  pending review/merge.** Curated notes are bundled offline, opened once per
  stamped semver per browser, and deferred behind Inbox/create/share/reconnect
  work. Desktop user menu and Ctrl+K, plus mobile More, provide manual entry.
  The release workflow validates matching changelog/catalog entries and
  publishes the full changelog section.
  `make ci` is green; the desktop light/dark, mobile sheet and empty-note
  screenshots were read, overlay audits were `ok`, and a stamped-build
  auto-open/reload cycle was verified on an ephemeral daemon.
- **ADR-0061 `packages/pi-compact` merges this session.** Opt-in MIT pi
  package: early compact (100k tokens or 50% of the window, floor 32k),
  cheap summarizer with thinking off (Flash → Haiku → session),
  `/compact` overlay (`edit` / `model` / `on` / `off`). Active defaults
  without a file; Pi overflow compact and the recent-token tail stay.
  `PI_COMPACT_AGENT` joins `Agent.SpawnEnv`. 51 package tests (tsc gate
  against pinned pi 0.85 types + pure table + handler fakes); the tsc
  gate caught a real API mismatch during authoring. Owner-requested
  fold-in on the same branch: the user-menu Documentation link opens
  the docs site (verified live on an ephemeral daemon; screenshot read).
  After deploy: dogfood with a real reload — `/compact edit`, an early
  trigger, the Flash-missing fallback. Composer chip / agent settings
  GUI is explicitly M2 (ADR-0061).
- **ADR-0060 live validation passed (2026-09-04 15:25).** A real reply to the
  `[Teste ADR-0060]` item landed inside this TUI, task delivered in 2.1s via
  the exact JSONL row, item done, tmux identity intact, no leftovers. The
  "Open terminal" card action was removed afterwards as noise (hotfix,
  deployed `0.1.0+f8b140f`).
- **Historical Inbox QA state needs reconciliation before dogfood.** A prior
  failed reply is absent from the captured `mobile-6bf740` JSONL. Earlier
  cleanup targeted `qa-switch-058577` while the pending task belonged to
  `mobile-6bf740`; the existing `[Teste 3]` item/task must be inspected and
  resolved deliberately before filing another live question.
- **ADR-0054 extension actuator** is coded on its own branch but still needs a
  real model-emitted `picode-act` dogfood before integration.
- **Browser preview follow-through:** upstream proposal/package emitter and a
  Browser panel remain open; the generic conversation renderer is shipped.
- **Tutorial video maintenance needs an incremental design.** The temporary
  policy removes global UI-tree freshness from CI while retaining composition,
  referenced-still and MP4 integrity. Decide how to map changed surfaces to
  only the affected tutorial, cache its capture/render, and run the strict
  audit manually or on a schedule without blocking ordinary delivery.
- **Remote modes:** real second-account, container, and public OIDC acceptance
  runs require owner-controlled sudo/infrastructure.

## Next up

1. Review and merge `feat/whats-new` (ADR-0062), then include the next
   release's curated entries in `web/src/data/whats-new.json` before tagging.
2. Dogfood `pi-compact` after the deploy (reload an agent, `/compact edit`,
   early trigger on a long session).
3. pi-compact M2 (composer chip / agent settings) only after dogfood.
4. The `tsc` type gate exists only in `packages/pi-compact`; `pi-roles`,
   `pi-inbox` and `pi-checklist` still ship un-typechecked (jiti at
   runtime). Same recipe applies when someone touches them.
5. `pi-roles` / `pi-compact` npm publishing is not automated; local path
   installation remains the supported route. pi 0.85's npm package has a
   broken pure-ESM graph (`@earendil-works/pi-server` missing), so
   pi-compact carries it as a devDependency for handler tests.
6. Inspect the live store's historical Inbox rows; close or repair only the
   exact stale rows before any new live test.
7. Design the incremental docs-video maintenance path: per-tutorial input
   fingerprints, selective capture/render, cache boundaries and the trigger
   policy (manual, scheduled or both).
8. Continue the browser-preview emitter/panel and ADR-0054 real-page dogfood.
9. Run the owner-controlled remote-mode acceptance matrix, then decide the
   SaaS track.
10. Build Providers Models/Activity only after confirming their current study
   still matches Pi's provider data.

## Known debts / open questions

- A deploy onto a daemon that still has the old binwatch will SIGKILL once
  (the outgoing process re-execs). The next restart of `0.1.0+6cf705d` stops
  in the 5s HTTP drain with no re-exec and no SIGKILL. `KillMode=process`
  stays as the tmux safety net. Long-lived feeds still occupy that drain.
- Terminal detail still makes two swallowed agent-only requests (`role-state`
  and `slash`); this predates the manual Pi sensor and does not affect state.
- The ADR-0060 tmux paste fallback can land in a draft the operator had open,
  and cannot verify which session a legacy pane is showing — the JSONL row
  proof still gates whether the item stays done (owner-accepted tradeoffs).
- A receiver ack means the TUI owns the queued reply; if that TUI dies before
  pi processes it, boot/timeout reconciliation reopens the item (small honest
  window).
- Hosted CI is green on Ubuntu, macOS, and Windows after PR #3. Full-path PR
  run `33878224835` and post-merge main run `33878695007` passed in
  3m54s/3m52s; metadata-only run `33879212363` passed in 32s with every heavy
  job skipped; follow-up run `33879325513` repeated that row in 27s.
  Cross-platform acceptance of the ADR-0060 reply path (non-Linux paste
  encoding) has not been live-proved.
- Tutorial videos can now be visually stale after an unrelated UI-tree change
  without failing CI. Missing/tampered MP4s and changed compositions or
  referenced stills still fail. Until the incremental design above ships, run
  `make docs-videos-fresh` during deliberate docs maintenance.
- Pi still exposes one active credential slot; concurrent agents share it.
  Per-agent OAuth isolation and proactive quota switching require an owner
  decision and measurement.
- ADR-0048 ephemeral events can be missed across reconnects until a later
  state change or explicit reconciliation.
- Checklist follow-ups remain: align client/server validation limits and add
  a reminder-loop guard if Pi changes its follow-up hook invariant.
- `pi-roles` publishing is not automated; local path installation remains the
  supported development route.
- The terminal Shift+Enter shim remains until a stable xterm/tmux protocol
  combination replaces it without changing other control-key semantics.
- Branch protection and CODEOWNERS still require owner action on GitHub.

## Recent activity

- **2026-09-04 — What’s New release highlights (ADR-0062) implemented on
  `feat/whats-new`.** Added the shared ResponsiveDialog surface, bundled
  semver catalog and local acknowledgement policy to both shells; wired
  desktop menu/palette and mobile More entry points; exposed the stamped-build
  flag from `/api/version`; and made releases publish their changelog section
  as notes. Unit, Go, docs parity and full `make ci` pass. Visual-review: PASS
  (desktop light/dark, mobile sheet and empty-note PNGs read; overlay audit
  `ok`; primary Got it action clicked and state stayed closed after reload).
- **2026-09-04 — pi-compact (ADR-0061) merged and deployed (`0.1.0+21d7144`).**
  Rebased onto the new CI process (video freshness decoupled from
  delivery; docs media regenerated for the UserMenu change). 51 package
  tests + full `make ci` green; docs parity re-shot. Deploy verified:
  health 200, served asset matches the built bundle (UserMenu docs link
  live), service active, 89 tmux sessions intact. visual-review: PASS
  (menu click verified on an ephemeral daemon earlier).
- **2026-09-04 — "Open terminal" removed from Inbox item cards
  (`32158c13`, deployed `0.1.0+f8b140f`).** With ADR-0060 the reply itself
  lands in the agent's terminal, so the card-level escape hatch read as
  noise. Action, host wiring, and the `open-terminal` case deleted; the view
  test now asserts the card has no such action. First CI run under the new
  process (video freshness decoupled) is green. visual-review: n/a
  (removal only).
- **2026-09-04 — tutorial video rendering left the delivery critical path.**
  Default docs CI now checks composition, referenced-still, render and shipped
  MP4 integrity without treating every global UI-tree change as a mandatory
  three-video refresh. The strict comparison moved to
  `make docs-videos-fresh`; capture/render remains explicit in
  `make docs-videos`. A four-row decision table covers exact parity, unrelated
  UI drift, changed rendered inputs and missing/tampered MP4s. Current app
  screenshots were regenerated and read; the interrupted video render was not
  retained. The first hosted run also exposed an inherited trailing blank line
  in a frontend test; the whitespace gate is corrected. Because the current
  screenshot fingerprint includes test files, that no-pixel change forced a
  second screenshot capture and confirms the need for per-surface inputs.
  Full `make ci` passed. visual-review: PASS (generated desktop and mobile docs
  screenshots read; no layout defect, overlay change or new state).
- **2026-09-04 — root README refactored around the reader's journey.** Product
  value and proof now lead into shipped capabilities, native Pi boundaries,
  current setup, daily commands and a compact runtime diagram; the stale
  milestone roadmap is gone. Public setup metadata and `llms.txt` were kept in
  sync. Markdown render/local references, docs build/parity, Vale and full
  `make ci` passed. visual-review: N/A (docs-only; generated screenshot read,
  no app pixels changed).
- **2026-09-04 — ADR-0060: replies land in the running TUI (`a9c814a`,
  `0.1.0+a9c814a`).** The ADR-0059 burst machinery (coordinator, holder swap,
  transient RPC writer, cancel route, burst card, feed events) is deleted;
  a receiver extension inside every spawned TUI submits Inbox replies through
  `pi.sendUserMessage`, with tmux bracketed paste as the legacy fallback and
  the session-JSONL row as the only delivery proof. Boot reconciliation
  replaced holder/lease startup. `make ci` green; deploy stopped cleanly with
  no tmux loss. Worktree and branch removed after merge.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
