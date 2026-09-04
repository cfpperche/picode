# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Visual gate:** read `uiux-review` before JSX/CSS, then run
`visual-review`. Empty/blocked/error states and screenshots are required;
`window.__picodeOverlayAudit()` must be `ok`.

**Repository:** local `main` contains ADR-0059 feature commit `81ac872b` plus
this deployment handoff; `origin/main` remains `dd98084c` (no push requested).
`make ci` passed at the exact feature commit. `make deploy` installed
`0.1.0+81ac872`; health, embedded assets, migration, browser load, and session
preservation were verified.

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
- Browser tool previews (ADR-0057) render generic `details.preview` frames; a
  package-side emitter and dedicated Browser surface remain open.

### ADR-0059 transient Inbox reply bursts

The shipped feature replaces the consented TUI-to-chat switch with one private
RPC turn on the Inbox item's exact captured session:

- `ask_human` persists `sessionPath`; Reply, Accept, or Decline atomically park
  the item and create one correlated `inbox-burst:<itemID>` task. Missing or
  unsafe exact sessions are refused, and unrelated queue rows are never
  claimed.
- A per-agent control guard serializes replies with pane/session mutations.
  `respawn-pane -k` leaves tmux and browser attachment intact while a holder
  swaps the TUI for one parent-bound RPC writer; logical mode stays interactive.
- Delivery uses one explicit `prompt` and requires an exact normalized user row
  in newly appended JSONL bytes. Replacement-aware, payload-sized verification
  and timestamp-correlated recovery prevent acknowledgements, stale rows,
  cancellation races, or crash ambiguity from duplicating a reply.
- Three bounded attempts reopen the exact item prefilled on pre-delivery
  failure. Generation ownership prevents stale callbacks, feed events, or
  cancellation from touching a newer burst; output is UTF-8-safe and capped.
- Desktop/mobile remain on the terminal surface for
  `receiving → processing → returning → done|failed`; thoughts, chat,
  composer, model picker, and managed mode stay hidden.
- Startup settles holder leases before SQLite opens and fails closed on a
  potentially live writer. The runtime retains its lease through process join;
  Linux adds `Pdeathsig` and holder PID/re-exec checks.
- Holder and direct-respawn restoration get independent deadlines. If both
  fail, a retryable card explicitly replaces the stale session and remounts the
  terminal client. Direct `send-keys` remains an explicit fallback only.

Integration evidence:

- Fake-Pi/tmux, runtime, store, server, and frontend coverage includes exact
  older sessions, one writer, task isolation, control races, delivery/recovery,
  daemon death/re-exec, restoration deadlines, and failed-restart retry.
  Full `make ci`, focused package suites, and race tests passed on `81ac872b`.
- The deployed service is healthy at `0.1.0+81ac872`; its installed binary
  matches `bin/picode`, migration 023 exposes `inbox_items.session_path`, and
  no burst holder marker survived startup. The embedded app serves
  `/assets/index-CNWNR34c.js` with the burst and restart paths present.
- All 54 immediate pre-deploy tmux name/session-ID pairs are unchanged,
  including the original 50. Both interactive agents kept their exact selected
  JSONLs; all 12 terminals and both agents remain running with identical API
  projections.
- The deployed desktop loaded in Chromium with first-party requests succeeding,
  no page errors, and `overlayAudit` ok. The earlier desktop/390×844 visual
  review covered receiving, completion, restarting, failed restart, reduced
  motion, and the real `open?restart=1` path. visual-review: PASS.

## In flight

- **No ADR-0059 implementation, merge, or deployment work remains.** A real-Pi
  Inbox reply/cancel remains deliberately separate dogfood and has not been
  authorized or performed.
- **Historical Inbox QA state needs reconciliation before dogfood.** A prior
  failed reply is absent from the captured `mobile-6bf740` JSONL. Earlier
  cleanup targeted `qa-switch-058577` while the pending task belonged to
  `mobile-6bf740`; the existing `[Teste 3]` item/task must be inspected and
  resolved deliberately before filing another live question.
- **ADR-0054 extension actuator** is coded on its own branch but still needs a
  real model-emitted `picode-act` dogfood before integration.
- **Browser preview follow-through:** upstream proposal/package emitter and a
  Browser panel remain open; the generic conversation renderer is shipped.
- **Remote modes:** real second-account, container, and public OIDC acceptance
  runs require owner-controlled sudo/infrastructure.

## Next up

1. Diagnose why the service process regularly survives SIGTERM until systemd's
   30-second stop timeout, and prove the bounded shutdown path before the next
   deployment.
2. Inspect the live store's historical Inbox item and correlated tasks; close
   or repair only the exact stale rows. Do not create another live test first.
3. With separate authorization, run one controlled real-Pi Inbox reply and a
   second Cancel turn against the exact captured session.
4. Continue the browser-preview emitter/panel and ADR-0054 real-page dogfood.
5. Run the owner-controlled remote-mode acceptance matrix, then decide the
   SaaS track.
6. Build Providers Models/Activity only after confirming their current study
   still matches Pi's provider data.

## Known debts / open questions

- This deployment again hit the pre-existing `picode.service` stop timeout: the
  old daemon survived SIGTERM until systemd SIGKILLed only its main process at
  30 seconds. `KillMode=process` preserved every tmux session and the new daemon
  started healthy with no burst lease, but the graceful-stop hang is unresolved.
- Terminal detail still makes two swallowed agent-only requests (`role-state`
  and `slash`); this predates the manual Pi sensor and does not affect state.
- ADR-0059 hard parent-death enforcement is Linux-specific; non-Linux builds
  rely on graceful shutdown and the holder's daemon-PID polling. Cross-platform
  hard-crash behavior has not been live-proved.
- The explicit last-resort restart intentionally replaces tmux identity only
  after both identity-preserving restoration paths have failed.
- A crash after user-row materialization but before `agent_settled` leaves the
  response potentially incomplete. Startup correlates the full payload and row
  timestamp, marks the task delivered, restores the TUI, and never replays the
  same user message automatically.
- The burst follows the session captured when `ask_human` filed the item. That
  is deliberate even if the operator later browses another TUI session.
- Direct `send-keys` delivery is available only as a deliberate fallback; no
  automatic fallback bypasses exact-session verification or consent.
- Hosted CI is green on Ubuntu, macOS, and Windows after PR #2. Hard-crash
  burst behavior outside Linux still lacks a live platform acceptance run.
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

- **2026-09-04 — ADR-0059 integrated and deployed.** Local `main` fast-forwarded
  to gated feature commit `81ac872b`; `make deploy` now serves
  `0.1.0+81ac872`. The original 50 and immediate 54 tmux identities, both exact
  agent session paths, and all 12 terminals survived unchanged. Health,
  migration, embedded assets, and a live Chromium smoke passed. The recurring
  systemd stop timeout is recorded as debt; real-Pi Inbox dogfood was not run.
  visual-review: PASS (pre-merge edge-state screenshots read; post-deploy app
  read, no page errors, first-party requests healthy, overlayAudit ok).

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
