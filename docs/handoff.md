# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** local `main` is clean, with the runtime-favicon merge
`dd70c8d0`, its generated-capture refresh, and this deployment handoff in
history. The resulting tree preserves the Inbox-to-TUI receiver (ADR-0060), pi-compact (ADR-0061),
compact supervision rows, authoritative terminal CLI presence (ADR-0062),
the official runtime favicon fix, and the What’s New release surface
(ADR-0063). Release cadence research, proposed ADR-0064 and the maintainer
runbook are also in the tree; no official cadence or date is set. Local
commits have not been pushed to the remote.

**Deployment:** this session's `make deploy` completed successfully after the
runtime-favicon merge. The installed service is active and serves semver
`0.1.0`; `GET /api/health` returned `status: ok` with boot id
`3f5e79f70cb92a96`, and `GET /api/version` confirmed source build
`0.1.0+eac0924` (`release: false`). The existing fleet remained available
through the restart; 104 PiCode-owned tmux sessions are present now.

**Quality:** post-merge `make ci` passes, including 465 frontend tests,
Go/package tests, docs/OpenAPI/llms parity, Vale, and the embedded build.
The generated captures were refreshed and `make docs-check` passed.

**UI evidence:** post-deploy desktop, mobile, and menu screenshots were read:
`/tmp/picode-postmerge-desktop.png`, `/tmp/picode-postmerge-mobile.png`, and
`/tmp/picode-postmerge-menu.png`. Runtime rows and official favicon badges are
legible; the menu stayed inside the viewport and
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
  now 404s for newer Google accounts. 59 package tests; `make ci` green.

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

- **2026-09-04 — release cadence process documented (proposed ADR-0064).**
  Added a benchmark study covering VS Code, Linear, Zed, Cursor and Go; a
  proposed source/dogfood versus Stable release-lane decision; and a
  maintainer runbook covering scope freeze, quality gates, tagging, artifact
  verification, observation and hotfixes. Linked the documents from the
  contributor, README, benchmark and ADR indexes. No official cadence, date,
  Preview channel or scheduled workflow was activated. `make ci` passed.

- **2026-09-04 — runtime CLI favicon/alignment fix merged and deployed
  (`dd70c8d0`, docs refresh `eac09242`, `0.1.0+eac0924`).** Claude Code,
  Codex, Grok, and Pi badges now prefer their official runtime favicon,
  retain a text fallback on asset failure, and align identity icons with the
  first text line. Generated docs captures were refreshed; `make ci` and
  `make docs-check` passed. The installed service is active and healthy with
  104 PiCode-owned tmux sessions present. Post-deploy desktop/mobile/menu
  screenshots were read; visual-review: PASS and overlay audit: ok.
- **2026-09-04 — git graph diff cards preview binary assets (merged from
  `feat/git-asset-preview`).** `gitgraph.Blob` plus
  `GET /api/{agents|terminals|workspaces}/{id}/git/blob` serve one blob at
  one revision (hex hash or `HEAD`, in-tree path, 32 MB cap, blob-MIME
  allowlist); `FileDiff` gained `status` (added/deleted/renamed derived from
  mode lines; untracked files marked added). New `GitAssetPreview` replaces
  the "Binary file — no text diff." line in UncommittedDetail, CommitDetail
  and WorkingDiff: images before|after with lightbox, video/audio/pdf/3D on
  the changed side, honest fallback for the rest. Verified live against a
  scratch repo (modified/deleted/untracked PNG, committed video, audio, pdf,
  zip fallback, text diff intact, shallow clone where every file is an
  addition, WorkingDiff via workspace owner) with screenshots read;
  overlayAudit ok; lightbox closes. Gates green (fmt/vet/test/test-js/build).
  Not pixel-exercised: the missing-blob image error line (API 404 path unit-
  tested; needs a corrupted/shallowed parent to appear) and the first-commit
  deleted-asset note. visual-review: PASS.
- **2026-09-04 — Next.js workspace favicons deployed (`0.1.0+4e9cedb`).**
  Lookup is project-agnostic (`icon.svg` and `apps/<name>/app`, not a named
  repo). Fast-forwarded `feat/workspace-favicon-nextjs` onto `main` after
  merging sidebar `hasFavicon` so the list and the image endpoint share one
  finder. Live: COGNIXSE `hasFavicon: true`, favicon 200 SVG 24 673 bytes.
  Reload once if a card still shows a folder. visual-review: N/A (no JSX).
- **2026-09-04 — workspace cards find Next.js `icon.svg`.** Favicon lookup
  now checks `icon.svg`/`png`/`ico` (svg still beats png/ico) and scans
  `apps/<name>/{public,app,src/app}` after the static dirs, with `apps/web`
  first. Cognixse (`apps/web/app/icon.svg`) is the reproducing case.
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
