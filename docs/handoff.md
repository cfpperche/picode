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
surface (ADR-0063). Local commits have not been pushed to the remote.

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

## In flight

- No implementation or deployment step from the sidebar merge remains.
- Real CLI dogfood was intentionally not run. The historical Inbox `[Teste 3]`
  and `mobile-6bf740` rows still need deliberate reconciliation before a new
  live question is filed.
- ADR-0054 `picode-act` still needs real model-emitted dogfood before merge.
  The Browser preview emitter/panel remains open.
- Second-account, container, public-OIDC, and other remote-mode acceptance
  runs require owner-controlled infrastructure.

## Next up

1. Review the current local main and decide when to push/promote it; the
   What’s New merge and deploy are complete locally.
2. Inspect the exact historical Inbox rows before any real TUI reply test.
3. Run the owner-controlled remote-mode acceptance matrix.
4. Continue the Browser preview panel and ADR-0054 dogfood.
5. Decide whether selective docs-video capture/render should be scheduled;
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

- **2026-09-04 — What’s New release highlights merged and deployed (ADR-0063).**
  Resolved the ADR-number collision with the already accepted terminal CLI
  presence ADR-0062, merged the responsive desktop/mobile surface, regenerated
  the public app screenshots, rebuilt/restarted the installed service, and
  verified `/api/health` plus `/api/version` on `0.1.0`. Post-merge `make ci`
  passed; visual review of the dialog states remained green.

- **2026-09-04 — merged and deployed compact supervision rows and CLI
  presence (`9bfd1f01`), then verified the newer local main (`66bee74f`,
  `0.1.0+66bee74`).** Resolved the overlap with the Inbox/TUI and compaction
  work, added ADR-0062 to the decision index, ran the complete CI/docs gates,
  restarted the installed service, and verified health, version, current tmux
  fleet, desktop/mobile UI, menu containment, reload, and Escape close.
  visual-review: PASS (screenshots read; overlay audit ok; no clipping,
  unreadable controls, double scroll or dead hover).
- **2026-09-04 — workspace favicon and docs-surface parity work landed.** The
  workspace list now advertises favicon availability and generated screenshot/
  tutorial inputs use named surface fingerprints. Full CI and docs parity
  passed; older detail is archived.
- **2026-09-04 — sidebar scrollbar hides until hover/focus; deployed to
  the installed service as `29183241`.** The
  sidebar's `.side-section` thumb is `scrollbar-color: transparent` at
  rest and fades in over 180ms on `#sidebar:hover` or `:focus-within`
  (all five tabs share the one scroll container); `::-webkit-scrollbar`
  fallback covers engines that ignore the standard property. The gutter
  stays reserved (`scrollbar-width: thin`), so reveal is a pure fade
  with no layout shift. Verified live on a seeded dev instance:
  computed thumb color per state (rest `rgba(0,0,0,0)` / hover 45% /
  focus 45%), transition interpolated mid-flight, `clientWidth` 243 in
  every state, overlay audit ok. visual-review: UNVERIFIED for the
  thumb pixels — a forced-red control proved Chromium CDP screenshots
  never paint scrollbars in any state, so the pixel check is
  mechanically impossible in this harness; surface screenshots (read)
  confirm no layout shift, no clipping and unchanged chrome.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
