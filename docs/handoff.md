# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** local `main` is clean and contains the runtime-favicon merge
(`dd70c8d0`), WebSocket writer fix (`98d6bcad`), git asset preview merge
(`72f9e395`), proposed release-cadence ADR-0064, and the pi-compact command
routing merge (`0098081c`). The tree preserves ADRs 0060–0063 and the
maintainer runbook; no official release cadence or date is set. Local commits
have not been pushed to the remote.

**Deployment:** the installed service is active on `0.1.0+65be297`
(`release: false`), which carries the runtime badge favicon refinement.
`GET /api/health` returned `status: ok` with boot id `a89dacd8f72395e3`, and
113 PiCode-owned tmux sessions are present (additions only across the
restart). The WebSocket writer fix has remained stable with no further panic
in the observed uptime.

**Quality:** post-integration `make ci` passes, including 468 frontend tests,
54 pi-compact tests, Go/package tests, docs/OpenAPI/llms parity, Vale, and the
embedded build. Generated captures were refreshed and `make docs-check` passed.

**UI evidence:** final deployed desktop, mobile, and menu screenshots were
read: `/tmp/picode-final-deployed-desktop.png`,
`/tmp/picode-final-deployed-mobile.png`, and
`/tmp/picode-final-deployed-menu.png`. Runtime rows and official favicon
badges are legible; the menu stayed inside the viewport and
`window.__picodeOverlayAudit()` returned `ok: true`; Escape closed it. The
attached browser QA passed with no new console/page/network error.
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
  CLI is never promoted to an Agent. CLI badges wear the runtime's official
  favicon (first working link wins), rendered bare like workspace favicons;
  the compact text mark is only an asset-load fallback.
- Public docs use VitePress, generated OpenAPI, Vale, committed screenshots,
  and integrity-checked tutorial videos.
- Public release mechanics are tag-driven. The cadence study, proposed
  ADR-0064 and maintainer checklist are documented, but no calendar-triggered
  release or Preview lane is active.

### ADR-0061 compaction policy package (`pi-compact`)

Shipped, deployed, then amended after the first real compaction (owner
directive: **no defaults, ever**). The checkout currently has no
`.pi/compact.json`, so the package remains dormant until an explicit dogfood
configuration is created:

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
3. Configure and re-dogfood `pi-compact`, or explicitly leave it dormant.
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

- **2026-09-04 — public guides for pi-roles and pi-inbox.** Both missing
  guides landed in `www/guide/` using the compact template: extension-not-core
  positioning up front, where-it-runs and config-scope tables (no machine
  layer in either), commands/routing tables, canonical schema link (roles),
  loopback POST mechanics (inbox), and a "how you know it worked" check.
  Sidebar gains "Model roles" and "Inbox tools for pi"; the Packages guide
  now lists all four packages. Docs-only; no deploy.


- **2026-09-04 — pi-compact documented as an extension, not core.** The
  public guide leads with the positioning (optional package, dormant until
  configured, removable without a trace) and gains where-it-runs and
  config-scope tables, a minimal config example with the canonical schema
  link, and a "how you know it worked" JSONL check.
  `docs/architecture.md`'s Compaction policy section — stale since the two
  ADR-0061 amendments — now matches HEAD and names the Go server's only
  involvement: exporting `PI_COMPACT_AGENT`. Package README and the
  Packages guide entry carry the same framing. Docs-only; no deploy.

- **2026-09-04 — runtime badge favicons refined (owner report), deployed as
  `0.1.0+65be297`.** The owner's screenshot showed Claude Code still on its
  boxed text mark: `claude.ai`'s favicon hangs/403s as a browser subresource.
  CLI favicons now load from a preference-ordered list of official assets
  (Claude: Anthropic's own icon first; Codex: `openai.com` then OpenAI's CDN
  touch icon) and a loaded favicon renders bare — the box is reserved for the
  text-mark fallback. Verified on a scratch daemon with seeded wrapper
  runtimes: all four CLI badges show real favicons on desktop rows, the tab
  strip and mobile rows; the shell fallback stays boxed; 468 frontend tests
  pass; menu contained and closes on Escape; overlay audit ok; deploy kept
  every tmux session. visual-review: PASS.
- **2026-09-04 — final handoff reconciled after integration.** The local
  checkout has no uncommitted product configuration, so `pi-compact` remains
  dormant by the accepted ADR-0061 policy; its schema and source comment now
  state that behavior consistently. Deployment and release-cadence status
  remain unchanged and local commits are still unpushed.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
