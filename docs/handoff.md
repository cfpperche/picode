# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Visual gate:** read `uiux-review` before JSX/CSS, then run
`visual-review`. Empty/blocked/error states and screenshots are required;
`window.__picodeOverlayAudit()` must be `ok`.

**Repository:** local `main` and `origin/main` are at `dd98084c`. The
installed service is at `b7c63d34` (`0.1.0+b7c63d3`) from the cross-platform
CI deployment. ADR-0059 is rebased onto `dd98084c`; the owner authorized merge
and deployment. Focused integration/race suites, docs parity, and final
visual review pass; the exact amended commit still needs its full `make ci`.

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

The isolated feature branch replaces the consented TUI-to-chat switch with one
private RPC turn on the Inbox item's exact captured session:

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

Evidence in this branch:

- Fake-Pi/tmux, runtime, store, server, and frontend tests cover exact older
  sessions, one writer, task isolation, control races, rollback, retries,
  cancellation, large/escaped payloads, daemon death/re-exec, crash recovery,
  stale feed generations, restoration deadlines, and failed-restart retry.
- Focused Go/frontend/package suites and race tests for runtime shutdown and
  server burst/control paths pass after the final rebase. Docs screenshots and
  all three HyperFrames tutorials were regenerated; manifests and media probes
  pass. The exact amended commit still needs its full `make ci`.
- Desktop and 390×844 mobile screenshots were read after the final rebase for
  receiving, completion, restarting, retry-after-failed-restart, and reduced
  motion. Return reached exact-agent `open?restart=1`; retry cards survived
  failure, page errors were empty, reduced motion reported no animation, and
  desktop/mobile overlay audits returned `{"ok":true,"hits":[],"rows":[]}`.

## In flight

- **ADR-0059 is rebased and awaiting its exact-commit `make ci`.** Merge and
  deployment are owner-authorized. A real-Pi Inbox reply remains deliberately
  separate dogfood and has not been authorized or performed.
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

1. Run `make ci` on the amended ADR-0059 commit, fast-forward `main`, deploy
   once, and prove service health, served assets, and unchanged tmux identities.
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

- **2026-09-04 — Hosted CI restored across Ubuntu, macOS, and Windows.** PR #2
  merged as `b7c63d34`; the follow-up handoff commit is `dd98084c`. Linux and
  macOS run the daemon suite; native Windows compiles every package/test with
  race instrumentation and exercises the tray/browser-host boundary. The
  active service was deployed at `0.1.0+b7c63d3`.
- **2026-09-03 — ADR-0059 transient RPC burst implemented and gated**
  (`feat/transient-rpc-burst`). Inbox replies to an interactive Pi agent now
  borrow the exact asking session for one private RPC turn, prove durable
  JSONL delivery, stream a terminal-only lifecycle, restore the same tmux
  pane, and recover/cancel without exposing chat mode. Final audit added
  agent-scoped mobile takeover, process-join writer leases, pre-store marker
  release, timestamp-correlated delivered-on-crash recovery, selected-session
  rollback, exact delivery matching, large-row recovery, fresh fallback
  deadlines, and force-restart/reconnect coverage. `make ci` and focused race
  tests are green. visual-review: PASS (desktop/mobile done exit, restart,
  retry, and reduced-motion captures read; both `open?restart=1` actions
  clicked; overlayAudit ok; visual card 5/5).
  Not merged or deployed.
- **2026-09-03 — Manual Pi TUI terminal status completed (ADR-0056).** The
  opt-in scoped wrapper reports idle, working, needs-you, prompt return,
  completion, and interruption without touching the user's Pi configuration;
  owner acceptance and live desktop/mobile dogfood passed.
- **2026-09-03 — Three tutorial videos shipped.** The docs harness now captures
  deterministic desktop/mobile stills, renders the HyperFrames compositions,
  and parity-checks the published videos in CI.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
