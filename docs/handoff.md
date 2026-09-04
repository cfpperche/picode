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
still gates CI without capture or rendering. Screenshot parity and the strict
manual video audit now use named, per-surface input fingerprints instead of a
global UI-tree hash. The owner approved publishing the accumulated local
commits with this merge. The ADR-0059 burst machinery is removed: Inbox
replies now land directly in the running TUI. The owner's systemd stop bound
keeps deploys stopping cleanly (verified: no SIGKILL, no timeout).

Workspace cards still miss Next.js App Router marks on the deployed binary
(`0.1.0+21d7144`). Branch `feat/workspace-favicon-nextjs` (worktree
`.worktrees/workspace-favicon-nextjs`) extends favicon lookup to `icon.svg`
and `apps/<name>/{public,app,src/app}` — Cognixse's
`apps/web/app/icon.svg` is found. Not merged, not deployed. The primary
checkout has an unfinished merge; leave it.

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
  Screenshot profiles exclude tests and unrelated handlers while following
  each screen's imported code and selected data producers. Video CI is
  hash-only integrity; `make docs-videos-fresh` reports drift by tutorial and
  surface, and `make docs-videos` performs the expensive refresh.

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

- **Workspace Next.js favicon** on `feat/workspace-favicon-nextjs`. Lookup
  covers `icon.svg`/`png`/`ico` plus `apps/<name>/{public,app,src/app}`
  (`apps/web` first). Server tests pass, including a live check of
  `/home/goat/cognixse/apps/web/app/icon.svg` (24 673 bytes). UI is unchanged
  (`WsFavicon` already renders the endpoint); a page-load Set caches a 404,
  so deploy must be followed by a reload. visual-review: N/A (no JSX/CSS).
- **ADR-0060 is deployed; the live validation reply is the owner's next move.**
  This TUI (`mobile-6bf740`) predates the receiver, so its first live reply
  exercises the tmux paste fallback; respawning a TUI (`open?restart=1` or a
  fresh Start) switches that agent to the receiver channel.
- **Historical Inbox QA state needs reconciliation before dogfood.** A prior
  failed reply is absent from the captured `mobile-6bf740` JSONL. Earlier
  cleanup targeted `qa-switch-058577` while the pending task belonged to
  `mobile-6bf740`; the existing `[Teste 3]` item/task must be inspected and
  resolved deliberately before filing another live question.
- **ADR-0054 extension actuator** is coded on its own branch but still needs a
  real model-emitted `picode-act` dogfood before integration.
- **Browser preview follow-through:** upstream proposal/package emitter and a
  Browser panel remain open; the generic conversation renderer is shipped.
- **Tutorial video refresh still needs selective execution.** Named surface
  fingerprints and tutorial mappings are shipped. The remaining work is to
  capture and render only stale stills/tutorials, define cache boundaries, and
  choose manual, scheduled, or both triggers without blocking delivery.
- **Remote modes:** real second-account, container, and public OIDC acceptance
  runs require owner-controlled sudo/infrastructure.

## Next up

1. Merge `feat/workspace-favicon-nextjs` and deploy; hard-reload the sidebar
   so COGNIXSE's cached favicon 404 does not stick.
2. Owner validates one live Inbox reply on this TUI (paste fallback), then
   respawns a TUI to prove the receiver channel end to end.
3. Inspect the live store's historical Inbox rows; close or repair only the
   exact stale rows before any new live test.
4. Implement selective docs-video capture/render from the shipped surface
   profiles, add cache boundaries, and choose the maintenance trigger policy
   (manual, scheduled or both).
5. Continue the browser-preview emitter/panel and ADR-0054 real-page dogfood.
6. Run the owner-controlled remote-mode acceptance matrix, then decide the
   SaaS track.
7. Build Providers Models/Activity only after confirming their current study
   still matches Pi's provider data.

## Known debts / open questions

- The primary `picode` checkout has an unfinished merge in progress. Do not
  `git switch` or `git clean -fdx` there.
- A Pi TUI compact at ~100.8k tokens aborted an in-flight `read` and did not
  resume the turn (session `w nodeterm compact`, 2026-09-04). ADR-0061 says
  early compact is `turn_end`; this looked mid-turn. Separate from the
  favicon branch.
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
- Tutorial videos can be visually stale without failing CI. Missing/tampered
  MP4s and changed compositions or referenced stills still fail. The current
  manual audit flags `create-agent` (agents/create/agent), `automate-it`
  (automations/inbox), and `take-it-anywhere` (work/agent/inbox) because those
  surface inputs changed since their last capture. Until selective refresh
  ships, run `make docs-videos-fresh` during deliberate docs maintenance.
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

- **2026-09-04 — workspace cards find Next.js `icon.svg`.** Favicon lookup
  now checks `icon.svg`/`png`/`ico` (svg still beats png/ico) and scans
  `apps/<name>/{public,app,src/app}` after the static dirs, with `apps/web`
  first. Cognixse (`apps/web/app/icon.svg`) is the reproducing case. Branch
  `feat/workspace-favicon-nextjs`, not deployed. visual-review: N/A (lookup
  only; sidebar already wears `/api/workspaces/{id}/favicon`).
- **2026-09-04 — docs captures gained per-surface fingerprints.** Public
  screenshots and tutorial stills now map to named desktop/mobile profiles;
  local screen imports, shared shell/style/fixture inputs and selected data
  producers determine freshness, while tests and unrelated handlers do not.
  The strict manual video audit identifies the affected tutorial and profile;
  CI remains an integrity-only, non-rendering floor. A decision table covers
  unchanged, test-only, shared-style, desktop-only, mobile-only and
  cross-pipeline changes. Public screenshots were regenerated and the old
  global UI-tree helper was removed. Selective capture/render, caching and a
  maintenance trigger remain explicitly in flight. Full `make ci` passed;
  `make docs-videos-fresh` deliberately reports the eight stale profiles named
  under Known debts. visual-review: PASS (all three final screenshots read;
  text and controls are legible, with no clipping, overlay or dead state).

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
