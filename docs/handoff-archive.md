# Handoff archive

Moved off `docs/handoff.md` when it exceeded ~150 lines. Newest living
state is always `docs/handoff.md`. Do not treat this file as current.

## Agent CLIs v1 validation (archived 2026-09-05 during v2)

- **2026-09-04 — Agent CLIs v1.** Moved Terminal status out of Preferences;
  added launch defaults/overrides, terminal control, setup checks and honest
  applied/pending diagnostics. Pi remains the only managed agent runtime.
  CI passed (486 frontend tests), along with focused CLI/Store/server race
  checks. Merged and deployed as `0.1.0+567280b`, boot `6b6567e8816a5596`,
  with all 129 tmux sessions/pane PIDs and legacy switches intact.
  Installed Pi 0.85.0, Claude Code 2.1.261, Codex 0.153.2 and Grok 1.0.13
  opened their native TUIs without model turns or bypassing native trust.
  Pi/Claude initial activity was observed; Codex/Grok presence was not
  presented as activity. Manual Claude adoption preserved its pane PID.
  Empty/blocked/error, desktop/mobile, light/dark, menus and confirmations
  passed screenshot review and settled overlay audits. Captures:
  `docs/screenshots/cli-v1-*.png`, `var/screenshots/cli-v1-deployed-*.png`.
  All disposable QA processes/data were cleaned up. The version-specific
  working/approval/settled matrix remained open.

## Recent activity (archived 2026-09-04 during Agent CLIs v1)

- **2026-09-04 — Docker v3 resources, health and supervised maintenance.**
  Added reviewed project actions, shared locks, durable steps, opt-in monitoring
  and Inbox review links. Real disposable Engine QA and Pi tool loading passed;
  visual-review: PASS (screenshots read; overlay audits ok). CI passed, including
  480 frontend tests and Docker/Store race tests; deployed as `0.1.0+7029b04`,
  boot `00c5867e6f1aacb0`. Installed assets/binary and real inventory verified.
  Real groups were bidwar (12), cognixse (11), hull (9), pgtenant (2); cards
  filled the main canvas at 1920px. Resources/Health were checked; monitoring
  remained off. All 124 sessions present at deployment survived. The earlier
  125-session baseline included `terminal-8-e0d7f1`, deleted via a Store event
  at 00:25:14 UTC before service restart at 00:25:44 UTC. No replacement was
  created. Engine QA covered project lifecycle, image/network removal,
  stopped-consumer protection and honest health samples. Pi 0.85.0 loaded all
  11 Sysadmin tools and exercised read APIs without a model turn. Disposable
  Engine resources, QA servers and browser sessions were cleaned up.

## Recent activity (archived 2026-09-04 during Docker v3)

- **2026-09-04 — Docker width fix deployed as `0.1.0+ea756cc`.** Removed
  the fixed 960px limit and checked wide/narrow/mobile viewports. The v3 plan
  sequences resources, health and assisted maintenance after v2.
  `make ci` passed. visual-review: PASS (screenshots read, overlay audit ok).

## Recent activity (archived 2026-09-04 during Docker width correction)

- **2026-09-04 — Docker groups deployed as `0.1.0+900ac98` (ADR-0066).** Native
  disclosures stay in the app; exact Compose labels, standalone fallback,
  state summaries, saved folds and search work together. Long phone rows
  wrap. V2 separates detected groups from registered Compose deployments.
  `make ci` passed. visual-review: PASS (screenshots read, overlay audit ok).
  Isolated test servers, browsers and the feature worktree were cleaned up.

## Recent activity (archived 2026-09-04 during Docker grouping)

- **2026-09-04 — terminal favicons merged and deployed as `0.1.0+cf9aafc`.**
  The owner's `v0.1.0+a2e377e` screenshot still had the leftover chip because
  the first worktree preview was overwritten by the Docker deploy and the
  branch had not been merged. Rebased onto Docker `main`, merged, deployed.
  Live sidebar: Claude/Codex/Grok/Pi fill the 22px slot with no plate.
  visual-review: PASS (`/tmp/picode-term-favicon-deployed.png` read).

- **2026-09-04 — Docker App deployed; pi-sysadmin package ready (ADR-0065).**
  Extended existing Apps primitives and mobile navigation; added bounded
  Engine API operations, idempotent background jobs, verified outcomes and
  durable history. Real disposable-container start/stop/restart passed.
  Event bursts initially starved detail reads; container-state event filters
  and a serialized refresh queue fixed the observed failure, with regression
  coverage. Plain-text logs, empty/blocked/error and mobile confirmations
  passed screenshot review. `make ci` passed. visual-review: PASS.

## Recent activity (archived 2026-09-04 during Docker delivery)

- **2026-09-04 — terminal identity survives restarts; favicon cards fixed
  (owner report); deployed as `0.1.0+9ffaa39`.** The owner's reload showed
  every wrapped terminal back at "Shell session" and Codex still boxed:
  wrapper presence was memory-only, and the loaded OpenAI `.ico` paints its
  own opaque white card. Reconciliation now revives CLI presence from the
  pane's process tree (exact command, or a `/proc` walk through wrapper
  shells and interpreters, PID + start-token validated, dropped when the CLI
  exits), and CLI badges lead with the same transparent SVG marks the
  provider faces use. End-to-end proof on a scratch daemon (revive after
  restart, drop on CLI exit), then live: every wrapped terminal regained its
  identity after the deploy and stayed stable across polls; badges render
  bare with no white card. visual-review: PASS (live sidebar screenshot
  read; overlay audit ok).
- **2026-09-04 — public guides for pi-roles and pi-inbox.** Both missing
  guides landed in `www/guide/` using the compact template: extension-not-core
  positioning up front, where-it-runs and config-scope tables (no machine
  layer in either), commands/routing tables, canonical schema link (roles),
  loopback POST mechanics (inbox), and a "how you know it worked" check.
  Sidebar gains "Model roles" and "Inbox tools for pi"; the Packages guide
  now lists all four packages. Docs-only; no deploy. Follow-up: the
  Checklist guide was aligned to the same template with an honest core row
  (tool and rule in the package; sidebar, cards and Level in PiCode).

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

## Recent activity (archived after the final local integration)

- **2026-09-04 — pi-compact command routing was corrected.** The second
  ADR-0061 amendment registers `/compact-edit|model|on|off`, leaves bare
  `/compact` to Pi, and was merged as `0098081c`; 54 package tests passed.
- **2026-09-04 — proposed release cadence was documented (ADR-0064).** The
  benchmark, maintainer runbook and source-versus-stable proposal were linked
  without activating a schedule or release date.
- **2026-09-04 — runtime favicons and compact-row alignment landed.** Claude,
  Codex, Grok and Pi badges prefer official assets with a text fallback; the
  work was merged as `dd70c8d0` and its generated captures were refreshed.
- **2026-09-04 — binary git-asset previews landed (`72f9e395`).** The API and
  diff-card preview support passed live scratch-repository checks; the missing
  parent/error visual paths remain documented as untested.
- **2026-09-04 — terminal bridge writes were serialized.** Output, pings and
  error frames now share one per-connection writer; targeted and race tests
  passed, and the deployed service remained free of the observed panic.
- **2026-09-04 — Next.js workspace favicon lookup and surface fingerprints
  were delivered.** The project-agnostic finder covers nested app icons and
  the docs capture pipeline records named desktop/mobile inputs.

## Recent activity (archived 2026-09-04 after runtime-favicon deployment)

- **2026-09-04 — ADR-0061 amended after dogfood: no defaults, safe trigger,
  self-healing chain.** The first real compaction exposed two defects: the
  early trigger fired from `turn_end` and `ctx.compact()`'s leading `abort()`
  killed the active run ("This operation was aborted"; agent never
  continued), and the auto chain led with gemini-2.5-flash, which now 404s
  for newer Google accounts, so compaction silently fell back to Pi's
  summarizer. Fixes: dormant-until-configured semantics (owner directive —
  no defaults, ever), trigger moved to `agent_settled` + `isIdle()` guard,
  per-link chain retry with gemini-3.6-flash → Haiku. ADR-0061 amended in
  place; guide/README/CHANGELOG updated; package tests 59/59; `make ci`
  green (docs shots refreshed after the whats-new UI merge). Merged to
  `main` and deployed: `0.1.0+18e6788`, health ok (new boot), service
  active, 103 tmux sessions intact.

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

## Recent activity (archived after docs surface fingerprints)

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

## Recent activity (archived after README refactor)

- **2026-09-04 — systemd stop hang merged and deployed (`6cf705dd`, `0.1.0+6cf705d`).**
  Fast-forwarded `feat/fix-systemd-stop` onto local `main`. First `make deploy`
  still SIGKILLed at 30s because the *outgoing* daemon re-exec'd the new
  binary. The next restart of that new process: SIGTERM at 11:07:14 →
  `shutting down` → `Stopped` at 11:07:19, no `newer on disk — reloading`,
  no `stop-sigterm`, no SIGKILL. All 57 tmux sessions unchanged. Health 200,
  installed binary matches `bin/picode`. Worktree removed.

## Recent activity (archived after systemd stop diagnosis)

- **2026-09-04 — Hosted CI split by workload and merged (PR #3).** Frontend is
  built/tested once on Ubuntu and its 14 MB output feeds the embedded build;
  public docs and the Linux/macOS/Windows Go boundary run in parallel without
  repeated Node installs. The fail-safe path classifier and final gate are
  decision-table tested, superseded runs cancel, and parity now checks the
  committed OpenAPI/llms.txt before regeneration. Full PR/main runs
  `33878224835`/`33878695007` passed in 3m54s/3m52s; metadata-only runs
  `33879212363`/`33879325513` ran only classifier + final gate and passed in
  32s/27s. No product deployment was needed.
- **2026-09-04 — Passive extension UI no longer kills Inbox replies.** Local,
  unpushed `259eb50a` ignores fire-and-forget status/widget/title/notify/editor
  updates during a reply burst while retaining fail-closed behavior for real
  select/confirm/input/editor dialogs. Tests and docs media are in the commit;
  the installed service reports that version, but it remains unpushed.
- **2026-09-04 — ADR-0059 integrated and deployed.** Local `main` fast-forwarded
  to gated feature commit `81ac872b`; `make deploy` now serves
  `0.1.0+81ac872`. The original 50 and immediate 54 tmux identities, both exact
  agent session paths, and all 12 terminals survived unchanged. Health,
  migration, embedded assets, and a live Chromium smoke passed. The recurring
  systemd stop timeout is recorded as debt; real-Pi Inbox dogfood was not run.
  visual-review: PASS (pre-merge edge-state screenshots read; post-deploy app
  read, no page errors, first-party requests healthy, overlayAudit ok).

## Recent activity (archived after ADR-0059 deployment)

- **2026-09-04 — Hosted CI restored across Ubuntu, macOS, and Windows.** PR #2
  merged as `b7c63d34`; the follow-up handoff commit is `dd98084c`. Linux and
  macOS run the daemon suite; native Windows compiles every package/test with
  race instrumentation and exercises the tray/browser-host boundary. The
  service was deployed at `0.1.0+b7c63d3` before ADR-0059 superseded it.
- **2026-09-03 — ADR-0059 transient RPC burst implemented and gated.** Inbox
  replies borrow the exact asking session for one private RPC turn, prove
  durable JSONL delivery, stream a terminal-only lifecycle, restore the same
  tmux pane, and recover/cancel without exposing chat mode. `make ci`, focused
  race tests, generated docs/media parity, and the desktop/mobile visual card
  passed on the final feature commit.
- **2026-09-03 — Manual Pi TUI terminal status completed (ADR-0056).** The
  opt-in scoped wrapper reports idle, working, needs-you, prompt return,
  completion, and interruption without touching the user's Pi configuration;
  owner acceptance and live desktop/mobile dogfood passed.
- **2026-09-03 — Three tutorial videos shipped.** The docs harness captures
  deterministic desktop/mobile stills, renders the HyperFrames compositions,
  and parity-checks the published videos in CI.

## Recent activity (archived after ADR-0059)

- **2026-09-03 — tmux cwd test no longer depends on scheduler timing.** The
  test polls both initial cwd and a later shell `cd`; it passed ten consecutive
  package runs and the full gate.
- **2026-09-03 — main advanced to `6609ac9e`.** Docs harness/OpenAPI/Vale,
  guest-terminal interception and HTTPS fixes, terminal icon state, and Codex
  lifecycle/interrupt hooks landed while ADR-0059 remained isolated.

## Recent activity (archived 2026-09-03 before ADR-0059)

- **2026-09-03 — L4 closed: the refused checklist card leads with its
  refusal line** (`fix/checklist-refusal-detail`, merged + deployed).
  `checklistRefusal()` handles both wire shapes (live: text in
  `result.content[].text`, detail is JSON — never leaked; replay: text
  flattened into `detail`), the card shows a red refusal line +
  "Refused" in the head, healthy cards untouched. Row-by-row tests
  (453 JS green). Visual review on a scratch daemon with a hand-built
  session JSONL: refused card reads Refused + red line + attempted
  list; healthy card unchanged (checklist-card-*.png; overlayAudit
  ok; card 5/5). Scratch torn down by PID (environ match), per policy.

- **2026-09-03 — dashboard counting scope clarified (read-only, no code).**
  Owner asked whether the dashboard counts only PiCode agent sessions or
  also terminal `pi` runs. Traced: `/api/sessions/stats` →
  `session.StatsRoot(session.Root())` scans **every** JSONL under the
  shared `~/.pi/agent/sessions` (`internal/session/stats.go`,
  `internal/server/session_stats.go`), so terminal/tmux pi sessions count
  in Spend/Activity/Sessions/Daily/models/workspaces/Tokens/Tools/
  Reliability/Top Sessions — the only escape is a pi run pointed at a
  non-default sessions dir. The FLEET tile is separate: `fleetStats`
  counts only store-registered agents (workspaces + free agents); an
  unmanaged terminal pi never appears there. Recorded as a scope note on
  the ADR-0042 line in *Current state*.

- **2026-09-03 — adversarial review of `packages/pi-checklist` + the
  staleness fixes, live-dogfooded** (`fix/checklist-staleness`):
  probed the gate, the reminder loop (confirmed against pi 0.84.4
  source that follow-up turns do not re-emit `before_agent_start`, so
  the 3-reminder cap holds), TLS loopback canonicalization
  (`0x7f000001` → `127.0.0.1` via WHATWG — solid), agent-id traversal
  (blocked). Found and fixed the two real ones (stale row on fresh
  session; stale items on absent/blocked). Live dogfood on a scratch
  daemon: gate refusal → "No checklist", + New → silence, resume →
  republish. visual-review: PASS (checklist-sidebar-*.png read;
  overlayAudit ok; card 5/5). **Near-miss, owned:** the scratch cleanup
  ran the exact banned pattern (`pkill -f picode`) that the ADR-0057
  incident below warns about — it coincided with the service already
  down (restored by that session at 14:11:46, three minutes after my
  pkill), so no harm landed, but the lesson is now policy for this
  agent too: cleanup targets PIDs, never name patterns.

- **2026-09-03 — Devices list no longer accumulates "This machine"
  duplicates (ADR-0049 amendment, `feat/devices-hygiene`).** Owner report:
  the Devices view piled up "This machine · Linux" rows, worst after
  release deploys. Root cause: every browser-like loopback visit without
  a cookie minted a fresh 90-day browser session (`internal/auth` Wrap),
  and every headless agent-browser QA profile is such a visit (12 rows in
  one afternoon on the owner's DB); `PruneSessions` had no caller and
  `ListSessions` did not even filter expired rows. Fix: the mint reuses
  the newest live session with the same label by rotating its secret
  (presence asked first so an active browser keeps its cookie; a 30 s
  burst window lets a first-visit request race reuse instead of mint —
  the race was caught live in QA as two "Windows" rows 50 ms apart);
  headless UAs label "Headless browser"; expired rows stop listing;
  a daily sweep prunes revoked/expired after 7 days; Devices gains a
  batch **Forget offline (N)** that skips token sessions. Verified live
  on a scratch instance: first visit = exactly one row, cookie survives
  a server restart (deploy = zero new rows), batch forget cleans 8 seeds
  in one click. visual-review: PASS (qa-devices-before/confirm/after/final
  screenshots read; overlayAudit ok; blocked Reconnecting state also
  captured). Concurrency trade-off recorded in the ADR. Gates: make ci
  green (Go + 564 JS tests). Branch ready to merge + `make deploy`.

- **2026-09-03 — Tool live previews in the conversation (ADR-0057,
  `feat/browser-preview-core`).** Owner approved the browser-preview plan.
  Shipped the core half of the ADR: a tool-agnostic `details.preview`
  contract (`lib/toolPreview.js`) rendered inline in the tool pill
  (frame + title/URL caption, click → lightbox); the reducer and the
  desktop `handleEvent` now consume `tool_execution_update` (previously
  dropped on the floor), the final result carries the persisted frame so
  replay renders it, and `replay.js` constructs the same item shape.
  Decision table tested (434 JS tests green: valid/invalid frames, unknown
  ids, replace-don't-accumulate, end-with/end-without preview, replay).
  Visual-review PASS on a scratch instance (WS init-script fixture + a
  hand-built session JSONL): live mid-stream frame in the pill
  (`qa-live-mid.png`), replayed frames + captions
  (`qa-replay-frames.png`), lightbox (`qa-lightbox.png`, overlayAudit
  ok). **Incident:** my `pkill -f picode` during scratch cleanup also
  killed the owner's installed service (8445) — restored in ~5 min
  (`systemctl --user start picode`, health OK); scratch cleanup must
  target PIDs, never name patterns. Next: the package-side emitter
  (PR 2 — upstream [pi-agent-browser-native#157](https://github.com/fitchmultz/pi-agent-browser-native/issues/157)
  opened with the `details.preview` proposal; PR or companion package
  pending their answer) and the Browser panel surface on `#/agent/<id>`.

- **2026-09-03 — Guest terminal status, tier 1 of ADR-0056**
  (`feat/guest-term-state`): a coding CLI inside a PiCode terminal can
  now report its own lifecycle — `POST /api/terminals/{id}/state`
  (`working` / `needs-you` / `idle`, auth-gated like every route),
  republished as ephemeral `terminal.state` on the ADR-0048 feed and
  carried by the terminal views for reconciliation. Correlation is
  configuration-free: every terminal PiCode opens gets
  `PICODE_TERM_ID` + `PICODE_TERM_URL` in its tmux session env at
  creation (`new-session -e`), so hook processes inherit it. Chips:
  sidebar row (spinner / accent "Needs you"), terminal tab (green /
  accent dot), mobile TermRow; no chip = no signal, and a silent
  `working` expires after 30 min (`StartTermStateSweep`) so a dead
  sensor can never leave a spinner. Registry is in-memory by design —
  a restart is honest "no signal" (verified live). ADR-0056 accepted
  with the owner's two-tier split (agents deferred; scraping refused).
  Guide: `www/guide/terminal-status.md` (Claude hooks + Codex notify
  + the `picode-hook` helper). **Wiring is now one click**
  (`feat/term-wiring`): Preferences → Terminal status installs the
  reporter at `<data>/picode-hook` and merges/strips exactly the
  marked hook entries in `~/.claude/settings.json` (idempotent, user
  content preserved, corrupt JSON refused with a visible reason);
  Codex stays manual by design (no stdlib TOML) and shows its state
  with the guide link. Verified live: enable → hooks + executable
  reporter → the real script reported a terminal's `working` through
  the API; disable → only our entries removed; overlayAudit ok after
  aligning the rows (stretch, not center — two-line labels).
  Windows gap: the reporter is POSIX sh + curl — fine where the daemon
  runs (WSL/Linux), unresolved for Windows-hosted daemons.
  visual-review: PASS (qa-wiring-before2 / qa-wiring-on /
  qa-wiring-final.png read, card 5/5). Verified earlier in the
  scratch instance: env inheritance inside the pane shell,
  working→needs-you flip patched the open browser via the feed
  without reload, desktop chip + tab dot and mobile chip screenshots
  read; overlayAudit ok; plain-shell terminal stays chipless.
  visual-review: PASS (qa-term-working-clean / qa-term-needsyou /
  qa-term-final / qa-term-mobile.png, card 5/5).

- **2026-09-03 — Providers view v2 shipped** (`feat/providers-v2`,
  ADR-0058, study `docs/benchmarks/2026-09-03-providers-view-v2.md`).
  Quota moved onto the roster in three honest states (live / age-labelled /
  a word saying which kind of nothing), served from a process cache so a
  page load makes zero vendor calls; `StartUsageRefresh` warms only the
  active, non-paused slot of each meterable provider, sequentially, every
  5 min. Identity (email + normalised plan) is read from the vendor and
  written back to the vault row — `default_claude_max_5x` renders "Max 5x"
  and `billing_type` is no longer shown at all. **Verify** runs
  `pi auth check --provider X --json --no-refresh` (pi ships the primitive;
  no test completion, no token). **Pause** keeps a credential but leaves
  play. Sign out names its blast radius from `agents.provider` /
  `automations.provider`. A provider supplied only by an env var is now
  signed in, named by the variable, with no Sign out — measured on pi
  v0.84.4 that `GROQ_API_KEY` alone answers `ready/api_key`, so those rows
  were invisible while every agent could use them. Dogfooded on a scratch
  instance with the real vault: 6 meterable accounts, anthropic 5h/7d bars,
  kimi's "Rate limited." shown as state, OpenRouter credits as an amount.
  overlayAudit ok, no clipping, dark and 390px checked, Verify/Check/search
  clicked for real. **Not built, and next:** the Models tab (price and
  context from `models-store.json`, which `pi --list-models` omits, plus a
  picker writing `enabledModels`) and the Activity tab (per-provider spend
  and burn-rate projection from our own session JSONL). Both are scoped in
  the study.
- **2026-09-03 — docs-harness benchmark study (plan presented).**
  Studied documentation benchmarks for a public-docs harness — Diátaxis
  IA, Scalar (MIT API reference, Vue), Mintlify/Fern (llms.txt),
  Vale (MIT prose linter, ships in Mintlify CI), Remotion license
  (free ≤3 people only — default refused), HyperFrames (installed,
  0.8.27) as the default video engine, D2/Mermaid for diagrams-as-code.
  Study: docs/benchmarks/2026-09-03-docs-harness.md. Plan: theme with
  app tokens, own-capture docs-shots pipeline over a seeded fixture
  daemon, route-walking openapi.json + Scalar page, llms.txt, Vale gate
  in make ci, 3 HyperFrames tutorials. Owner directive folded in:
  **full parity** — the harness captures its own screenshots
  (`docs/screenshots/` stays agent evidence, never user docs);
  `make docs-check` re-captures and diffs in CI so UI drift without
  regenerated images fails; video compositions declare their surfaces
  and are flagged stale the same way. ADR pending owner approval.

- **2026-09-03 — adversarial review of the feed migration + one fix**
  (`fix/git-watch-workspace-scope`): `gitDirs` attached the workspace id
  to an agent's own-workPath event, so the worktree's branch could
  poison `ws.git` (the fallback pills source) until the workspace path
  changed again; the id now rides only the directory it describes.
  Accepted trade-offs documented: ephemeral events are lossy by design
  (debt), package scans grow with workspace count (code comment).

- **2026-09-03 — Providers view v2: benchmark study, no code**
  ([`docs/benchmarks/2026-09-03-providers-view-v2.md`](benchmarks/2026-09-03-providers-view-v2.md)).
  Three web sweeps: agent IDEs/CLIs (Kilo four-section IA + source badges,
  Zed `ApiKeySource` origin display, Roo profiles + lock-across-modes,
  Cursor Verify, Raycast Verify + console link + key icon at point of use),
  multi-account switchers and quota monitors (cc-switch v3.13 renders quota
  **inline on the card**, claude-swap's proactive 90 % auto-switch with
  cooldown/hysteresis and per-terminal account scoping, ccusage burn-rate
  projection, CCUM's official-limit trust layer, CodexBar, oh-my-pi's
  round-robin credentials), and credential dashboards (OpenRouter
  Prioritized/Fallback BYOK, Cloudflare `cf-aig-byok-alias` over a
  `default`, Vercel Test Key with a raw-response badge, Anthropic graduated
  expiry mail, Stripe roll-with-grace, Zapier dependent count + blast
  radius). Confirmed against pi v0.84.4 that `auth.json` is still
  `Record<string, Credential>` — **native multi-account has not landed**,
  and `pi auth check --provider X --json --no-refresh` is the Verify
  primitive we never wired. Proposal is three tabs (Accounts / Models /
  Activity) with quota inline in three honest states (live / stale / —),
  identity from `oauth/profile`, credential-source badge, dependent count
  before Sign out, Pause vs Sign out, and 7-day spend from
  `stats.byProvider[]`. **Owner's call before any ADR:** per-agent
  credential pinning vs proactive auto-switch, both of which move
  ADR-0013's single-active-slot line. No code changed.

- **2026-09-03 — Degrau 2 shipped: inbox replies switch a TUI agent to
  chat mode** (`feat/inbox-reply-switch` + `fix/form-interactive-
  normalize` + `fix/open-terminal-goto`). The respond form on an
  interactive agent confirms the trade, parks the reply
  (`RespondAndPark`), switches the agent to managed (TUI ends — user
  consented), and the delivery loop drains into the same thread
  (ADR-0053). The action returns `ActionResult.Goto` ("agentchat:<id>")
  and the shell undocks the terminal, landing on the chat watching the
  answer. Open terminal (Degrau 1) navigates out of the inbox the same
  way (`Goto: "agent:<id>"` → docked TUI). Verified live end to end
  (qa-switch free agent): confirm dialog, mode → managed, task
  `delivered`, item responded. ADR-0037 amendment records the decision
  + the send-keys benchmark research (claude-squad uses guarded
  send-keys, marked experimental; PiCode keeps it parked). A gate
  regression (normalizeView dropped the form's interactive flag) was
  caught in QA and fixed forward.

- **2026-09-03 — workspace install refused + Packages tabs**
  (`fix/packages-local-trust-tabs`): `pi install -l … --no-approve`
  answered "Project is not trusted. Use --approve"; measured on 0.84.4:
  `--no-approve` distrusts the project for the run and blocks local
  config writes even with trust.json trusting the folder; `install -l`
  works with no flag, `remove -l` in an untrusted folder needs
  `--approve`, and `--approve` never writes trust.json. `MutateArgs` /
  `UpdateArgs` pass `--approve` for the project scope (the click is the
  approval). UI: Installed |
  Marketplace tabs, installed packages as the same cards (`pkgName`).
- **2026-09-03 — package update checks ride the change feed**
  (`feat/feed-packages-updates`, phase 4 — polling→feed migration
  complete). Fleet-wide scan ticker publishes `packages.updates` on
  change; the browser applies events and polls only as fallback.

- **2026-09-03 — CI's git-guard self-test unbroken (ubuntu).** The
  self-test's throwaway repo relied on the invoking clone's
  `init.defaultBranch`: CI runners have none, so `git init` started on
  `master`, the pre-commit guard rightly refused the init commit, the
  repo stayed unborn, and two assertions failed in cascade. One-line
  fix (`git init -b main`, c503b9da), reproduced locally under null git
  config — the old script under CI conditions fails exactly like run
  33769826099, the fixed one passes. macOS/Windows redness is older,
  separate debt (see Known debts).

- **2026-09-03 — Guest TUI agent state, benchmarked** (docs-only,
  `feat/guest-agent-state`): the owner runs Claude Code, Codex, Grok,
  Antigravity, opencode in PiCode terminals and wants the sidebar's
  spinner + "needs you" for them. Study
  `docs/benchmarks/2026-09-03-guest-tui-agent-state.md` (fetched live
  from ACP docs/registry, Claude Code hooks/statusline/terminal-config,
  Codex config/app-server/non-interactive docs, opencode server/ACP,
  Grok CLI, Vibe Kanban, Crystal, Conductor): four integration levels;
  nobody credible scrapes pixels as the primary source; Claude hooks
  can POST to HTTP (`Notification` types include `agent_needs_input`),
  Codex has `notify` + an official app-server, opencode has a server
  API — and the ACP registry already lists **pi (pi-acp)**.
  Deliverable: **ADR-0056 (proposed)** — per-tool sensors → ADR-0048
  feed with pi's state vocabulary + honest `unknown`; scraping refused
  as primary; ACP/control deferred to a future ADR; ADR-0003's letter
  (no vendored SDKs) untouched. Awaiting the owner's decision on the
  ADR.
- **2026-09-03 — manager could not remove a path package**
  (`fix/pkg-remove-relative`): owner installed `pi-checklist` for the
  machine and Remove failed. Cause: pi stores path sources relative to
  the settings dir; the daemon ran `pi remove <relative>` from its own
  cwd. Fix: `pipkg.AbsPathSource(source, settingsDir)` before remove and
  update (`packageSettingsDir` picks `~/.pi/agent` or `<ws>/.pi`), and
  `existingInstallPath` resolves the same way. Verified against the
  scratch home's real relative entry.
- **2026-09-03 — the public Pages site is green again.** The red
  `pages.yml` runs since 09-02 16:30 died in `make docs`:
  `www/guide/remote-server.md` opened a code span at a line break
  (`` `sudo picode users add ``), VitePress left it unclosed, and the
  dangling backtick let `<your login> <your user>` into the Vue
  template as raw HTML — "Element is missing end tag" (18:103),
  reproduced byte-for-byte locally. The cure was already on local
  `main` (a85ff516 + ec3a515c: one code span per line) but **55
  commits sat unpushed** since 09-02, so the site was frozen at the
  09-02 15:04 deploy. Pushed `7ffe8a9f..fe421595`; run 33769826235
  green end to end; live page verified serving the fixed guide.
  Earlier red runs (08-29 → 09-02) were the dead-link gate, fixed by
  PR #1. Lesson: run `make docs` before pushing `www/**` — docs that
  land straight on `main` have no PR gate for the Pages build.

- **2026-09-03 — internal checklist, ADR-0055** (`feat/pi-checklist`):
  researched Tachyon's checklist gate/reminder/sidebar line and pi
  0.84's extension API; built `packages/pi-checklist` (tool + gate on the
  first change per task + `always` reminder capped at 3 + POST to the
  daemon + TUI render), `agents.checklist` level + `PICODE_CHECKLIST`
  spawn env (read-only → never), `agent_checklists` + routes + feed
  event, `lib/checklist.js` line projection, sidebar/mobile lines, chat
  card, settings chip; guide `checklist.md`. Owner's decisions: level
  per agent, opt-in package, sidebar + card, hold the contract.
  Measured live on glm-5.3-flash: managed RPC path (plan → read → edit
  → 2/2), the gate (edit refused → checklist → edit ok), the TUI in tmux.
  Found: `pi -p` hangs on any blocked tool_call (pi's bug, minimal repro
  in the ADR); PiCode never uses `-p` for agents. Providers were flaky
  that morning (xai at capacity, anthropic out of usage, gemini 5/min).

- **2026-09-02 — git pills and the file tree ride the change feed**
  (`feat/feed-git-events`, phase 3 of the polling→feed plan). Fleet git
  watcher publishes `git.updated`; sidebar patches in place, tree
  reloads itself. visual-review: PASS (qa-git-live.png read;
  overlayAudit ok; e2e: commit cleared the badge, new file re-raised it
  — zero fleet refetches, one tree reload per event).

- **2026-09-02 — Benchmark study: live browser preview in chat / side
  panel** (`docs/study-browser-preview`). The owner asked to render
  `agent_browser` work live in the conversation or a side panel, and
  whether that needs a PiCode package system or a pi package. Researched
  how Cursor ("screenshots and actions in the chat, as well as the
  browser window itself either in a separate window or an inline pane"),
  Devin (session Browser tab + Progress unified log + take-over), Manus
  (Cloud Browser real-time view + Take Over), Operator (*inference*),
  Antigravity (browser subagent + action-video artifacts + allow/denylist),
  Browserbase + browser-use (embeddable live-view iframes, "URL is a
  credential"), OpenHands (rrweb recordings) and the local `agent-browser`
  (`stream enable` WebSocket, `record`, `artifactVerification`) solve it.
  Convergence: activity cards in chat + live surface with take-over.
  Key gap found: PiCode's web reducer never consumes
  `tool_execution_update` and tool pills render raw JSON — no image path.
  Answer: no new package system; add a generic `details.preview`
  rendering contract to core (never the string `agent_browser`) + a
  Browser panel fed by it; v2 = auth-gated WS proxy for `stream enable`.
  ADR is the next step (see *Next up* 0).
- **2026-09-02 — the inbox refusal to a TUI agent offers Open terminal**
  (`fix/inbox-open-terminal`). Replying to an agent-sourced item whose
  agent runs in a TUI is refused by design (ADR-0037/0006), but the
  refusal was a dead end. The item's view now shows an Open terminal
  action when the agent is interactive (same `h.AgentDeliverable` gate
  the reply uses); firing it starts the TUI server-side via
  `apps.Host.OpenAgentTerminal`, wired from `Deps.openAgentTUI`
  (handleAgentOpen's body extracted, sentinel errors keep the HTTP
  mapping identical). Works from the desktop and mobile inbox.
  Deployed; verified live on the owner's item + a QA item:
  action renders only when interactive, click starts/confirms the
  tmux session (mode → interactive), detail closes by design after
  acting. Follow-through (same day): the action now returns
  `ActionResult.Goto` ("agent:<id>") and the shell's AppSurface hands
  it to the host via `onGoto` → **leaves the inbox** and lands on the
  agent's tab with the TUI docked (owner decision: acting from the
  inbox must not dead-end there; mobile has no terminal surface and
  ignores the goto). visual-review: PASS (qa7 + qa8-landed.png,
  card 5/5). **Parked (owner's call, from the send-keys benchmark
  research):** Degrau 2 — consented mode-switch delivery (queue the
  reply + switch interactive→managed; ADR-0053 keeps the thread; on
  switch, the agent tab undocks the terminal and flips to chat; the
  TUI reopens later on the same session; trigger: owner hits the
  refusal again in dogfood); Degrau 3 — guarded send-keys delivery,
  parked with ADR-0002's rejection (claude-squad precedent noted:
  SendPrompt = keys + 100ms + TapEnter, autoyes experimental, per-CLI
  content sniffing). Follow-up (same day): the sidebar spinner meant
  "streaming OR waiting" — after the reply switch the agent sits in
  managed+waiting while the ask_human waits in the inbox, and the row
  spun "working" for what was really a needs-you state. The spinner
  now means streaming only; the inbox badge carries needs-you (a
  needs-you pill on the agent row is parked as polish).

- **2026-09-02 — MCP live status rides the change feed**
  (`feat/feed-mcp-events`, phase 1 of the polling→feed plan). Fleet-wide
  watcher publishes `mcp.updated`; the Mcps panel follows events and
  polls only as the feed-down fallback. visual-review: PASS
  (qa-mcps-empty.png + qa-mcps-live.png read; overlayAudit ok; e2e:
  live-file flip → exactly 1 request, badge Idle→Live→Failed).

- **2026-09-02 — Work head = Inbox head on the phone**
  (`fix/mobile-head-parity`): `.m-screen-head` takes the Inbox numbers
  (12/8 padding, hairline, margin 0); `.m-inbox .ft-head` matches and
  cancels the desktop 880px `margin-top: 6px`; the 32px segment rule is
  scoped to `.m-agent-state`. Both heads measure 57px, controls 36px.

- **2026-09-02 — LooksWorking anchors to pi's spinner frames**
  (`fix/looks-working-spinner-frames`). The working check grepped the
  pane tail for the substring "working": an idle agent whose last
  reply mentioned the word (this project's own docs/logs do, constantly)
  spun in the sidebar and showed "Working in the terminal" in the
  composer — the owner caught mobile doing it at 20:38 while pi sat at
  its prompt. The indicator is now what pi actually renders: a pane
  line whose first non-space rune is one of the braille spinner frames
  (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏ — pi 0.84.4 DEFAULT_FRAMES; re-check on pi upgrades).
  Immune to prose and to a renamed working message; a custom indicator
  without braille degrades to a benign false-negative. Decision table
  with real captured tails (working / idle / the false-positive prose)
  pins it. Deployed (PID 412915); verified live: mobile working →
  detected via ⠏ while its pane is full of the word, agente-auto idle →
  not listed. Probes one open design question (unfixed): inbox replies
  to interactive agents are refused by design (ADR-0037/0006 — the TUI
  picks up no follow-ups); options range from an "Open terminal" action
  on the refusal note to send-keys delivery — owner to decide if/when
  it itches.

- **2026-09-02 — black strip under terminals** (`fix/term-viewport-strip`):
  owner spotted a near-black band at the bottom of every terminal. Cause:
  xterm 6 sets the theme background on `.xterm-scrollable-element`, not
  `.xterm-viewport`, which keeps xterm.css's `#000`; the row-remainder
  showed it. Fix: `.xterm .xterm-viewport { background-color: transparent }`
  in `app.css` (surface carries the theme colour). Pixel-verified in the
  scratch shell.
- **2026-09-02 — Termux layout for the mobile keys** (`feat/mobile-keys-termux`):
  owner sent Termux screenshots — "vamos fazer o nosso igual". `KeyBar`
  is Termux's 2×7 grid (`ROWS`), flat cells on `--bg-base`,
  `grid-template-columns: repeat(7, minmax(0, 1fr))`; ⌨/× dropped
  (tap the terminal for the keyboard; header icon toggles). Sticky
  modifiers, viewport lift and hardware heuristic unchanged.
- **2026-09-02 — the chat shows when a TUI agent is working**
  (`fix/tui-working-in-chat`). The chat's event socket exists only for
  managed agents, so an interactive agent read as idle there no matter
  what pi was doing (owner: sent a message in the TUI, switched to the
  chat, no working sign). The server already scraped panes and
  published `agent.tui` (ADR-0048) — sidebar/dashboard spun on it, the
  chat didn't. When the selected agent is interactive and its pane is
  working, the composer now shows a "Working in the terminal" row with
  an Open button that docks the TUI; no fake streaming, no Stop. UI
  only; the server pieces shipped with ADR-0048 (watch tick 3s).
  Deployed and verified live on agente-auto: send-keys → row appears
  within one tick with the spinner, Open docks the TUI mid-work, row
  clears when the pane's working line ends; overlayAudit ok.
  visual-review: PASS (qa4-row-live.png + qa4-open-docked.png,
  card 5/5).

- **2026-09-02 — Mobile terminal keys to the benchmark (ADR-0044
  amendment)** (`feat/mobile-terminal-keys`): library search found
  nothing terminal-aware (simple-keyboard & co. are full QWERTYs), so
  the bar matches Termux/Blink/terminal-web: `lib/termSticky.js`
  (createSticky: arm/apply/applyKey/subscribe, control bytes, modified
  arrows, 5 s expiry; tested), wired on the ShellTerm entry
  (`entry.sticky`, filters `term.onData`); `KeyBar` is one scrollable
  row (`KEYS`), Ctrl/Alt light up (`aria-pressed`); `Terminal.jsx` sizes
  the screen to `visualViewport` (lift above the iOS keyboard, refit)
  and hides the row when focus comes with no viewport shrink (hardware
  keyboard). iPhone verification is the owner's: keys never open the
  keyboard, ⌨ does, Ctrl then c interrupts, bar above the keyboard.
- **2026-09-02 — Mobile key bar** (`fix/mobile-keybar-focus`): owner:
  every key opened the iPhone keyboard. `sendKey` refocused xterm
  unconditionally; now it refocuses only if the terminal host already
  held the focus, and a ⌨ key (`onType`) is the deliberate way to open
  the keyboard. `KeyBar` regrouped (escapes/chords · arrows/symbols),
  44px keys, close at the end of row two. Not browser-verified here
  (owner tests on the phone).
- **2026-09-02 — the composer's "+ New" starts a fresh session for real**
  (`fix/new-session-fresh-start`). The button cleared the pointer and
  then lost it three ways: the restart spawned in `wk.Path` (free
  agents: the `ws_free` sentinel → "That folder doesn't exist" — the
  toast the owner screenshotted), ADR-0053 adoption re-adopted the
  abandoned thread, and the list's newest-fallback re-selected it.
  Fixes: `restartSameMode` resolves `store.AgentCwd` (+MkdirAll);
  `store.SealPendingAgentSessions` closes the adoption window — and
  moves attribution to the path row, because a plain UPDATE collided
  with the sibling path row on UNIQUE (agent_id, session_path)
  (probed live on agente-auto's DB); the list fallback now surfaces
  only a pointer the loop itself healed; the web pane clears
  optimistically. Decision table pinned (resume→list, New stopped→
  fresh, next spawn mints fresh ≠ abandoned id, free agent + TUI open,
  free agent + chat running). Deployed and verified live on
  agente-auto: chat clears, no toast, old sessions still listed,
  tmux respawned in the agent's own folder; overlayAudit ok.
  visual-review: PASS (qa2-before/qa2-after-stable.png, card 5/5).

- **2026-09-02 — Mobile Inbox without a title** (`fix/mobile-inbox-no-title`):
  owner: the Inbox was the only main tab with a header. CSS hides the
  AppSurface icon + `.ft-title` under `.m-inbox`; the filters stay as
  the sticky head; the section keeps its aria-label.
- **2026-09-02 — Mobile shell in Safari's own tab** (`fix/mobile-safari-headers`):
  owner's iPhone screenshots — heads clipped under the status bar,
  keys button over the TUI. Sticky heads lose the negative margin and
  the negative `top`; the screen hands its top padding to the head via
  `:has()` (`mobile.css`); the terminal's keys toggle is a header
  button (`.m-keys-btn`), `.m-fab` removed. Measured in Chromium at
  390×480: head border box at 0 at rest and when scrolled.
- **2026-09-02 — Content-Security-Policy (ADR-0052 amendment)**
  (`feat/csp`): `internal/server/csp.go` — the app shell's inline theme
  bootstrap is allowed by sha256 hash computed from the served
  index.html (no nonce); `script-src 'self' 'wasm-unsafe-eval' <hash>`,
  inline styles allowed (React), `img-src https:` (unpkg icons),
  `connect-src` names the request host's ws/wss; assets carry no policy
  (excalidraw's worker uses `new Function`). `/pair` and gateway pages:
  `PageCSP` (no scripts, no framing). `securityHeaders` wraps the UI
  handler. Test skips without a built UI; run after `make build`.
- **2026-09-02 — `make desktop-restart` guardrail after the tray/VM outage.**
  Swapping the Windows exes, this session killed the tray and relaunched it
  with `picode-desktop --tray &` from a WSL tool shell; the process died with
  the shell, the keepalive died with it, and the 60s WSL idle timeout
  reclaimed the VM — server down, open tmux sessions lost. Fix: the swap is
  now one audited command, `scripts/desktop-swap.sh` (kill → copy → re-register
  host → `schtasks /run /tn PiCodeDesktop` → verify tasklist), wired as
  `make desktop-restart` and listed in AGENTS.md's commands table. The script
  header and the make help state the rule: never background a Windows exe
  from WSL. Ran once for real after the fix: exes swapped, tray up, bootId
  unchanged (no VM bounce).
- **2026-09-02 — Chrome extension Track C, the actuator (ADR-0054)**
  (`feat/ext-track-c`). Send with "Let the agent act on this page" →
  `[browser-act]` prompt intro → settle watcher parses the last
  ```picode-act block → `act_batches` (migration 021) → panel polls
  `GET /api/extension/act/next` through the native host (origin-gated
  claim, blocked stays pending) → per-origin grant → visible
  step-by-step execution (`chrome.scripting`, one action per injection,
  highlight + native setters) → outcomes POST back as one more watched
  turn; 3-round cap, Stop, 10-min expiry. Parser/lifecycle/routes
  table-tested; panel states captured (`ext-act-grant/-acting/-actdone`).
  (Actuator ADR numbered 0054 — 0053 went to session isolation.)

- **2026-09-02 — Session isolation follow-through** (`fix/session-followthrough`):
  owner filed one bug with three faces on a freshly created agent: the bar
  opened with another agent's context/spend/cache, the picker said "No
  sessions yet" after it had chatted, and opening its TUI started a new
  session (then a chat send killed the TUI's work). Root causes: the picker
  never saw ADR-0040's private dir (so the lazy `session_path` backfill
  never fired), every run-mode switch minted a competing `--session-id`,
  and the desktop `/status` fetch omitted `?agent=` (server falls back to
  the workspace's first agent). Fix = ADR-0053: adopt-at-spawn + picker
  union + agent-scoped bar, table-tested; verified in the browser against a
  scratch instance with a two-agent fixture (mobile's bar bare, alpha's bar
  showing its own $23.88). Also fixed here: a pre-existing cross-line code
  span in `www/guide/remote-server.md` broke the Pages build on `main`
  (`make docs` failed before and after my changes until that span was
  closed — reproduced on main first, fix committed in this branch).

- **2026-09-02 — Chat follows an automation's turn** (`fix/chat-follows-automation-run`):
  owner: run "Running" but the chat showed nothing. Three causes in the
  desktop: the panel effect gated on `selected` (the *workspace*), so a
  free agent never got its socket; a closed panel for the same agent
  blocked reconnecting (onclose now nulls `panelRef`, the effect checks
  `readyState`); and a turn nobody typed here (automation, another tab)
  had no prompt bubble — `foreignTurnRef` reloads the session file on
  snapshot(streaming) and on settle so the thread mirrors the file, in
  order. Diagnosed on the scratch with a raw WebSocket and a patched
  `window.WebSocket` (the app opened no socket at all).
- **2026-09-02 — Terminal open vs a run in flight** (`fix/automation-run-guard`):
  the owner clicked the agent's terminal icon during a message run; the
  open path's `Runtime.Stop` killed the managed process and the run
  hung. `deps.automationRunOn` (ManagedAgent.Observed) → 409 on the four
  interactive-open paths; `runWatch.exited(expected)` fails the run as
  `reasonStopped` unless the run itself asked (`letGo`). Unit test.
- **2026-09-02 — Message runs deliver (ADR-0045 amendment)**
  (`fix/automation-message-delivers`): owner ran a message automation,
  got "Done" and an untouched agent. `messageRun` starts an idle agent
  (`startManaged`, its own session), sends a prompt (SendTurn makes it a
  follow-up when busy), observes with `runWatch` and stops it on settle;
  `keepAlive` for an already-running agent (cost cap aborts the turn
  only); decision table: pi needed unless the agent runs, busy when
  another run observes the agent; `ManagedAgent.Observed()`. E2E
  `TestAutomationMessageRunEndToEnd` against the fake pi.
- **2026-09-02 — Automation detail facts** (`fix/automation-detail-facts`):
  owner: a notify URL saved but nothing on the detail. Facts list gained
  Messages/Runs in (agent + workspace via `workspaceOfAgent`), Model (or
  "pi's default"), Notifies (host + copy), limits; `whenLine(a, agents)`
  names the agent (List needed the `agents` prop — a blank root until it
  had it); the secret box's Done is a primary button.
- **2026-09-02 — Automation editor layout** (`fix/automation-editor-layout`):
  owner: "componentes se amontoando, péssima UI/UX". The what-it-does
  block is a labeled `.auto-grid` (Workspace → Agent, or Workspace →
  Provider/Model/Thinking); message mode lists the chosen workspace's
  agents via `agentsOf` (free agents under "No workspace") with their
  model, and an existing automation's workspace is derived from its
  target agent; `lib/providers.js usableProviders` filters to signed-in
  providers (keeping a selected one, marked) and `ConfigFields` uses it
  too, so New agent matches.
- **2026-09-02 — Automations notify a channel (ADR-0045 amendment)**
  (`feat/automation-notify`): `notify_url` on automations (migration
  020, validated http(s) in the store), `automate.BuildNotify` (Slack
  shape, clipped summary), `runner.notifyOut` in
  `internal/server/automations_notify.go` (background, one retry on
  network/5xx, `automation.notify` event), hooked into `settled` and the
  failure path of `finish`; editor field + "Notifies" tag; guide section.
- **2026-09-02 — Automation webhooks through the gateway (ADR-0045
  amendment)** (`feat/gateway-webhook`): `POST /-/hook/<user>/<id>` on the
  gateway (no identity, per-peer limit, member check, Authorization
  passed through, cookies dropped); the automation view carries
  `webhookUrl` computed from the daemon's situation (origin / public URL
  / gateway form, via `os/user.Current`); the detail page shows it; guide
  recipes for GitHub Actions, Sentry and cron.
- **2026-09-02 — Track D, public access (ADR-0052)** (`feat/public-access`):
  gateway identity chain (tailnet whois → signed cookie → login page),
  `plainListen` + `trustedProxies` (last XFF hop from a trusted CIDR
  only), Google OIDC (discovery, PKCE, nonce, JWKS RS256) and GitHub
  OAuth (`<login>@github`) in `internal/gateway/oidc.go`, signed session
  cookie (`session.go`), per-peer limiters on `/-/auth/*` and `POST
  /pair`, `/-/` routes and a login page, extra security headers;
  `picode gateway oidc set|unset`, `--plain` (alias `--insecure-listen`),
  secrets in `gateway.secret.json`; the SPA shows **Sign in** on a 401
  that carries `login`. D.2: `ContainerSteps` + `install.ContainerUnitFile`
  (nspawn, private users, limits, host networking), `--container`/`--remove`.
  Tests: full login round trip against a fake OIDC provider, claim
  checks (aud/exp/nonce/iss/verified/unknown login), forged cookie,
  untrusted XFF, logout, limiter, GitHub spelling, unit text, steps.
- **2026-09-02 — ADR-0053 deployed to the owner's service; the "not
  fixed" report was a stale binary.** The installed
  `~/.local/bin/picode` predated the 17:13–17:14 fixes (16:49 build, 0
  hits for `ResolvePendingAgentSession`) while the on-disk UI was
  fresh — so the chat still read "No sessions yet" for an agent whose
  sessions were visible in the TUI, and the status bar fell back to
  the workspace's first agent (grok's $23.88 / 100% cached shown on
  the brand-new `mobile` agent). `make deploy`; no code changes.
  Verified live on `mobile-6bf740`: the picker returns the agent's two
  private-dir sessions and adopts the TUI's pending `6903ca7b…` as
  current; `GET /status?agent=` returns that agent's own bar ($0.02,
  4.3% of 1M) instead of grok's. visual-review: UNVERIFIED (API-level
  checks only).

## Recent activity (archived 2026-09-02)

- **2026-09-02 — Devices footer spacing + centering.** List rows
  unchanged. `.devs-foot` adds 16px spacer + divider under the last
  device; Pair / Copy link and the extension note are centred. Empty
  state untouched. visual-review: PASS (devices-foot.png + overlayAudit
  ok; card 5/5).
- **2026-09-02 — `picode gateway uninstall [--purge]`**: the owner asked
  how to remove the gateway completely, not just disable it. Stops and
  removes the system unit; `--purge` deletes `/etc/picode` and the
  shared binary unless a member unit still runs it. Members are never
  touched (their own `picode uninstall`).
- **2026-09-02 — Track C, shared tailnet box (ADR-0051)**
  (`feat/shared-server`): `internal/gateway` — config (`/etc/picode/
  gateway.json`, login → Linux user), identity via `tailscale whois`
  (60 s cache, non-tailnet refused), backend from the member's own
  `server.json`/`token`, reverse proxy with header hygiene and
  `FlushInterval: -1`, first-visit auto-pair (mint with the daemon's
  token, redirect to `/pair`), branded 403/503/502 pages. CLI: `picode
  gateway [install|status] [--insecure-listen]`, `picode users`,
  `provision --shared` (`MemberSteps`: account, linger, binary, env
  drop-in, member's own pass via `runuser`, loopback health). Daemon
  side: `PICODE_AUTH_MODE` and `PICODE_PUBLIC_URL` env fallbacks;
  `share.Diagnose` treats a proxied daemon as trusted; no trust listener
  when insecure. Scratch seams: `PICODE_GATEWAY_FAKE_IDENTITY` and
  `PICODE_GATEWAY_FAKE_HOMES`, only with a loopback plain listener.
  Not yet done: the two-real-accounts acceptance run (owner's sudo).
- **2026-09-02 — GitHub Pages frozen since 2026-08-29**
  (`fix/pages-dead-links`). VitePress 1.6.4 fails the build on a bare
  `https://localhost:8445` in `www/guide/getting-started.md` (dead-link
  check). Four Pages runs failed in ~20s; the live site stayed on the
  last green artifact (2026-08-26), so `/guide/mcp` 404'd even though
  `mcp.md` is on `origin/main`. Fix: wrap the URL as inline code,
  `ignoreDeadLinks` for localhost / 127.0.0.1, `make docs` in `make ci`
  (ubuntu job) so Pages is not the first gate. Merged as PR #1.

- **2026-09-02 — provision reads systemd from any shell**
  (`fix/provision-session-env`): from a PiCode terminal (no
  `XDG_RUNTIME_DIR`), the service row said "present but not enabled" and
  its fix "still fails" — the check's `systemctl --user` could not reach
  the manager while the fix (install.Run, which fills the session env)
  could. provision's `run`/`output` now carry `install.SessionEnv()`; a
  check that still cannot reach systemd says so (blocked) instead of
  claiming the unit is off. `tailscale cert` errors now carry the action:
  the account error → enable HTTPS Certificates in the admin console;
  access denied → `tailscale set --operator`.
- **2026-09-02 — Track B.2, the Tailscale certificate (ADR-0050
  amendment)** (`feat/tailscale-cert`): `internal/tlsutil/tailscale.go`
  issues `tailscale-cert.pem`/`-key.pem` for the MagicDNS name via
  `tailscale cert`, `LiveConfig` picks the leaf by the requested name,
  `KeepTailscaleCert` renews daily (logs a failure once, with tailscale's
  hint); `provision` row `tailnet-cert`; `share.Diagnose` lists the name
  as a `trusted` target picked first after a public URL, the trust page
  is skipped for it (drawer and `pairLinks`). On this WSL box issuance
  needs `sudo tailscale set --operator=goat` once — not run by the agent.
- **2026-09-02 — Bind keeps loopback (ADR-0050 amendment)**
  (`fix/bind-keeps-loopback`): the owner chose *Tailnet only* and the
  tab hung: the new bind on the same port could not bind-new-first
  (0.0.0.0 overlaps), the loop reverted, and the UI had already moved to
  the tailnet address (unreachable from the Windows browser, and with
  no cookie there). Now an outside bind also listens on 127.0.0.1; host
  changes shut the old listener, bind the new, restore on failure; the
  tab keeps a local origin; the options read "Tailnet and this machine".
- **2026-09-02 — Track B.1, a PiCode on a tailnet server (ADR-0050)**
  (`feat/tailnet-server`): `server.host` and `server.public_url` are
  settings with routes (`PUT /api/server/host` rebinds, `/public-url` is
  advisory) and a Reach-this-server block in Preferences → Server that
  suggests the tailnet name/IP; `server.json` gained `bind` and
  `publicUrl`; `picode install --env` writes a systemd drop-in; `picode
  update` verifies `SHA256SUMS`; `provision --dry-run` gained pi/tailnet/
  reach rows (no `doctor` verb — same engine); `remote.json` +
  `extension-install --server --token --ca` point the native host at
  another machine, `pi-inbox` reads `PICODE_URL`/`PICODE_TOKEN`. Bug fixed
  on the way: the install token is now a real token session, so the
  extension shows on Devices. Guide: `www/guide/remote-server.md`.
  Deferred to B.2: `tailscale cert` by SNI (see ADR-0050).
- **2026-09-02 — Sidebar no longer duplicates a just-created terminal**
  (`feat/zai-reset-and-terminal-dup`). Owner screenshot: two identical
  "Terminal 4" cards after clicking "+". Root cause: `store.CreateTerminalIn`
  publishes `terminal.created` to the change feed (deduplicated by
  `applyFleet`) before `handleCreateTerminal` finishes `ensureShell`
  (spawns the tmux session) and responds — so the SSE subscriber's
  `setTerminals(next.terminals)` in `App.jsx` almost always lands *before*
  the POST response, and `createTerminal`'s own `setTerminals((cur) =>
  [...cur, page])` then appended the same terminal a second time,
  unconditionally (unlike `openTermTab`, which already replaces-or-appends
  by id). Fixed by skipping the append when the id is already present.
  Verified on a scratch instance (`:8471`) via `agent-browser`: created two
  terminals back to back, sidebar showed exactly one card each
  (`after-fix.png`). `make fmt-check vet test test-js build` all green.
  z.ai's own "Reset Quota" ask from the same session is a separate open
  question — see *Known debts* above.

- **2026-09-02 — /pair got PiCode's own look** (`fix/pair-page-branding`):
  the confirm and error pages were an unstyled system page with no
  branding — owner: "pessima UXUI, sem branding do picode". Rewrote
  `pairPage`/`pairPageTemplate` in `internal/server/auth.go` as a
  dark-first card matching the app's tokens (bg/border/accent, the π
  wordmark, "The browser is a door, not a cage." footer), with
  `prefers-color-scheme: light` for a light OS. The heading and button
  now name the device (`presence.Label`): "Pair this iPhone", "Pair this
  Browser" for an unrecognized UA. No React here — this loads before any
  cookie exists, so the CSS is inlined rather than shared with app.css.
- **2026-09-02 — /pair confirms on POST** (`fix/pair-confirm-post`): the
  iPhone camera prefetched the pairing link, spending the one-time code
  before Safari opened it ("This link was already used"). GET /pair now
  renders a one-tap page and only the POST pairs; a browser that already
  holds a session goes straight in; error copy points at Devices.
- **2026-09-02 — One pairing QR, tailnet first** (`fix/pair-one-qr`):
  owner scanned two different QRs (Devices and Open on phone) and the LAN
  address did not reach the phone. Devices → Pair a device now opens the
  phone drawer (`openPairDrawer`, one QR with the LAN/Tailnet choice);
  `share.Diagnose` prefers the official tailnet address (LAN needs the
  same Wi-Fi, the Windows firewall rule and WSL mirrored networking);
  each target carries a `note` saying what it needs; the "address" check
  no longer claims the phone can reach it. Trust listener: 8470 first,
  else a random port — the firewall rule only opens 8445/8470.
- **2026-09-02 — Devices is one surface** (`fix/devices-one-surface`):
  owner asked why the user menu and Preferences both had "Devices". The
  presence page (`#/devices`) now lists paired sessions with liveness
  joined by session id (`PingSession`); Preferences → Server keeps the
  pairing rule and the token (`AccessSection`). `DevicesSection` removed.
- **2026-09-02 — Pairing link reachable from the phone** (`fix/pair-url-reachable`):
  a request on loopback built `https://localhost:8445/pair?…`, which Safari
  on the phone cannot open. `pairLinks` now takes the share report's
  reachable address and, with mkcert, wraps the QR in the trust page.
- **2026-09-01 — ADR-0049 authentication, Track A of the remote-modes
  roadmap** (`feat/auth-core`): gate, modes, pairing, install token,
  Devices UI, CLI, client updates. Dogfood on a populated DB copy (note:
  the copy carries `server.port`, override it or the scratch collides
  with :8445): loopback auto-pairs (cookie), LAN without cookie → 401 +
  pairing screen, bearer works, foreign Origin and unknown Host → 403,
  pairing link → cookie → app.
- **2026-09-01 — Hotfix: empty sidebar after the feed-debts deploy**
  (`fix/boot-fleet-load`, `1688619`). The pattern-driven swap of
  post-mutation `loadWorkspaces()` → `refreshFleetFallback()` also hit
  the two **boot** loads; with the stream open before boot finished the
  desktop showed "No workspaces yet" (data untouched — the API and the
  events log proved it). Lessons: review every site of a mass
  replacement; dogfood boot paths against a populated database, not the
  scratch instance's empty one.
- **2026-09-01 — ADR-0048 debts closed** (`fix/feed-debts`): tmux
  watcher (`agent.tui`), per-message `agent.usage`, `device.offline`
  ticker, run mode on `agent.status` at all ten start sites, health
  watch idles at 20 s with the feed up, 17 redundant `loadWorkspaces()`
  → `refreshFleetFallback()`. Lesson: chaining `make build` behind a
  `grep` masked a vite failure and a broken commit went in for two
  minutes — verify by exit code, never by grep.
- **2026-09-01** — **Hotfix: mobile shell crashed to a blank screen on
  every Inbox visit**, on branch `fix/appsurface-feed-crash`. Owner:
  "app no web quebrado nao abre .. erro no console". The concurrent
  ADR-0048 change-feed merge (`a7712ee0`) left
  `AppSurface.jsx`'s feed-subscription effect referencing `app.id` —
  `app` was never declared in that component (only the `appId` prop
  exists). On desktop this went unnoticed: a coincidental `id="app"`
  element elsewhere in the page made the browser's legacy named-element-
  as-global-property behavor resolve `app` to that DOM node instead of
  throwing (`app.id` silently evaluated to the string `"app"`, quietly
  breaking the auto-reload-on-touch feature but not crashing). The
  mobile shell has no such element, so the reference genuinely threw
  `ReferenceError: app is not defined` on every render of the Inbox
  (list or item screen) — with no error boundary, React 18 unmounts the
  whole tree, emptying `#root`. Confirmed reproducible on a byte-
  identical isolated build (not a caching artifact) via an
  `--init-script` early error listener (agent-browser's own `console`/
  `errors` capture missed it — too early in the page lifecycle). Fix:
  both references now read `appId`. Verified: Inbox list and item
  screens both mount clean on mobile with zero captured errors.

- **2026-09-01 — ADR-0048 change feed** (`feat/change-feed`): durable
  event log + SSE with replay, push and presence on the feed, desktop and
  mobile patch from it. Scratch dogfood: desktop sidebar showed an agent
  created by curl without reload, Inbox badge rose on `inbox.created`;
  phone showed a new agent with no fleet poll (only health, presence and
  tui-working in 20 s); `curl -H Last-Event-ID` replayed; daemon
  restart → `hello.bootId` changed → reload. Owner pushed back on an
  invalidation-only design; the durable log is the accepted one.
- **2026-09-01** — **Two small UI fixes reported from the phone/desktop**,
  on branch `fix/apps-header-row-spacing`: (1) `AppsGrid` dropped its
  "Apps" section header — the sidebar's icon rail already marks which
  section is open, and the single "Inbox" tile beneath it repeated the
  same word (owner: "informação duplicada"). Follow-up
  (`fix/inbox-done-header`): the owner then pointed at a second
  duplicate — the Inbox's own "DONE 14" section label directly under
  the "Done 14" tab pill (same number, same word). `internal/apps/
  inbox.go`'s `doneView`/`donePane` now build that block with no Title
  at all (`BlockHead` already renders nothing when title/meta/at are
  all empty); the "All" tab's own trailing "Done" block keeps its title
  since there it names one of three real subsets (Needs you/Feed/Done),
  not a restatement of the active tab. Two inbox_test.go assertions
  that used the block's Title to find the Done pane now match on the
  done item's own ID instead. (2) `.app-row` (the shared
  list row `AppSurface.jsx` renders for every app, desktop and mobile)
  dropped its reserved 16px unread-dot gutter, blank on every read row;
  the unread mark is now an inline `●` before the title, the same
  convention the mobile Now screen already used. (3) The mobile swipe's
  `translateX` on `.app-row-actions` was silently replacing that row's
  own `translateY(-50%)`, so the revealed action buttons rendered
  vertically off-row ("fica pela metade") — fixed by combining both
  transforms. Verified on scratch: desktop Apps tab and Inbox list
  (`overlayAudit` n/a, plain layout), mobile swipe with the action
  button's rect centred within 3px of the row's.

- **2026-09-01** — **Mobile phase 3 (ADR-0044 addendum)**, on branch
  `feat/mobile-phase3`: `usePullToRefresh` + `PullScreen` (Now, Work;
  Inbox via `AppSurface.refreshKey`), touch swipe on `AppSurface` rows
  (`app-row-swiped` reveals the hover-only actions), `#/changes/<kind>/<id>`
  → `Changes.jsx` wrapping `UncommittedDetail` (now accepts owner kind
  `workspace`); entry buttons on the agent state row, the terminal header
  and the workspace card when `git.dirty > 0`. QA on scratch with the
  real dirty checkout: file list + expanded patch, synthesized touch
  swipe reveals actions, synthesized pull shows "Release to refresh" and
  refetches. Then: "a página inteira rola, o header mais botões devem
  ficar fixos e somente as listas devem rolar" — `.m-screen-head` (Work's
  segmented control + New, More's title) and `.m-inbox .ft-head`
  (Active/Done/All + filter) now use the same negative-margin/sticky-top
  trick `.m-head` already used for pushed screens, pinning them to the
  scroll container's own top edge while the rows beneath slide past
  (`fix/mobile-sticky-headers`). Agent and Terminal screens needed no
  change — their content panes already size via flex `min-height:0` and
  scroll internally, so the outer `.m-screen` never engages there.
  Verified on scratch with seeded overflow (9 workspaces, 13 inbox
  items): `getComputedStyle(...).position === "sticky"` and the header's
  `getBoundingClientRect().top` unchanged at `scrollTop 300`. Owner on
  the phone: pull did not fire on Inbox and the
  stacked split overflowed. `fix/mobile-inbox-subroute`: `AppSurface`
  `paneMode="list"|"detail"` + `onOpenItem`; mobile Inbox = list tab
  (`PullScreen` is the scroller) + pushed `#/inbox/<id>` item screen with
  Back; an action returning to the root pops the screen. Owner: an Inbox
  row needed two taps on the iPhone and swipe never showed — iOS Safari
  turns the first tap into hover when `:hover` changes content (the
  hover-only row actions). `fix/inbox-hover-touch`: the `.app-row` hover
  rules live under `@media (hover: hover)`; touch gets the swipe. Then: a
  left swipe was firing the pull — `usePullToRefresh` now decides the
  axis on the first 10px of movement and stands down for a horizontal
  drag (`fix/pull-axis`). Owner: "destaca o botão remover com a cor de
  perigo… faz um motion nesse swipe, tá muito seco" — the row now
  follows the finger (inline transform during the drag, `--swipe-w` from
  the action count) and snaps open/shut with `--motion-med`; the actions
  slide in from the right edge as 40px buttons, Delete in the danger
  tint; terminal Remove uses the same tint (`feat/swipe-motion`).

- **2026-09-01 — Automations debts closed** (`fix/automations-debts`):
  `RunObserver.OnCost` (sum of `message_end` usage) makes the cost cap
  event-driven and synchronous; `FailStaleRuns` takes a cost resolver
  (`automate.SessionCost`); the server test fake emits
  start / message_end(+usage) / end / settled, and
  `automations_e2e_test.go` runs a real `start` run plus a capped one.
- **2026-09-01 — ADR-0045 v2 `/automate` + templates** (`feat/automations-v2`):
  seven built-in templates, Suggested cards + editor select, `/automate`
  drafting from the current agent (verified on scratch: repo-specific
  prompt, weekdays 09:00, origin line), draft handoff lib. ADR renumbered
  0044 → **0045** after a same-day collision on main (dialogs moved to 0046).
- **2026-09-01** — **Web Push (ADR-0047)**, on branch `feat/mobile-push`
  — phase 2 of the approved mobile plan. `internal/push` in stdlib: VAPID
  keys (`<DataDir>/vapid.json`, 0600), RFC 8291 `aes128gcm` (Appendix A
  vector pinned in the test), sender (TTL/Urgency/Topic, 410 → drop),
  notifier with the decision table (host online → skip; result →
  *finished*; blocking → *actions*; unobserved dialog → *actions*). Hooks:
  `store.OnInboxCreated`, `rpc.Runtime.OnWaiting` (via
  `ManagedAgent.onWaiting`, `hub.Len()==0`), `presence.AnyHostOnline`.
  Store migration 018 + `store/push.go`; `server/push.go` endpoints;
  `main.go` wires one presence registry for both. Client: `sw.js` push +
  notificationclick, `main.jsx` navigate listener (SW now also on
  localhost), `lib/push.js` (+ blocked-reason test), `PushPrefs.jsx` in
  mobile More → Notifications and desktop Preferences → Notifications.
  Also: the responsive-dialogs ADR was renumbered 0045 → **0046** (another
  session's Automations ADR took 0045 in parallel); push is 0047.
  **Real-device dogfood (owner's iPhone, iOS 18.7, Home Screen install):**
  subscribe worked, `Send test` arrived. The first real trigger (blocking
  inbox question, host browser closed) got **400 from Apple** — the
  `Topic` header must be ≤32 URL-safe base64 chars and we sent
  `inbox:<slug>`; the test push only passed because its tag was `test`.
  Fixed: `topicFor` hashes the tag (`fix/push-topic`).

- **2026-09-01** — **Responsive dialogs (ADR-0046)**, on branch
  `feat/responsive-dialogs`. Owner on the phone: "New Agent abre como um
  drawer… o dialog Choose Folder já não abre assim… precisamos mapear
  todos os dialogs e aplicar a mesma regra". Inventory: 16 files with raw
  `@radix-ui/react-dialog`, 1 with `react-alert-dialog`, 1 with `vaul`
  (`CreateForm`'s own fork); 4 anchored popovers left alone. New
  `components/ResponsiveDialog.jsx` (Radix API, `useMedia("(min-width:
  720px)")` → Radix dialog or Vaul sheet; `Alert` set for confirms); one
  import line changed in 14 files; `CreateForm` lost its `useMedia`
  branch; `.dlg.dlg-sheet` in app.css reshapes the whole `.dlg-*` family;
  `lib/dialogPolicy.test.js` walks `web/src` and fails on any raw import
  outside the primitive + `Palette`/`Hotkeys`. Rule written into
  `docs/guidelines.md` and the uiux-review checklist. Owner's phone
  follow-up (`fix/mobile-sheet-touch`): the free-agent sheet sat half off
  the screen — Vaul's `repositionInputs` fought `interactive-widget=
  resizes-content`; the sheet now passes `repositionInputs={false}`. And
  a finger could not scroll a terminal: xterm scrolls its own buffer on
  touch but a tmux pane needs wheel events, so the terminal screen turns
  a vertical drag into one synthetic `wheel` per 16px on `.xterm-screen`
  (`touch-action: none` on the host), which then takes the desktop's own
  path (xterm SGR reports / lib/termWheel). Then: "os diálogos não
  deveriam ativar o teclado automaticamente" — `SheetContent` in
  ResponsiveDialog cancels Radix open-autofocus and blurs a React
  `autoFocus` in a layout effect; desktop unchanged.

- **2026-09-01 — ADR-0046 Automations** (`feat/automations`): store +
  migrations 016/017, `internal/cron`, `internal/automate`, server routes
  + runner, `Automations.jsx`, guide, benchmark study of Devin/Cursor/
  Claude Code/Codex/GitHub. Visual gate: empty, blocked (pi missing),
  editor, detail with secret, list with runs — overlayAudit ok.
- **2026-09-01** — **Mobile shell v2 (ADR-0044)**, on branch
  `feat/mobile-shell`. Owner: "o mobile não precisa refletir todo o shell
  web… observar, responder a chamados, tomar decisões, ações rápidas".
  Benchmarks (Remote Control, Codex mobile, Cursor iOS, AgentWatch,
  Tactic Remote, Nimbalyst, PagerDuty) → `docs/benchmarks/2026-09-01-
  mobile-agent-supervision.md`. Rewrote `web/src/mobile/` around Now /
  Inbox / Agents / More + a pushed agent screen; one server change
  (`agentView` live state) so the home knows who is waiting without a
  socket; agent events as a pure, tested reducer instead of touching the
  desktop's inline handler. Dogfooded on a scratch instance (`:8472`) with
  a **fake pi** on PATH (`scratchpad/dog/fakebin/pi`, speaks the JSONL RPC:
  streams a reply, or asks confirm/select/input on `ASK:*`): empty
  machine → create sheet → two agents asking → answered from Now → agent
  screen with the desktop's ask card → inbox question answered in the
  stacked detail → More → Providers. First captures found: the inbox's
  "Pick an item on the left" and Ctrl+Enter hint on a phone, Providers'
  account rows overlapping at 390px, Start/Stop icons reading as
  checkboxes, no history on the agent screen — fixed (CSS overrides
  scoped to `#m-app`, text Start/Stop, transcript tail replayed via
  `sock.seed`, verified against a real adopted session). Light + dark,
  `overlayAudit` ok on the sheet. Curated: `docs/screenshots/mobile-*.png`.
  Fixed on the way: mobile `!!` toast ReferenceError.
  Owner tested on the phone: "celular não tem view workspaces (agentes e
  terminais), não tem view de terminais… fica ruim agregar tudo em uma tab
  Agents… podemos tirar o header". Follow-up `feat/mobile-work`: header
  removed (QR stays in More); Agents tab → **Work** with the rail's three
  segments (`#/work/workspaces|agents|terminals`, last one remembered in
  `picode-mobile-work`); workspace cards list agents + terminals with
  "+ Agent / + Terminal"; `useFleet` also polls `/api/terminals`; pushed
  `#/term/<id>` screen = `TermSurface` + `KeyBar` (sends byte sequences
  through the `terms` map's socket, refocuses xterm); live terminals in
  Now → Running. QA on scratch with tmux: created a terminal from the
  phone, shell prompt rendered (34 rows), key bar clicks reach the pane,
  Remove stops the session. Owner: a workspace's terminal must not repeat
  in the Terminals segment — it now lists free terminals only
  (`freeTerminals`), like the desktop rail. Third pass (`feat/mobile-
  polish`): zoom lock (meta rewritten on mobile mount + `gesturestart`
  cancel + `touch-action: manipulation`), tab bar hidden on pushed screens
  (`#m-app.is-pushed`), key bar behind a floating `.m-fab` keyboard button
  with a close key, row actions as icons (`IconPlay`/`IconStop`/`IconTrash`,
  new `IconKeyboard` in Icons.jsx).

- **2026-09-01** — **Chrome extension Track B**, on branch `feat/ext-track-b`.
  `make desktop` builds `picode-nmh.exe`; `picode-desktop extension-install`
  registers that console host. Side panel pings `kind=extension`.
  Preferences → Server: connected / not connected + Open guide.
- **2026-09-01** — **Chrome extension Track A (ADR-0043)**, on branch
  `feat/browser-extension`. Sideload `ext/`; native host in `picode` /
  `picode-desktop`; `GET/POST /api/extension/*`. Sensor only (URL, title,
  selection, optional JPEG). Stopped agents Start and send; interactive
  refused; busy maps to follow_up via existing `SendTurn`. visual-review:
  PASS (ext-host / ext-down / ext-none / ext-chrome / ext-form / ext-terminal
  preview states; no SPA overlay). Owner dogfood of Windows Chrome still
  open (Next up).
- **2026-09-01** — **Dashboard v2 (ADR-0042)**, on branch
  `feat/dashboard-v2`. Owner asked for a benchmark sweep and a v2 plan
  after dogfooding v1; the plan was approved as written (four tiles,
  daily chart, six breakdown panels, fingerprint cache, live refresh,
  three refusals). Web research added ccusage, the Claude Code OTel
  metrics + Grafana 25052, the Claude Code Analytics API,
  pi-agent-dashboard, nicknisi/fleet and the 5of10 design rules to the
  study. Backend: `stats.go` reads the assistant message's inline
  `provider`/`model`/`usage`/`stopReason`/`toolCall`s in the same pass,
  the inline provider becomes the running truth for following lines
  (v1's `$6.15` "unknown" bucket was exactly this), `session.Fingerprint`
  + a per-range cache in `session_stats.go` (0.4 ms stat sweep vs 0.9 s
  cold all-time scan on the real 175 MB tree). Frontend: `RankedBars`
  generalises v1's `SpendByProvider`, `lib/barchart.js` + `DailyChart`,
  `TokenBar`, `TopSessions`, `StatTile` gained `deltaText`/`children`,
  `fleetStats` splits working/waiting/idle from the ids App already
  tracks for the sidebar spinner. Dogfooded on a scratch instance
  (`:8471`, copied sessions tree, two seeded workspaces): first capture
  showed a stretched "Spend by model" well beside a 25-row workspace
  list, a truncated "not a wor…" sub on every folder row, raw
  `--home-goat-…` fallback names, a `$0.00 unknown/unknown` model row
  and a squashed "where" column on top sessions — fixed with
  `align-items:start`, a six-row cap + folded tail, `folderLabel`,
  dropping the zero-cost unknown row, and a fixed-width column plus
  `lastAt` from the backend. Light + dark + 600 px + empty period all
  captured and read; `overlayAudit` ok. Curated:
  `docs/screenshots/dashboard-v2-{7d-light,breakdowns-dark,empty}.png`.
  Merged + deployed; owner's first look on real data (25 folders, 9
  models): "cards desalinhados" — the 2-column row grid with
  `align-items:start` left a hole under the short model panel. Follow-up
  `feat/dashboard-masonry`: `.dash-grid-2` is now CSS multi-column
  (`columns: 2`, `break-inside: avoid`), panels pack top-to-bottom per
  column; label column widened to `minmax(110px,190px)`. Re-captured on
  the scratch instance at All time — no holes. Second report: the
  scrollbar sat mid-window — `.dashboard-view` was both the scroller
  and the 1040px centred box. `feat/dashboard-scroll`: the outer element
  scrolls full-width, a new `.dash-inner` wrapper carries the cap and
  centring (`.dashboard-on` still keys on `.dashboard-view`). Measured
  on scratch at 1700px: scroller right = window right, inner right =
  1492.

- **2026-09-01** — **Logo opens the dashboard from anywhere**, on branch
  `feat/dashboard-logo-link`. Owner dogfooded ADR-0041's dashboard for
  real and found it unreachable in practice: it only ever showed with
  zero tabs open, so checking spend meant closing every open agent
  first. Considered a 6th sidebar icon (ADR-0026 already flagged the
  header as tight — 4 icons yield the version string below ~286px);
  owner's call instead: make the existing "PiCode" wordmark clickable.
  `App.jsx` gained `dashboardPinned` state — `showHome` is now
  `(noTabs || dashboardPinned) && hasData` — and a `.dashboard-on` class
  on `#workspace-view` (`> *:not(.main-tabs):not(.dashboard-view) {
  display:none}`) hides whatever surface was showing underneath, rather
  than enumerating every surface type the way `.term-on`/`.file-on`
  already do (those two only ever named the cases that visibly
  collided; this one has to be exhaustive by construction). Every
  open/select entry point (`openTab`, `openTermTab`, `openFileTab`,
  `openGitTab`, `openTreeTab`) unpins on call — an early version tried a
  single `useEffect` keyed on `selectedId` instead, which looked
  simpler but silently failed to unpin when the user clicked the very
  tab that was already selected before pinning (its value never
  changed, so the effect never re-fired): caught live via `agent-browser`
  clicking the same already-open tab twice, not by reasoning about the
  code. Verified against real session data (workspace with an open
  agent + 4 terminals): pin via logo, dismiss via the same tab, dismiss
  via a different terminal (the direct `openTermTab` path, which
  bypasses `openTab` entirely) — all three re-tested after the fix.
  `overlayAudit` clean, both themes screenshotted.

- **2026-09-01** — **Session observability dashboard (ADR-0041)**, on
  branch `feat/session-dashboard`. Owner rejected a first attempt
  (`feat/home-dashboard`, unmerged): a `HomeView` that listed workspaces/
  agents/terminals in the no-tabs-open main pane — "ux fraca... sem
  sentido visto que a sidebar já está disponível" — and asked for
  analytics/observability instead, explicit that it should not just be
  another quick-access list.
  - Research surfaced two existing refusals that needed reconciling
    before building anything: `docs/design/session-surface-roadmap.md`'s
    "Cost as a new page | it belongs on the session chip" (about *one
    session's* cost; this is a fleet aggregate, different question) and
    the "X is not the home" pattern used twice against other surfaces
    annexing the live agent-conversation identity (this only ever renders
    when nothing is open, so nothing is displaced). Owner confirmed this
    reading explicitly before implementation.
  - New `GET /api/sessions/stats?range=today|7d|30d|all`
    (`internal/session/stats.go`, `internal/server/session_stats.go`):
    scans session JSONL at **message** granularity (`entryTS`/`costFrom`,
    already unexported in the same package) rather than bucketing by
    `Summary.UpdatedAt` — file-mtime bucketing would dump a multi-day
    session's whole cost onto one day. Response carries only aggregates,
    never `Preview`/raw rows (sabotage-tested). `range=all` has no cheap
    pre-filter and no cache — accepted debt, same posture `ListAll()`
    already has.
  - Frontend: `DashboardView.jsx` + `StatTile.jsx` (value/delta/sparkline)
    + `SpendByProvider.jsx` (ranked one-hue bar list) + `DateRangePicker.jsx`
    (native radio segmented control, mirrors `.termset-seg`). New
    `lib/sparkline.js` (pure SVG path geometry, mirrors the
    `lib/gitgraph.js`/`GitGraph.jsx` split — no charting dependency
    added) and `lib/dashboardStats.js`, both unit-tested. Range choice
    persists per-viewer in `localStorage` (`lib/openTabs.js`), not the
    hash router — this app's `#` routes name what's open, never a view
    filter.
  - Visual-review caught one real defect before ship: the stat-tile
    loading skeleton's `<span class="skel-line">` had no block-level
    context (`.stat-tile-skel` lacked `display: flex`, unlike the
    sibling `.spend-skel` that already worked), so the skeleton bar was
    invisible — fixed, re-verified against an artificially slowed
    fixture (`agent-browser wait --fn` polling for `.stat-tile-skel`,
    since local fetches are normally too fast to catch by a fixed delay).
  - Cross-check: `range=all`'s `current.cost` matched `/api/sessions/all`'s
    summed `Summary.Cost` exactly (7.33 == 7.33) on the same fixture data
    — two independent code paths agreeing.
  - Ported the `WorkspaceRows.jsx` extraction (`AgentRow`/`TermRow` pulled
    out of `Sidebar.jsx`) from the abandoned `feat/home-dashboard` branch
    via `git apply` of just that file pair — still valid on its own,
    independent of the rejected `HomeView` it was built for. `HomeView.jsx`
    and its ADR were not ported.
  - Not merged; `feat/home-dashboard` (superseded, unmerged) left as-is
    for the owner to delete at their convenience.
  - **Follow-up fix, same day, caught during the owner's live preview**:
    `scanFile`'s message-timestamp read used `time.Unix(ts, 0)`, treating
    pi's real `message.timestamp` (epoch **milliseconds**, JS `Date.now()`
    convention) as seconds — the parsed time landed around the year
    58620, tripped the "future relative to window" guard, and silently
    dropped almost every real message. The unit tests never caught it
    because `msgLine()`'s synthetic fixtures also used `.Unix()`, so they
    validated the code's assumption against itself, not against pi's real
    wire format — the earlier "confirmed seconds from a real fixture"
    research note was reading a `compaction` event's unrelated top-level
    `timestamp`, not a `message.timestamp`. Found by copying real
    `~/.pi/agent/sessions` (83 files, 174M) into a scratch preview instead
    of trusting synthetic-only QA: `range=7d` and `range=all` came back
    identical (4 messages total) against 24,811 real ones. Fixed to
    `time.UnixMilli`; re-verified the cross-check from above against real
    data (854.9008726400004 vs `/api/sessions/all`'s independent
    854.9008726400006 — last-digit float ordering only). Commit
    `d532c31c` on the same branch.

- **2026-09-01** — **Apps host: tab height unified, split pane
  resizable** (`feat/apps-host-chrome`). Owner used the just-shipped
  Inbox tabs and flagged two chrome inconsistencies: "tabbar" height
  differs by location, and a "sidebar" (the Inbox's list pane) isn't
  resizable — clarified to mean the apps host's `.app-split` list pane,
  not `#sidebar` (confirmed live via dispatched PointerEvents that
  `#sidebar`'s own resize already works correctly; CDP's synthetic
  mouse events just don't trigger it reliably, a test-harness quirk,
  not a product bug).
  Audited every `role="tab"`/`role="tablist"` in `web/src/components/`:
  4 patterns, 3 heights — `.mtab` (window tab strip) and `.brand-tab`
  (sidebar's own Workspaces/Agents/Terminals/Apps/Pins switcher) both
  already resolve to 40px via `--chrome-h` (structural — that row is
  shared with the sidebar header, left alone); `.pref-tab` (Preferences
  nav) was `36px` hardcoded; `.app-tab` (Inbox Active/Done/All) was
  `30px` hardcoded, the value introduced this session without
  reconciling it against anything else. Both now reference `--ctl-h`
  (36px) — already the file's dominant control-height token, and the
  smaller visual delta for the Inbox. `.termset-seg-face` (Terminal
  Settings scope picker) stays on `--ctl-h` unchanged: confirmed it's a
  mutually-exclusive value picker (radio group), not view navigation —
  already correct, not part of the inconsistency.
  Surveyed every existing drag-to-resize sizer before adding a 5th:
  `Sidebar.jsx` (`picode-sidebar-w`), `FileTreeSurface.jsx`
  (`picode-ft-w`, `.ft-split`/`.ft-sizer`), `FilePane.jsx`
  (`picode-file-w`), `GitGraphSurface.jsx` (vertical,
  `picode.gg.detail-h`) — all four persist to `localStorage`, all four
  use a bare 6px hover-highlight handle with no permanent affordance.
  `.app-split` (`AppSurface.jsx`) converted from CSS Grid to flexbox
  (nothing else in `app.css` depended on it being a grid — checked) so
  the new `.app-split-sizer` could use the same
  flex-sibling-with-negative-margin technique as `.ft-sizer` instead of
  fighting the stacked-breakpoint media query with `!important` (a
  pattern that doesn't otherwise exist in this file). New `AppSurface.jsx`
  state (`listW`/`resizing`/`stacked`, the last one now tracked live via
  `matchMedia("(min-width: 881px)").addEventListener("change", …)`
  rather than the one-shot check the mount effect already had) and an
  `onSizerDown` handler copied in the same shape as the other four —
  deliberately not extracted into a shared hook, since the codebase has
  independently reimplemented this same ~15-line pattern four times
  already rather than factoring it out. Persistence key
  (`picode-app-split-w`) is global, not per-app-id, matching
  `FileTreeSurface`'s own global `TREE_KEY` precedent — one open
  question inherited unchanged from that precedent: two split-app tabs
  open at once don't live-sync a drag between them, only the next
  mount picks up the saved width.
  QA on isolated :8616: tab height measured 36px (was 30) in a live
  DOM read; list pane measured 380→530px across a real drag and held
  530px after a full page reload; at 760px width the sizer is absent
  from the DOM (not just hidden) and `.app-split`'s computed
  `flex-direction` is `column`; dark theme screenshot clean. No new
  tests: the JS runner (`node --test src/lib/*.test.js`) only covers
  `web/src/lib/`, there is no `.test.jsx` or DOM harness in the repo,
  and this change adds no new pure-logic surface to test (the clamp
  math is inline, same as the four precedents it copies).

- **2026-09-01** — **Per-agent `--session-dir` (ADR-0040)**, on branch
  `worktree-pi-tui-session-dir`. Owner reported (screenshots) that after
  ADR-0039 shipped, an Agent's **chat** picker correctly showed only its
  own session — but the *same* agent's raw interactive pi TUI still
  showed every session sharing that cwd via pi's own native "Resume
  Session" picker. Root cause: that picker is pi's own code, reading
  `~/.pi/agent/sessions/<cwd>/` straight off disk, with zero awareness of
  PiCode's HTTP API or `agent_sessions` — a DB-side filter structurally
  cannot reach it.
  - Reopened ADR-0039's own rejected alternative ("give each agent a
    private `--session-dir`") after three facts confirmed live against
    `pi 0.84.4`: it lands a fresh session directly in the given dir (no
    cwd sub-bucketing), `--continue` finds/appends there and never
    touches the default bucket (proves it governs lookup, not just
    storage), and an explicit `--session <path>` outside the dir still
    wins for a resume (so **no physical migration** of existing sessions
    was needed — only future fresh starts move).
  - New `session.AgentDir(agentID)` (`~/.pi/agent/sessions/<agentID>/`),
    nested *inside* `Root()` specifically so `ListAll`/`ListRoot`/
    `UnderRoot` needed zero changes — confirmed by reading `ListRoot`'s
    loop (no naming-shape filter on subdirectories).
  - `Agent.CLIFlags()` unconditionally appends `--session-dir` — reaches
    all three spawn chokepoints (tmux, managed RPC, and the headless MCP
    OAuth spawn) through one function. Placed last in the flag list to
    minimize test churn (4 assertions in `store_test.go` needed updated
    expected lengths; the rest only checked a lower bound or leading
    elements and were untouched).
  - `safeSessionPath` generalized from a single `(cwd, path)` root to
    `(path, dirs ...string)`; `handleManageSessions` and
    `sweepOrphanSessions` gained a `workspaceSessionDirs` union so the
    workspace manage view and the age sweep also cover each agent's
    private dir. The delete/purge-preview flow (`cleanup.go`) got the
    same treatment (`DirStatsAt`, `RemoveAgentDir`) — required, not
    optional, since without it "delete sessions too" would silently leave
    an agent's `AgentDir` behind after this change, a regression this
    design itself introduces.
  - **Bundled by owner's choice**: a separate, adjacent gap found while
    researching `agent_sessions` — the age-based orphan sweep only
    checked an agent's *current* `session_path`, never its full
    `agent_sessions` history, so an older session already resumable in
    that agent's own chat picker (ADR-0039) could be silently deleted
    once it aged out. Closed with `Store.AllAgentSessionPaths()` and one
    added skip-check in the sweep.
  - Verified live end-to-end, not just proxied through `--print`/
    `--continue`: scratch instance, two agents in one workspace sharing a
    cwd, both started in **interactive** mode, each sent a real message
    over `tmux send-keys`, then `/resume` triggered directly inside each
    agent's own tmux pane (`tmux capture-pane`) — each agent's native
    "Resume Session (Current Folder)" overlay showed **only its own**
    session, the other agent's completely absent. This was the one fact
    the design couldn't verify itself and had to be checked by hand.
    Also verified: `pi --mode rpc --no-session --session-dir <X>` starts
    clean and leaves `<X>` empty (the MCP OAuth spawn is unaffected by
    inheriting the flag).
  - `make fmt-check`/`vet`/`test`/`build` all green.
  - Not yet merged to main or deployed to `:8445` — branch
    `worktree-pi-tui-session-dir` ready for review.

- **2026-09-01** — **Inbox: Active/Done/All tabs + manual cleanup**
  (`worktree-feat-inbox-tabs`, ADR-0037 follow-up). Items were never
  deleted (`UPDATE` only, confirmed by reading every write path) but
  `done` was invisible — the default view only ever queried
  `unread`/`read`, so a resolved question just disappeared, recoverable
  only via a raw `?state=done` API call. Owner's call, in order:
  visibility first, manual cleanup second, retention policy explicitly
  deferred. A segmented `Active | Done | All` control (`View.Tabs`, new
  optional field — ADR-0036 amendment, not a new block type) sits above
  the list; Done rows carry a truncated summary of the reply; a done
  item's detail now mirrors the Done list beside it instead of Active
  (it used to "jump" to the wrong list). Delete is both per-row (hover
  reveals a trash icon, confirm) and bulk (**Clear all done**, confirm
  with the count); store gained `DeleteInboxItem`/
  `DeleteDoneInboxItems`/`CountInboxItems`/`CountAllInboxItems`,
  `ListInboxItems` untouched — Active and Done stay two existing calls
  combined at the app layer rather than a new "all states" filter mode.
  Two REST routes for API parity: `DELETE /api/inbox/{id}`, and `DELETE
  /api/inbox?state=done` (bare `DELETE /api/inbox` refused, 400 — no
  accidental full wipe).
  Two real bugs, both found only by driving a real browser (chromedp
  against an isolated scratch server), not by reading the diff: (1)
  `AppSurface`'s selected-row lookup assumed every list-pane block had
  `.items`, and crashed (`Cannot read properties of undefined`) on the
  new actions-only "Clear all done" block — fixed generally (`b.items
  || []`), not just for this one block. (2) the post-action
  `returnPath` rule collapsed to Active root whenever the acted-on path
  merely started with `item/`, which also fired for deleting a
  *sibling* row while a *different* item's detail was open; it now
  compares the acted-on id against the currently-open item's own id,
  and only then collapses — a regression test
  (`TestInboxDeleteSiblingRowWhileDetailOpen`) pins the distinction.

- **2026-09-01** — **Per-agent session ownership (ADR-0039)**, on branch
  `worktree-session-ownership`. Owner reported (screenshot) an Agent tab's
  Search sessions picker showing a session that actually belonged to a
  Terminal in the same folder. Root cause traced to
  `handleListSessions` (`internal/server/sessions.go`): `agent=<id>` only
  resolved a cwd, then listed every `.jsonl` pi had written there,
  unfiltered — confirmed against live code, not just the report.
  - New `agent_sessions` table (migration `015_agent_session_history.sql`;
    renumbered from `014` after merging main, which had independently
    claimed `014_inbox.sql` for ADR-0037)
    historizes every session id/path an agent is pointed at.
    `store.UpdateAgent` historizes on every `SessionPath` write (resume/
    fork/clone/adopt/import, one hook, not nine call sites).
  - Fresh spawns pre-mint a pi `--session-id` (confirmed live against the
    installed `pi 0.84.4`: creates-if-missing, reuses on repeat, not
    gated behind `--mode`) so a brand-new session is attributable from
    the moment it exists — `Agent.CLIFlagsForSpawn`, wired at the two
    real spawn chokepoints: `rpc.Runtime.Start` (managed) and a new
    `Deps.spawnFlags` helper (5 interactive/tmux call sites).
    `handleListSessions` filters `session.List(cwd)` against the agent's
    own historized set; current session is always shown as a safety net.
  - Terminals needed zero changes — confirmed they have no `SessionPath`
    field and never call this endpoint. Machine-wide `/sessions/manage`
    and `/sessions/all` deliberately stay unfiltered.
  - Caught in the same pass: the new `--session-id` flag broke
    `TestOpenCloseLifecycle`, which used bare `cat` as pi's stand-in
    (relies on zero args to block on stdin; real cat errors on
    unrecognized `--` flags). Fixed with a tiny wrapper script that
    ignores argv, not a change to production spawn logic.
  - Tests: new unit coverage in `internal/store`
    (`CLIFlagsForSpawn`, the historization hook, and a migration test
    that genuinely re-applies `015_...sql` via `Store.migrate()` against
    a pre-existing `session_path`, not a hand-copied approximation) and
    `internal/server` (two agents sharing a cwd each see only their own
    session; an unowned file is invisible to both but still shows in
    `/sessions/manage`). `make fmt-check`/`vet`/`test` green.
  - `make ci` (fmt-check, vet, test, test-js, build) green — no frontend
    files touched; `SessionBar.jsx` needed no changes since the scoping
    is entirely server-side.
  - **Verified live**, not just by unit test: scratch `HOME`/`PICODE_DATA`
    instance on :8491 (real `pi 0.84.4`, real auth), one workspace, two
    agents both defaulted to the same cwd — the exact shared-cwd
    precondition from the bug. Managed-started both, sent each a real
    prompt ("pong" / "ping"). Each agent's picker showed **only its own**
    session; `/sessions/manage` showed both, unfiltered, as intended.
  - **Second bug found in that same live pass, fixed before calling this
    done**: `agents.session_path` (not just the new `agent_sessions`
    table) never got backfilled after a *fresh* spawn — only an explicit
    resume ever wrote it. Harmless for the picker itself (it falls back
    to the newest owned session), but `/sessions/manage`'s `inUseBy` —
    the guard that stops the orphan-cleanup sweep from deleting an
    actively-used session — reads `agents.session_path` directly, so a
    freshly-started, never-resumed agent's session looked orphaned.
    `ResolveAgentSessionID` already existed for exactly this (written,
    never wired); `handleListSessions` now calls it — and backfills
    `agents.session_path` — the first time it notices an owned-by-id
    session whose path wasn't known yet. Re-verified live after the fix:
    `inUseBy` correctly attributes each session to its own agent.
    Regression test added (`TestListSessionsResolvesFreshSessionPath`).
  - Not yet merged to main or deployed to `:8445` — branch
    `worktree-session-ownership` is ready for review.

- **2026-08-31** — **`/roles` empty-state copy + notify-as-thread-line**
  (`fix/roles-empty-copy`). Owner asked why `/vision` is listed with no
  roles configured — it is (commands register at package load; config is
  read at run time, ADR-0028 dormant contract) — and approved fixing what
  it *answers* instead: lock commands with no config now say
  `No roles yet — /roles edit vision creates the first one.` (logic.ts),
  and a notify that answers a still-quiet slash segment becomes a thread
  **note line** (new item kind `note`: mark + command badge + text, the
  `/roles …` fragment as a prefill chip) instead of a fading toast —
  `slashNoteTarget()` in askForm.js decides; ask-memory persists notes.
  **Bug found while verifying**: `groupTurns` (turns.js:10) silently
  swallowed the new item kind — the note rendered nowhere despite the
  handler running; fixed by adding `note` to the loose kinds.
  Verified on the scratch rig (line renders, chip prefills, reload keeps
  2 notes); gates green (276 js + 60 pkg).

- **2026-09-01** — **portrait video preview fits the pane** (`feat/video-preview-fit`):
  a 1080×1920 `.mp4` used to set `width: min(100%, 960px)` with no max-height,
  so Preview grew a scrollbar and hid the controls under the viewport.
  `.file-preview-fit` is a one-cell grid that can shrink below intrinsic size;
  the `<video>` / `<img>` then `object-fit: contain` inside it. Markdown, SVG
  and mermaid still scroll. Scratch `:8460` QA: video 290×516 inside a 548px
  pane, `paneScroll === paneClient`, `overlayAudit ok`. Captures: happy
  portrait, Raw ("Can't show this file."), gone, decode error.
  visual-review: PASS (file-video-portrait-fit.png + overlayAudit ok; card 5/5).

- **2026-09-01** — **tab surfaces keep their state** (`fix/tab-state`):
  the file tree, git graph and app surfaces were rendered only while
  selected, so every tab switch unmounted them and threw away the
  expanded folders, the scrolled-in history with its open commit,
  search and branch filter, and the app's open item. They now follow
  the shape `TermSurface` already used — one mounted instance per open
  tab, `hidden` while another is selected — in `App.jsx` via
  `tabs.filter(isTreeTab|isGitTab|isAppTab).map(...)`, with the
  handlers closing over the tab's own id (a hidden tree resolving its
  root used to rename whatever tab was selected). `hidden` is set on
  every `<section>` a surface can return, skeleton and error included:
  one branch left out paints its skeleton beside the visible tab.
  New `web/src/lib/keepScroll.js` — `useKeptScroll(hidden, selectors)`
  restores scroll offsets in a layout effect (display:none drops the
  scroll box, so reading it at hide time reads 0) using a *capturing*
  listener, which is how `.gg-rows` is covered without threading a ref
  through `GitGraph`. Refresh discipline: hidden surfaces skip the
  window-focus refetch (otherwise every open tab re-reads folders
  nobody is looking at), and a reveal refetches only past
  `REVEAL_STALE_MS` (10s) — the git graph is deliberately excluded,
  its refresh has been manual since the ADR-0038 poll was dropped.
  QA on isolated :8613: 501-commit graph + open commit + `fix` search
  + scroll 3000 survived a switch; inbox item stayed open; tree kept 3
  folders and scroll 750; cycling all three tabs showed 3 mounted, 1
  visible, 1 overflowing scroller each time; branches overlay
  `overlayAudit ok`. Accepted cost: a restored tab loads eagerly at
  boot (one browse+gitstatus per tree tab), far less than the
  websocket+xterm terminals already pay. Not covered: a browser reload
  still starts collapsed — the state lives in the mounted component,
  and a reload is a new page.

- **2026-09-01** — **Inbox UI refined** (branch `worktree-feat-inbox-ux`):
  split view (list left / detail right; stacked and scroll-into-view
  under 880px), rows with unread dot + relative time (absolute on hover)
  + kind lozenge + hairline-separated meta, per-row hairlines with the
  selected row flush and stripe-marked, sections as uppercase eyebrows
  with counts and a divider between them, icon-only Done/Snooze revealed
  on hover/focus (the timestamp steps aside), reply form with
  Ctrl+Enter, one filled button per row (the app declares
  `Action.primary`), approvals gained **Decline** (forwards a real "no"
  instead of silence), centred blankslates for empty inbox and empty
  detail pane. Apps host grew optional fields only — layout/pane/empty
  hints, block+row meta/at, row tone/unread, action icon/primary — the
  four block types are unchanged (ADR-0036 amendment 2026-09-01).
  Global side-fixes: `.md pre` never had block styling (fences rendered
  as inline chips) and Tailwind's preflight had stripped markdown list
  markers — both fixed for every markdown surface.
  New: `scripts/inbox-smoke.sh <base-url> [agent-id]` fills one item of
  every shape (incl. long title, dead-agent question, snoozed, answered)
  for visual QA; `web/src/lib/relTime.js` is the shared relative-time
  helper (SessionsView/Packages still carry their own divergent copies —
  worth folding into it next time either is touched).
  Reviewed against Sentry/Linear/Geist/Primer/Grafana/Atlassian specs;
  measured computed styles to check claims (meta/timestamps were already
  on the right tokens). **Known, deliberately left**: `.btn-primary` in
  dark is `#7c8cf8` with white text (~2.5:1) — a product-wide contrast
  bug, not inbox-specific; fix deserves its own pass. No keyboard list
  navigation (j/k) yet.

- **2026-09-01** — **The repository root is now pinned to main by git
  itself**, not by convention (AGENTS.md §5). `.githooks/reference-transaction`
  aborts a `git switch`/`git checkout` that would move the root checkout off
  main; `.githooks/pre-commit` refuses feature commits made there. Git has no
  pre-checkout hook, but a branch switch is a symref update of HEAD and a
  reference transaction can be aborted (git ≥ 2.28) — so enforcement is
  tool-agnostic: it holds for every agent runtime, editor and script, which a
  Claude-Code-only hook would not.
  The trap worth remembering: `git worktree add -b <branch>` writes the NEW
  worktree's HEAD through the *same* transaction, from the same directory,
  with the same git dir, the same common dir and the same
  `ref:refs/heads/...` payload — the naive hook blocked the very flow it is
  meant to force. The only discriminator is the invoking command, read from
  `/proc/$PPID/cmdline` (with a `ps` fallback); so the hook is Linux/WSL-first
  by design.
  `make hooks` wires `core.hooksPath` (implied by `make dev` and `make ci`);
  a clone that never ran make has no guard, and `git -c core.hooksPath= …`
  bypasses everything — this is protection against carelessness, not malice.
  Escape hatch: `PICODE_ALLOW_SWITCH=1`. Note `gh pr checkout <N>` shells out
  to `git checkout` and is therefore refused in the root: review PRs from a
  worktree, or use the override.
  Verified on a clone of this repo, not in theory: switch/-c/checkout -b
  blocked, worktree add + switch/commit inside a worktree fine, commit on
  main in the root fine, `checkout -- <file>` fine, return to main always
  allowed, override works, and pre-commit blocks a feature commit when the
  root is off main. GitHub Actions calls individual make targets, so it never
  sets hooksPath.

- **2026-09-01** — **`make ci` now proves the git guards instead of assuming
  them.** `scripts/hooks-selftest.sh` runs the whole policy matrix against a
  throwaway repo (never this clone): switch/`-c`/`checkout -b` refused in the
  root, worktree add + switch + feature commit inside a worktree allowed,
  commit on main in the root allowed, `checkout -- <path>` allowed, override
  works, return to main always allowed, and pre-commit refusing a feature
  commit when the root is off main. Wired as `make hooks-check` (a `make ci`
  prerequisite) and as a Linux-only step in GitHub CI, which calls individual
  make targets and would otherwise never see it.
  It is sabotage-verified both ways: removing the parent-command
  discriminator makes the worktree checks fail (the guard would block the
  flow it demands), and making the hook permissive makes the refusal checks
  fail. Either regression is silent without this test.
  `make hooks` moved into `scripts/hooks-enable.sh` and now **fails** when
  `core.hooksPath` was redirected elsewhere. Trap found while building it:
  `core.hooksPath` is a path-type config, so git reads it back **absolute**
  from a linked worktree even when written as `.githooks` — comparing the
  literal string reported every correctly-configured worktree as broken,
  which would have failed `make ci` for every agent. The script compares
  resolved paths against the main worktree's `.githooks`.

- **2026-09-01** — **Two Inbox fixes, both found by testing the feature
  live** (`pi install -l ./packages/pi-inbox`, a real `ask_human` turn).

  **Typography**: the split-view refactor had set `.app-surface { font-
  family: var(--sans) }`, flipping the whole app surface (titles, meta,
  buttons, section labels) to sans. That breaks the house rule — `body`
  is mono by default and only `.conversation` (chat prose) opts into
  sans; git graph, file tree, sidebar, settings all stay mono. Removed
  both overrides (`.app-surface` and `.app-surface .ft-title`) and the
  reply textarea's explicit `--sans`; the surface now inherits mono like
  every other list/detail app in the product.

  **Park-and-wake gap**: replying to a question from an agent parked in
  a TUI/tmux session queued a `follow_up` task that nothing ever drains
  — `deliverLoop` only exists for the RPC runtime (ADR-0037 promised
  "the agent picks it up on the next start" without that caveat).
  Fixed by refusing early: `store.RespondAndForward` gained a
  `deliverable AgentDeliverable` parameter; when it says no the item is
  annotated and stays open instead of enqueueing a message that would
  sit forever. `deps.agentInteractive` (internal/server/agents.go, next
  to the existing `runMode`) computes it from `Runtime.Get` + tmux.
  **The regression that shipped first**: the raw `/api/inbox/{id}/respond`
  route negated correctly; `internal/server/apps.go`'s `appsHost()` —
  which the actual UI goes through — passed the same boolean straight
  through under a *differently named* field
  (`Host.AgentInteractive`, true=interactive) into a parameter
  expecting the opposite polarity (`deliverable`, true=ok). Silently
  inverted: the UI accepted the reply and queued it forever. Caught
  live in the browser, not by the unit test I'd written for the app
  layer — that test set the Host field directly, bypassing `appsHost()`
  entirely, so it could not see the wiring bug. Fixed by unifying the
  name and polarity (`Host.AgentDeliverable`, matching
  `store.AgentDeliverable` exactly) and adding
  `TestInboxAppActionRespondInteractiveAgent`, an HTTP round trip
  through the real apps route against a real tmux session — sabotage-
  verified: reintroducing the missing negation fails it.
  Also hardened in passing: `rpc.Runtime.Get` panicked on a nil
  receiver (`r.mu.Lock()`); every other test server in this codebase
  either sets `Runtime` or gets lucky and never calls `runMode`/
  `agentInteractive`. Now nil-safe, matching the rest of the codebase's
  nil-safe accessors (`apps.Registry`, `Deps.Apps`).
  `pi-inbox` stays installed in this workspace's `.pi/settings.json`
  (`../packages/pi-inbox`, relative — portable across clones) as the
  live proof the package works; nothing else depends on it being there.

- **2026-09-01** — **pi-inbox + pi-roles coexistence verified live.**
  `.pi/settings.json` now carries both as project packages
  (`../packages/pi-inbox`, `../packages/pi-roles`); `npm:pi-agent-
  browser-native` moved out of the project list but stays active from
  the user's machine-wide settings, so nothing lost there. Two real
  `pi -p --no-session` turns with both extensions loaded: (1) no
  `PICODE_AGENT_ID` — `ask_human` filed correctly under the
  `sourceKind: system, sourceId: "pi (unmanaged)"` fallback; (2) a real
  free agent, identity attributed correctly, reply POSTed → 200, item
  `done`, `follow_up` task `queued` → agent started managed →
  `delivered` in ~4s. No load conflict between the two extensions in
  either case.

## Recent activity (archived 2026-09-01)

- **2026-08-31** — sidebar version reflects the build (`fix/version-truth`):
  `version.Build()` appends the Go-embedded `vcs.revision` (+ dirty `*`)
  unless the release workflow stamped `version.Stamped`; wired into
  `/api/version` (new `semver` field keeps comparisons pure), `picode
  update`, and the sidebar (ellipsis + title for narrow widths).
  install.Newer/backup keep plain `Version`. Tests in `version_test.go`.
  Follow-up in the same session: the dirty `*` marker was dropped — Go
  stamps `vcs.modified` against the primary checkout (not the linked
  worktree being built), and with parallel agents that checkout is
  routinely dirty, so the flag was pure noise; the revision alone is the
  signal.

- **2026-08-31** — roles follow-up (`fix/roles-gate`): the `role-state`
  endpoint is gated on pi-roles being in the agent's effective package
  list (`agentHasRolesPackage` over `loadPackageReport`; honors
  `PackagesIsolated`, fails open on listing errors) — an uninstall no
  longer leaves an orphaned composer chip. `/roles auto` accepted as an
  alias for `/auto` (pi-roles 0.5.1). Gate covered in
  `roles_state_test.go` (valid v1 file + no package → null; back → state).

- **2026-08-31** — **/roles: active-role chip, restored locks, rich
  pickers, scoped remove** (`feat/roles-active`, planned + approved;
  ADR-0033 amendment #2). pi-roles 0.5.0.
  - **State contract v1**: extension writes
    `~/.pi/agent/roles-state/<agent>.json` on mode/roles changes
    (`stateJson`/`parseState` in logic.ts, tested); `session_start`
    restores a lock whose role still resolves (model applies on next
    input, never at startup). Server: `GET /api/agents/{id}/role-state`
    (`roles_state.go`, null on missing/broken/future-version, tested).
  - **Composer chip** (`RoleChip.jsx`, SearchCombo like the other chips;
    presence-driven, no settings flag): `auto` quiet, lock in accent with
    a Lock icon; dropdown = roles with definitions + Edit roles…;
    picking sends `/role <name>`/`/auto` through sendTask. Refetch on
    select/snapshot/settled/notify.
  - **Rich select labels** (`roleOption`/`roleFromChoice`) in
    `/roles` picker, edit and remove; web trims the decoration off pills
    and definition lines.
  - **Scoped remove** with smart-skip (`removeScopes`): one layer →
    no question; both → `Remove from` select; without the env →
    workspace, as always. `Removed x (scope)` renders as its own line.
  - Bugs caught while verifying: missing `BACK` import in roles.ts
    (remove crashed); RoleChip's `triggerClassName` replaced
    `cockpit-chip` instead of extending it (squished chip). Also learned:
    a follow_up carrying an extension command is rejected by pi — only
    reachable via direct API posts (the web queues follow-ups locally).
  - Decision table 1–14 verified on the scratch rig (state file content,
    restore across restart, fallback-to-auto when the role vanishes,
    chip present/absent/lock, dropdown pick, rich pickers, remove both
    scopes, dark, reload); rows 13–14 by unit tests + design (TUI).
    Gates green (286 js + 69 pkg + Go incl. new endpoint test).
  - **Note (env)**: agent-browser now runs in a named session
    (`AGENT_BROWSER_SESSION`) — the default shared browser was hijacked
    mid-QA by another agent's session; use named sessions in dogfoods.

- **2026-08-31** — **`/roles` empty-state copy + notify-as-thread-line**
  (`fix/roles-empty-copy`). Owner asked why `/vision` is listed with no
  roles configured — it is (commands register at package load; config is
  read at run time, ADR-0028 dormant contract) — and approved fixing what
  it *answers* instead: lock commands with no config now say
  `No roles yet — /roles edit vision creates the first one.` (logic.ts),
  and a notify that answers a still-quiet slash segment becomes a thread
  **note line** (new item kind `note`: mark + command badge + text, the
  `/roles …` fragment as a prefill chip) instead of a fading toast —
  `slashNoteTarget()` in askForm.js decides; ask-memory persists notes.
  **Bug found while verifying**: `groupTurns` (turns.js:10) silently
  swallowed the new item kind — the note rendered nowhere despite the
  handler running; fixed by adding `note` to the loose kinds.
  Verified on the scratch rig (line renders, chip prefills, reload keeps
  2 notes); gates green (276 js + 60 pkg).

- **2026-08-31** — **`/roles` chat UX pass** (`feat/roles-chat-ux`; owner
  verdict on the shipped surfaces: "que porcaria de UIUX … texto seco,
  zero identificação"). P0+P1 of the approved plan:
  - Ask cards remember the slash command that opened them (`cmd` on the
    card, `cmdOf()` in `askForm.js`) — the open stepper now has a
    `ROLES <args>` header with Cancel in the corner; pills connect with
    `›`; the combo trigger names the field ("Choose provider…").
  - Confirms are a block (question + file chip + verbs); titles starting
    Delete/Remove get **Delete** (danger-at-rest, scoped to `.ask-confirm`)
    / **Keep** instead of Yes/No.
  - `summaryLine` refactored over a typed `summaryParts()` (definition /
    role / cleared / kept / empty / text); `AskOutcome` renders finished
    flows as one-liners with mark + `ROLES` badge + chips (provider icon
    via `ProviderFace`, thinking, scope, file). The nothing-to-clear line
    carries a "Set one up → /roles add" chip that prefills the composer
    (`onPrefill` threaded App → ChatSurface → Conversation).
  - Extension 0.4.1: confirm-No notifies `Kept <rel>` so the line stops
    degrading to "this agent · No". TUI otherwise untouched.
  - Verified on the scratch rig, light+dark, reload persistence (cmd and
    note serialize through ask-memory; cancelled still drops by design),
    overlayAudit ok on the dropdown. Gates green (274 js + 60 + 60 pkg).
    P2 (slash-bubble compaction, spacing rhythm) not started — follow-up.

- **2026-08-31** — MCP adapter detection from a terminal tab. `#/mcps`
  passed `selectedId` (`t:…`) as `agent`, GET 404'd, and the catch painted
  "Install the MCP adapter" even with `npm:pi-mcp-adapter` on the machine.
  Same bug Packages already fixed (3b3713e3). Server ignores a non-agent
  *and* a non-workspace id; a workspace terminal still carries that
  folder as MCP/Packages context so the machine list stays; the pane
  passes `agent.id`. A load error is Retry. Table-tested.
  **visual-review: PASS** (empty+adapter, blocked Open packages, error
  Retry; overlayAudit ok; card 5/5). First deploy was overwritten by
  `main`'s binary — user still saw the bug on :8445 until redeploy.
  That repeated on 08-31 evening: three more deploys from `main` (git
  graph work) shipped without this branch until it was merged.

- **2026-08-31** — Remove workspace can delete local data (ADR-0035):
  opt-in checkbox + GitHub-style typed folder-name confirmation; server
  re-verifies the name and refuses root/home (guard sabotage-tested);
  remote repo never touched. Clone-form segmented now full-width with
  folder/git icons. QA on 8448: wrong name keeps Remove disabled, right
  name deletes the folder from disk, plain remove keeps it; light+dark
  read, overlayAudit ok. **visual-review: PASS** (shots 10-13; card 5/5).

- **2026-08-31** — **pi-roles: choose the save target; `/roles clear`**
  (`feat/roles-scope`, ADR-0033 amendment). Owner asked how the chat
  session picker scopes (folder, confirmed by reading `session.List` /
  `AgentCwd` — sessions and roles are both per-cwd, not per-agent; only
  `agent.SessionPath` differs) and then approved this instead of only
  documenting the workaround (`cp` the overlay to the workspace file, or
  edit from a plain terminal).
  - `pickStart`/`pickAnswer` (`packages/pi-roles/src/logic.ts`) gained a
    `scope` stage: under `PI_ROLES_AGENT`, thinking is followed by a
    **Save to** select (*this agent* / *workspace* / `‹ back`); without
    the env the question is skipped, as before. `editFlow`/`addFlow` in
    `extensions/roles.ts` resolve a `layerFor()` (agent overlay vs.
    `.pi/roles.json`) from the answer.
  - New command **`/roles clear [agent|workspace]`**: confirm, then
    delete the whole file. No arg under the env asks which; a lock whose
    role stops resolving falls back to `/auto`.
  - Chat: `fieldLabel`/`summaryLine` (`web/src/lib/askForm.js`) learned
    the `Save`/`Clear` labels and the `(workspace)` suffix on the
    definition line.
  - **Bug caught in the same dogfood pass, fixed before shipping**: the
    existing role-picker regex (`/\broles?\b/`) also matched "Delete
    this roles file?", mislabeling the confirm's "Yes" as a role name —
    `/roles clear agent` rendered `Yes — .pi/roles/qa-213680.json`
    instead of `Cleared …`. Narrowed to `/^roles\b/` (only the role
    *picker* titles) and made the delete-confirm title itself carry the
    `Clear` label too, since the arg form (`clear agent`) skips the
    select step and needs the confirm alone to still gate the
    note-vs-fallback branch in `summaryLine`. Regression tests added
    (`askForm.test.js`, `logic.test.ts`).
  - Verified live on the roles-adversarial scratch rig (`PICODE_DATA`
    + port 8471, scratch `HOME` pointed at this worktree's package):
    Save-to appears/steps back correctly, workspace vs. agent writes
    land in the right file, `/roles clear` (both arg and no-arg forms,
    both scopes) deletes and reports correctly, "nothing to clear" warns
    without a stuck card, the ordinary `/roles` role picker still labels
    "Role". Gates: fmt/vet/test/test-js green, `npm test` 60/60,
    `make build` ok. **Merged to main and deployed to :8445.**

- **2026-08-31** — New workspace can clone a remote repo (ADR-0034):
  "Local folder | Clone repository" switch in the same dialog, URL →
  derived editable name/destination, blocking `POST /api/workspaces/clone`
  with host git credentials and prompts disabled, same-origin destination
  adopted. New `internal/gitclone` package; argv-injection defenses
  sabotage-tested. QA on an isolated 8447 server: real clone of
  octocat/Hello-World, `/tree/<branch>` honored, adopt, 409, classified
  not-found error, dark + 480px drawer, overlayAudit ok.
  **visual-review: PASS** (shots 01–09; card 5/5).

- **2026-08-31** — **`/roles` chat stepper: adversarial review + redo**
  (`fix/roles-adversarial`, **merged to main and deployed to :8445**;
  running agents must be restarted to reload the path package). The
  owner called the shipped stepper
  broken; the review confirmed the cancel-as-back design was the root
  cause and replaced it. What changed:
  - **Extension (ADR-0028 amendment):** cancel aborts the whole flow;
    going back is an explicit `‹ back` option on selects with a prior
    field. `pickAssignment` is now a pure state machine in
    `packages/pi-roles/src/logic.ts` (`pickStart`/`pickAnswer`, tested).
    A lock that lands on the already-running model still notifies, so the
    UI always gets a definition. pi-roles → 0.3.0.
  - **Web:** optimistic Working ends at the first real signal (dialog,
    notify, answer, `task_delivered` + 3s fallback) — never sticks;
    the completion notify folds into the card as its definition line
    (`default — xai/grok-4.6 · high`); back-walk answers `‹ back`
    instead of auto-cancelling; ask-memory gained a per-agent live slot
    (fresh agents persist across reload) and no longer cross-writes
    slots on tab switch; process exit / agent stop closes open cards;
    Stop shows while waiting, not only streaming; a queued follow-up is
    marked sent at flush time (was delivered twice).
  - **Go:** `deliver`/`SendTurn` no longer fail a task at the 60s
    deadline while a dialog is pending (a human thinking in a picker is
    delivery, not failure) — this was the red `context deadline
    exceeded` bubble. Regression test `TestSlowDialogIsDeliveredNotFailed`.
  - Decision table rows 1–9 verified in a scratch instance by browser
    (screenshots read; overlayAudit ok). Row 10 (TUI) verified by the
    package state machine tests + design (still one select at a time;
    Esc now aborts instead of stepping back — documented in README/ADR).
  - **Debt:** the TUI flow was not exercised interactively this session
    (no tmux dogfood) — behavior change is Esc=abort + visible `‹ back`
    rows. If the owner dislikes `‹ back` in the TUI list, the option can
    be dropped there only at the cost of reintroducing cancel ambiguity.
  - **Not done (refused per owner):** no `#/roles` page, no modal wizard;
    the flow stays in the conversation (C1).

- **2026-08-31** — Ask back-step: reopen the clicked pill, skip mismatched
  dialogs. Merged and deployed. Reload the agent. **superseded by the
  adversarial redo above** (auto-cancel walk removed).

- **2026-08-31** — Ask form UX: definition line persists, back on pills,
  compact stepper. Merged and deployed. Reload the agent for `/roles` back.
  **visual-review: UNVERIFIED** (owner dogfood).

- **2026-08-31** — File tree header shows the full folder path (mono,
  ellipsized, `title` tooltip) instead of just the basename — the owner
  noted the tab strip already carries the name, the header is where you
  confirm which folder. `.ft-title` gained `min-width:0` so a long path
  can no longer push Reveal/Refresh/Close off the header.

- **2026-08-31** — Extension select in chat is one growing form (pills +
  filter dropdown), then a pill line. Merged and deployed.
  **visual-review: UNVERIFIED** (owner dogfood on 8445).

- **2026-08-31** — Z.AI Usage parse: GLM Coding Plan now sends
  `CREDIT_LIMIT` (unit 3/5 = 5h, unit 6/1 = week) instead of
  `TOKENS_LIMIT`. Live Pro payload (0% / 100%) is the test fixture.
  Merged `fix/zai-credit-limit`.

- **2026-08-31** — pi-roles v2 (ADR-0033): per-agent overlay via `PI_ROLES_AGENT`.
  Workspace `.pi/roles.json` stays the default; `/roles` in a PiCode agent
  writes `.pi/roles/<id>.json`. Merged and deployed. Reload the agent.
  **visual-review: n/a** (no UI chrome).

- **2026-08-31** — **Provider Usage V3** (ADR-0031). Usage is per vault
  account (`GET/POST /api/providers/{id}/accounts/{aid}/usage[/reset]`)
  without swapping `auth.json`. OpenRouter / MiniMax / MiniMax CN / Kimi
  API keys get meters. Grok resets try PiCode OAuth, then Grok CLI
  `~/.grok/auth.json`, then `GROK_COOKIE`. Qwen Token Plan stays hidden
  (no API-key quota JSON). Chrome cookie dump refused. QA on :8451 —
  Usage on each account row, OpenRouter credits, empty/error/auth
  overlays. **visual-review: PASS**. Merged `feat/provider-usage-v3`.

- **2026-08-31** — Stop during `/roles` cancels the dialog and clears Working.
  Merged `fix/abort-clears-extension-wait`.

- **2026-08-31** — **File tree V2** (ADR-0032): Changes rows expand into
  working-tree diffs (`gitgraph.WorkingDiff`; `gitLoose` keeps stdout on
  --no-index's exit 1, /dev/null fallback only for ls-files-empty paths —
  both sabotage-proven); `POST …/reveal` opens the folder in the host
  file manager (`internal/osopen`, extracted from backup, WSL dedup);
  focus/visibility refresh instead of polling; branch pill badges
  `gitinfo.Dirty` (porcelain -uall count; cost: +1 subprocess per row per
  list, recorded in the ADR with the ?dirty=1 fallback).

- **2026-08-31** — `/roles edit|add` picks provider, then model, then thinking.
  Merged and deployed. Reload the agent to load the path package. **visual-review: n/a**.

- **2026-08-31** — **Provider Usage V2** (ADR-0031). ZAI and OpenCode Go
  API keys get Usage. Codex/Grok banked resets show in the dialog; Redeem
  confirms then `POST /usage/reset`. Grok reset row is omitted if the
  grok.com call needs cookies — weekly windows still load.

- **2026-08-31** — Sidebar row2 refined into pills (owner feedback after
  testing the file tree): `.ws-pill` — [folder + dir] opens the tree,
  [git + branch] opens the graph; repoLine/termLine expose `dir` alone.
  QA on :8501 — repo/non-repo rows, dark, 180px narrow. **visual-review:
  PASS**.

- **2026-08-31** — Packages Installed row is back when opened from a
  terminal tab (machine packages were never deleted). Merged
  `fix/packages-list-when-terminal`.

- **2026-08-31** — **Provider Usage dialog** (ADR-0031). `#/providers` shows
  **Usage** only when `quotaKind` matches the active oauth slot (Claude,
  Codex, Copilot, Kimi, xAI). `GET /api/providers/{id}/usage` fetches vendor
  windows in-process; tokens stay on the server. Dialog: skeleton → bars /
  empty / error / Sign in. Statusbar still does not invent quotas.
  visual-review: PASS (usage-windows/empty/error/loading/auth, overlayAudit ok).

- **2026-08-31** — **File tree per workspace/terminal/agent** (ADR-0030):
  `#/tree/<w|t|a>/<id>` opens a read-only tree of the owner's folder, tab
  deduped by canonical root (`d:<root>`, owners in `picode-tree-owners`).
  A **Changes** section on top lists `…/gitstatus` (porcelain `-z -uall`,
  re-anchored to the owner's cwd; rename records consume two NUL fields —
  sabotage-proven); changed files and ancestor folders carry kind dots.
  Workspaces became file-reading owners (`browse|text|blob|file|gitstatus`,
  `ws_free` refused) so empty workspaces (ADR-0027) browse too; terminals
  gained `/browse` at the live pane cwd; `browseAgentDir` answers `root`.
  Entry points: row2 folder icon (agents/terminals), Files on the
  workspace card + palette. Terminal trees pin; manual Refresh after a
  `cd` renames/merges the tab. Known limit inherited from the git graph:
  navigating by URL to an owner created seconds ago can flash "gone"
  until the app's list refreshes. QA: dedupe, cd+rename, empty/non-repo/
  deleted-folder states, dark/light, overlayAudit ok.

- **2026-08-31** — User menu fit: popover 236→268px so the Theme/Layout
  segments ("Desktop · Auto · Mobile" + icons, mono 12px) stop clipping;
  segments are equal-width and centered; Install app spans the row
  (`.um-install`) and carries IconDownload (mobile drawer inherits the
  icon). Isolated visual on :8507. **visual-review: PASS**.

- **2026-08-31** — Merged `feat/model-roles` as ADR-0028/0029 (numbers
  shifted: main already had 0025 tmux catalog and 0026 sidebar). Isolated
  visual on :8477; installed :8445 untouched. **visual-review: PASS**.

- **2026-08-31** — **Workspaces start empty** (ADR-0027): POST /api/workspaces
  registers the folder only; the New-workspace form is name + folder
  (Provider/Model/Thinking and the session shortcut stay on agent forms);
  `workspaceView.Agent` is a pointer with omitempty (the zero-object read
  as a truthy agent everywhere); empty-workspace open/close/sessions/status
  answer 409. connectPanel now takes the agent id (fixed: with two agents
  it connected to the workspace's first). Also shipped: workspace cards
  wear the project favicon (`GET /api/workspaces/{id}/favicon`, confined,
  sandbox CSP; fallback IconFolder — debt: a favicon added mid-session
  shows only after reload, and faviconRels is a fixed list), the
  Workspaces tab icon is lucide Folders, and the folder picker's address
  bar filters/navigates as you type (lib/pathFilter.js + useDebounced).


- **2026-08-31** — **ADR-0036 + ADR-0037 accepted** (docs only, no code
  yet): extensions host — apps as manifest + schema-driven primitives
  (no in-process JS ever, iframe deferred to v2, WASM deferred), fifth
  sidebar tab with an app grid seeded first-party; and the Inbox — core
  data plane (SQLite mailbox, `POST /api/inbox`, blocking-count badge)
  with the view as the first app, `packages/pi-inbox` giving agents
  async `notify_human`/`ask_human`. Web-benchmark research with sources
  in both ADRs. Next step: implementation plan for the 0036 pipeline.

- **2026-08-31** — **Apps host pipeline (ADR-0036)** on branch
  `worktree-feat-apps-host`: `internal/apps` (Manifest/Badge/Host,
  primitives View/Block/ListItem/Form/Field/Action + Validate, explicit
  Registry, hidden demo app), routes `GET /api/apps` (badges inline,
  failure-proof), `GET /api/apps/{id}/view?path=`, `POST
  /api/apps/{id}/action`; `Deps.Apps` nil-safe; env read in cmd only.
  Web: `x:<id>` tabs + `#/app/<id>` (all six dispatch points), fifth
  sidebar tab with grid/badges (tabs tighten to 26px under 240px so
  PiCode fits at 180px), AppSurface renderer (Field methods mirror
  rpc.UIDialog; Confirm→ConfirmDialog, toast/view/path results),
  palette entries, `lib/appPrimitives.js` normalizers. QA on isolated
  :8611 with PICODE_DEMO_APP=1: grid+badge, list→detail→danger
  confirm→toast+replaced view, path navigation, form (4 methods),
  gone card, no-env placeholder + empty `apps: []`. `make ci` green.
  Note: first screenshot at 1.5s settle caught boot mid-flight —
  use `--wait-ms 4000` for boot-dependent shots.

- **2026-08-31** — **ADR-0036 amended** (owner decision after the host
  shipped): in the marketplace era the sandboxed iframe (separate
  origin + bridge + published tokens/component package) is the
  first-class body surface for third-party apps; primitives stay as the
  cheap default, the host-chrome tissue, and the ONLY surface for
  sensitive actions (agent approvals, destructive confirms — tokens
  don't stop phishing, host-rendered controls do); the primitive
  vocabulary is frozen at the four blocks. v1 refusals unchanged.

- **2026-08-31** — **Inbox (ADR-0037)** on branch `worktree-feat-inbox`:
  migration 014 `inbox_items` (no FKs — items outlive sources), store
  CRUD + `RespondAndForward` (verb/state validated BEFORE forwarding —
  a rejected verb must never enqueue; caught by test), runtime filing
  in `pumpEvents` (`Hub.Len()` unobserved gate, `lastFinal` from
  agent_end, `stopRequested` so manual Stop files nothing), routes
  `POST/GET /api/inbox` + respond (409 + annotation on dead agent) +
  state/snooze, `internal/apps/inbox.go` seeded in `BuiltIns` (grid no
  longer empty), `packages/pi-inbox` (MIT; notify_human/ask_human with
  terminate:true, node:https loopback, soft failure when PiCode is
  down), `PICODE_AGENT_ID` in SpawnEnv. Fixed in passing: AppSurface
  load race (busyRef ate a navigation racing a focus refetch —
  latest-wins seq now). QA on :8612: 4 kinds via curl, badge 2+dot,
  needs-me/feed, question detail form → reply → toast → follow_up task
  `queued/source=inbox` verified in SQLite, dead agent 409 + body
  annotation + item open. `make ci` green. Live pi smoke of pi-inbox
  (`pi -e packages/pi-inbox/extensions/inbox.ts` + ask_human) left for
  the owner — spends provider credits.

## Recent activity (archived 2026-08-30)

- **2026-08-30** — **Sidebar restructured into four flat tabs** (ADR-0026):
  Agents (free, flat, name-sorted), Workspaces (one collapsible card per
  workspace — section collapses are gone), Terminals (free only), Pins.
  Workspaces now own terminals: migration 013 adds
  `terminals.workspace_id` (default `ws_free`, no FK — SQLite refuses ADD
  COLUMN with REFERENCES + non-NULL default; cascade is app-driven),
  `POST /api/terminals` takes `workspaceId` and a workspace terminal is
  born in the workspace folder. Removing a workspace kills its terminals
  (tmux best-effort, records + settings in one tx) and the cleanup dialog
  warns with the preview's count. Wire stays flat; grouping is client-side
  (`web/src/lib/termGroups.js`). Group hover actions became an absolute
  overlay — four buttons reserving grid space squeezed the workspace name
  to nothing at 180px. The brand version yields below 254px. Known
  behaviors, by decision: a stored `picode-side-tab:"agents"` now shows
  only free agents (no migration — indetectable; empty states carry the
  action); V1 has no move-terminal-between-workspaces; a tmux kill that
  fails after the DELETE leaves an orphan session recoverable via the tmux
  catalog (ADR-0025).

- **2026-08-30** — The last ADR-0025 debt is paid: tmux **array options are
  editable** in `#/termset` (`command-alias`, `terminal-features`,
  `terminal-overrides`, `status-format`, `update-environment`). They are
  edited as text, one entry per line; **Start from inherited** copies the
  inherited entries in; Apply rewrites the list per index and unsets whatever
  the layer held past the new length — measured first: tmux leaves a stale
  `name[2]` in place forever otherwise, and a whole-option unset before the
  rewrite resurfaces the layer below. An empty block is refused (tmux keeps no
  empty array layer, so it would be a pin that behaves as inherit).
  Browser QA found a bug that predates arrays: the **global** panel dropped a
  cleared non-curated key from the store but never unset it on the live
  sessions, so it kept applying. Fixed with `unsetClearedEverywhere` and a
  test that fails without it. Also fixed: `.dlg-input` pins height to one
  control row, which collapsed every list editor to a single line.
- **2026-08-30** — Agent cards got the terminal treatment: the name is the
  rename control (hover accent + dotted underline, click opens "Rename
  agent", `PATCH /api/agents/{id}`), prefilled with the shown name so a
  workspace `default` agent never opens a blank field. **No gear was added**:
  measured in the browser, the hover action row already spans x=120–230 of a
  244px sidebar, so a fifth icon would have cut the name's clickable run from
  49px to 23px. Debt (pre-existing, not from this change): with four actions
  the overlay covers the tail of a long agent name on hover — "Claude Code"
  reads "Claude". The full name is still in the `title`.

- **2026-08-30** — Terminal rows in the sidebar lost the pencil: the name
  itself is the rename control (hover paints it accent with a dotted
  underline, click opens the rename dialog), and the hover action row is
  now remove then settings, so the gear is the last icon on the line. The
  rest of the row still selects the terminal; the name button claims only
  its own text (`flex: 0 1 auto`), not the whole line. Owner's request from
  a screenshot.

- **2026-08-30** — Terminal appearance moved into `#/termset` ("Appearance —
  this browser" section, global page only); Preferences lost its Terminal
  tab and `#/preferences/terminal` degrades to Appearance. Storage homes
  unchanged (ADR-0024 amended). Shared pieces extracted: `ThemeCard.jsx`,
  `TermAppearance.jsx`.

- **2026-08-30** — Follow-up caught by the owner's screenshot: the SELECTED
  terminal showed `~ / main` — an impossible pair. `POST /open` still
  answered with the record cwd and no git; the app merges that response into
  its list, so the stale path overwrote the live one while the old git
  survived. All four terminal-returning handlers now share `liveTermView`;
  a test opens a terminal after a `cd` and asserts the live path comes back.
- **2026-08-30** — Sidebar cards unified (terminals ↔ agents): second line is
  icon + path, or git icon + `path / branch` in a repo. `GET /api/terminals`
  now reports the live pane cwd (it printed the creation dir forever while
  the git facts beside it were live — the two disagreed after any `cd`).
  Workspace agent views carry per-agent git from the agent's effective dir;
  `repoLine` returns the git object (fixing the tooltip that read `.branch`
  off a boolean) and never pairs one directory's path with another's branch.

- **2026-08-30** — **Compact status moved into the chat** (merged from
  `feat/compact-chat-line`, Claude Code-inspired, PiCode tokens). The
  “Compacting” segment is gone from the composer statusbar; in-flight
  compaction is now a live line at the end of the conversation — pulsing
  accent dot (`.work-dot`), “Compacting session…”, elapsed in the chat's
  `turns.js` `1m 05s` format — and the finished compact folds into the
  existing one-line collapsible `compaction-card`, which `compaction_end`
  now fills from `ev.result.summary` so auto-compacts land live instead of
  on next reload (dedup by summary text; user-initiated flow still replays
  via `loadSessions`). “Nothing left to compact.” and failures are chat
  alerts; `picode-compacting` localStorage survives reloads/rebuilds.
  `make ci` green. **Visually verified** on an isolated scratch server
  (8468) with a crafted session, screenshots read: light collapsed card +
  live line, dark expanded markdown body, dark live line, collapse cycle
  back to one line, `overlayAudit ok`, composer carries no compact
  segment in any state.

- **2026-08-30** — Two guards, both closing holes opened earlier today. `picode install`/`deploy` now refuse a binary with no embedded UI: one was deployed by a plain `go build` and the browser got the ADR-0023 "not built yet" page. The check sits in the command layer, not in `install.Deploy`, because `picode update` deploys a *downloaded* release where "does this binary embed the UI" is the wrong question. And the `node_modules` make guard now stamps on `node_modules/.package-lock.json` rather than the directory — an empty directory with a fresh mtime satisfied make and the build then died on `vite: not found`, which is what it did. Both verified in both directions.
- **2026-08-30** — Pi item reappeared in the user menu: **a parallel agent deployed a stale `bin/picode`** (built before `4913a3a5`), not a source regression — main was clean. Fixed by `make build` + deploy from current main. **Rule for every session: before `bin/picode deploy`, run `make build` on a tree at current main** — deploying an old binary silently reverts UI changes that are already merged.

- **2026-08-30** — Removed the **Pi** item from the user menu (owner call): the update surface is the System card only. Also restored the pi-update CHANGELOG entry — it was lost in a conflicted merge earlier.
- **2026-08-30** — **Pi update alert shipped** and proven on a real release: System card with installed → latest, Copy command, and **Update now** (`POST /api/system/pi-update` → `pi update --self`, background ctx so a client disconnect cannot kill the install). Live run updated pi 0.84.3 → 0.84.4 end to end. **Ops note:** deploying with a plain `go build` binary (no `-tags embedui`) installs a disk-mode server that serves "UI has not been built" — always `make build` before `bin/picode deploy` (this bit us once today; fixed by rebuilding embedded).

- **2026-08-30** — ADR-0024: terminal settings. Written after removing tmux's forced `mouse on` broke scrolling in Pi's TUI while leaving Claude Code's alone — Claude Code takes the mouse itself, Pi does not, and one constant cannot serve both. The shape is Windows Terminal's (`profiles.defaults` plus profiles declaring only what they change), extended with user presets. Two storage homes on purpose: tmux behaviour is per session and shared across devices, xterm appearance is per browser and should differ. Decision only; no code.
- **2026-08-30** — **Pi update alert shipped** and proven on a real release: dot on the user-menu **Pi** item (registry check on /api/system, 6 h cache with stale-fallback for hiccups), System card with installed → latest, Copy command, and **Update now** (`POST /api/system/pi-update` → `pi update --self`, background ctx so a client disconnect cannot kill the install). Live run updated pi 0.84.3 → 0.84.4 end to end; dot cleared after. **Ops note:** deploying with a plain `go build` binary (no `-tags embedui`) installs a disk-mode server that serves "UI has not been built" — always `make build` before `bin/picode deploy` (this bit us once today; fixed by rebuilding embedded).

- **2026-08-30** — Memoised the repository lookup in the occupant scan: 200 agents sharing a subfolder went from 4.6s to 22ms, and the cost stopped growing with the agent count. Implementing it uncovered a real bug shipped in ADR-0022 G1 — `gitgraph.Key` resolved git's relative answer (`../.git` one level down) against `--show-toplevel` instead of against the directory asked about, so any cwd below the repo root got a key one level too high. Effect: an agent in a subfolder was silently dropped from the graph. `TestNestedRepoIsNotAnOccupant` had been passing for the wrong reason and now passes for the right one.
- **2026-08-30** — ADR-0022's two unmeasured costs, measured. **Commit ceiling: there isn't one** within what the product allows — layout is 14ms for 10,000 commits, the server answers `?limit=2000` in 0.12s (408KB), and the browser holds 2,000 rows / 17k DOM nodes with a row click at 0.4ms and scrolling at 0.1ms. The 250 default is conservative, not a limit. **Occupant scan has a cliff**: free when agents sit at worktree roots, ~23ms per agent whose cwd is below one — see debts. Also worth recording: mid-measurement I nearly filed 'Load earlier is broken' as a bug. Instrumenting the button showed the clicks were never reaching it — `agent-browser click "text=…"` does not hit it, while dispatching on the element does. The feature works.
- **2026-08-30** — `picode install` / `deploy` survive a non-login shell. `systemctl --user` needs `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`, which a script, cron job or agent shell does not have; both commands copied the binary *before* calling it, so the failure left the new binary on disk with the old one running — hit exactly that during today's deploy, and only a hash comparison showed it. `install.Run` now fills the two variables from `/run/user/<own uid>` when the socket is there, and `EnsureUserSession` refuses before copying when it is not, naming `loginctl enable-linger`. Verified the injected values turn `systemctl --user is-system-running` from a bus error into `running`, and that the guard is what prevents the half-update: without it the test finds the installed binary replaced anyway.
- **2026-08-30** — Frontend tests run in CI. 197 of them were passing where nothing watched. The blocker was ordering — `npm test` needed `node_modules`, which only `make build` installed — so installing moved into its own target gated on `web/package-lock.json`, and `web` and the new `test-js` both depend on it. That also removes a second full `npm ci` per `make ci`. Verified the gate can actually fail: a deliberately broken test takes `make ci` to exit 2, and the guard skips the install on a second run (10s → 2.7s) but reruns it when the lockfile is touched.


- **2026-08-30** — `POST /api/workspaces/{id}/agents` accepts `workPath`; it was hardcoded empty, so ADR-0022's centrepiece — two agents in sibling worktrees of one repo — could only be built from free agents. Reuses `resolveAgentWorkDir`, the same resolver free agents use, so the two creation paths cannot drift; blank stays blank and keeps the agent on the workspace folder. Verified the test has teeth: with the old hardcode both agents pile onto `main`. **API only — no UI was added**, since nothing asked for one.
- **2026-08-30** — Clipboard validated in a browser, closing the ADR-0023-era debt. Text emitted as OSC 52 from inside a tmux pane came back out of the *system* clipboard via a real Ctrl+V — the whole chain, not an inference. Chrome refuses the write without a recent user gesture and accepts it right after a click; the handler's toast covers the refusal. That refusal path needed a synthetic probe to reach at all, so it is not being designed around. Firefox is still unchecked: the automation here only drives Chrome.
- **2026-08-30** — `make fmt` and `make fmt-check` stop reaching into `.worktrees/`. Both walked the tree with `.`, so a sibling agent's uncommitted code failed this gate — and `fmt` was worse than reported: `gofmt -w .` would have *rewritten* their files. Both now walk the directories `go list ./...` reports, the same module boundary `vet` and `test` always respected. The `git ls-files` fix I had written in this file was wrong and is dropped: tested, it misses a new file not yet `git add`ed, so it would pass locally and fail in CI. `go list` catches it, and covers 204 of 204 `.go` files in the module including `//go:build ignore` ones. CI's inline gofmt step now calls the same target instead of repeating the command.
- **2026-08-30** — ADR-0023 implemented. `internal/web` splits into a disk loader (default) and an embedded one (`-tags embedui`), which is what `make build`, `ci.yml` and `release.yml` now use; `internal/web/public/` is untracked and gitignored. The ADR's gating question is answered: over https the service worker registers, activates and fills `picode-assets-v1` with the hashed assets in disk mode, and both modes serve byte-identical asset URLs with identical Cache-Control (`immutable` for `/assets/`, `no-cache` elsewhere). The earlier `swRegs: 0` was a red herring — `main.jsx:16` only registers over https, so it was 0 in both modes. A disk build with no UI answers 503 with `run make web` instead of 404s, and starts serving the moment the files appear.
- **2026-08-30** — ADR-0023: the built UI stops being committed. It had been tracked since the bootstrap commit with no decision behind it — 330 files, 33 MB, 77% of the repo's commits, 133 files rewritten per UI change, and 335/334 `rename/rename` conflicts in two merges that day, every one resolved by rebuilding. Checked six peers (Grafana, Prometheus, Vault, Coder, Gitea, Syncthing): none commits it. Prometheus also answers the constraint that kept us — `//go:build !builtinassets` serves from disk and the default build does not embed at all. Decision recorded; the code change is not made yet.
- **2026-08-30** — User menu gained a **Sessions** item (between System and Providers) that opens the machine-wide `#/sessions` view; `go("sessions")` short-circuits the `/sessions/:id` template. Live click-through verified: menu → 36 folders · 99 sessions · audit ok.
- **2026-08-30** — Terminals stopped wearing tmux's skin. PiCode forced tmux's own `mouse on` since before `termWheel.js` grew its SGR fallback; its only remaining effect was tmux copy-mode on every drag, which is why text could not be selected. Owner tested `mouse off` on the real machine — the wheel still scrolls — so both call sites drop it. Paired with `allow-passthrough on` and a write-only OSC 52 handler so a copy made inside the pane reaches the system clipboard. Proved A/B in the browser: with passthrough on the handler fires, with it off the sequence never arrives. The read form (`52;c;?`) is refused on purpose — `@xterm/addon-clipboard` implements it, which would let any agent in a pane read the user's clipboard. Benchmark: `docs/benchmarks/2026-08-30-web-terminal-clipboard.md`.
- **2026-08-30** — **All-folders Sessions view** (`#/sessions`): every Pi session on the machine grouped by folder (workspaces first; "not a workspace" badges; 36 folders / 99 sessions / ~554 MB on the QA machine). Same actions per row — Open with… only where a workspace owns the folder (disabled + reason otherwise), Delete validated against the sessions root, Compact for in-use. New endpoints `GET/DELETE /api/sessions/all`; `ListAll` summaries now carry size. QA live: fixture in a non-workspace folder deleted end-to-end, scope link both ways, audit ok after height fix (19→36 px on the scope link).
- **2026-08-30** — **Sessions view shipped (A+B)**: `#/sessions/<id>` lists every Pi session under the folder (size/age/msgs/cost/provider, `inUseBy`) with Open with… (resume, no copy), Compact (in-use), Delete (orphans, confirm; in-use 409 with reason) — plus **auto-clean orphans** (Off/30/60/90 d; boot+daily+on-change sweep, default Off). New: `session.ListDir` (+Size in Summary, missing dir = empty), manage endpoints, `StartSessionSweep`. QA live on the 629 MB workspace: fixture deleted end-to-end, in-use blocked with tooltip, dialog lists workspace agents, empty state after the missing-dir fix, audit ok. Known debt: list rescans all JSONLs (same cost as the adopt picker) — needs a stat-only mode if it ever feels slow.
- **2026-08-30** — `web/node_modules` was tracked as a symlink to an absolute local path (added by accident in `971cc632` via `git add -A`); on any other clone it dangles. Untracked it and dropped the trailing slash from both `.gitignore` files — verified in a scratch repo that `node_modules/` leaves a symlink of that name untracked while `node_modules` ignores it, which is exactly how it got in.
- **2026-08-30** — Git graph G3 (ADR-0022): clicking a commit opens its diff. `git show -m --first-parent` is a correctness fix, not a preference — without it a merge arrives as a combined diff (`diff --cc`, `@@@`) that the unified-diff reader misreads silently; proved by removing the flags and watching the test go red. The hash is the only user-controlled part of the git command line, so anything but 40/64 hex is refused before it reaches git. `DiffLine` moved out of Conversation.jsx so the chat and the graph render diffs the same way. Screenshot caught the selected row scrolling out of sight when the pane opened.
- **2026-08-30** — Git graph G1+G2 (ADR-0022). `internal/gitgraph` reads the DAG, refs and worktrees; the column allocator is ported from mhutchie's Git Graph (MIT, attributed in the file header) minus its uncommitted-changes row. Two parser bugs caught by tests with teeth: git hands back a literal 0x1f typed into a message, which a plain Split turned into a *dropped commit*, and a 0x1e split a record into a phantom commit whose hash was someone's subject. Verified on the real repo: 250 rows, `overlayAudit ok`, no h-scroll, 26px rows, dark and light both read, and the occupant chips show `default` on main beside `graph-impl` on its worktree.
- **2026-08-30** — Green bar under the agent TUI killed: agent tmux sessions were the only ones without `status off` (tmux's default status line renders green). Set at `NewSession` and on every bridge attach — old sessions heal on next view (verified live: picode2 pane went 45→46 rows). Scrollbar note: each attach is a `tmux attach` in a PTY, so tmux owns the scrollback — the native xterm scrollbar never fills in either surface; the wheel scrolls tmux history (mouse on at attach). That is inherent to attaching any terminal to tmux, not a PiCode bug.
- **2026-08-30** — Agent TUI and PiCode terminals share **one surface**: the desktop agent terminal view now renders through TermSurface/ShellTerm (same xterm options, wheel/keys/links wiring, padding 0, custom scrollbar, screen margin) instead of the TerminalDock wrapper — computed-style diff between the two is now identical. Managed mode shows a one-line hint + Open TUI action. `closeTerm` moved to `lib/terms.js` (mobile still uses the dock component). Also recovered `web/node_modules` in the primary checkout after a parallel session replaced it with a self-referential symlink.
- **2026-08-30** — Compaction residue fixed at the root: the transcript window now cuts at the **last compaction boundary** (what pi replays), so reload no longer resurrects pre-compaction history. The summary renders as a collapsible **Session compacted** card (39 K chars confined to 253 px + scroll) instead of a giant assistant message. `compacted` in the API response separates "needs /compact" from "already compacted, file stays large" (cold boots stay slow — pi#8843).
- **2026-08-30** — /compact progress moved out of the conversation into the **composer statusbar**: "⠋ Compacting 1:23" segment with spinner + 1 s ticker, persisted per agent in localStorage (survives the TUI→managed panel rebuild and page reloads), cleared by the HTTP answer or `compaction_end`. Verified live on the 140 MB adopted session (78 events post-boundary vs 9 673 raw).
- **2026-08-30** — Sidebar spinner covers TUI work too: GET /api/tui-working polls tmux capture-pane for pi's Working state (3 s).
- **2026-08-30** — /compact is visible end to end: "Compacting session…" line in the thread, closes on HTTP answer or pi's compaction_end, "too small" maps to "Nothing left to compact."
- **2026-08-30** — Terminal/chat view persists per agent across reload (localStorage `picode-term-view`).
- **2026-08-30** — Transcript API paginated (`?tail=&skip=`): opening the 129.5 MB session loads a 200-event slice (~1 s server, small payload); Load earlier fetches older turns from the server with scroll anchoring.
- **2026-08-30** — Huge sessions render a window (~60 turns) with Load earlier on scroll; composer height tracks local text. Proven on a 122 MB session (226 turns → 60 mounted).
- **2026-08-29** — ADR-0022: the git graph belongs to a repository, not to an agent. Studied VS Code's Git Graph first (it is mhutchie's extension, not VS Code): one runtime dependency, `iconv-lite`, no framework and no charting library — 913 lines of DOM + SVG, of which 73 place the columns. Measured that worktrees share refs and differ only in HEAD, which is why one graph with N marked heads beats N near-identical graphs. Read-only in this ADR: no write path exists that could hit a worktree while an agent is mid-turn.
- **2026-08-29** — ADR-0021 adopt Pi session by copy. GET/POST `/api/pi-sessions`. New agent → From a Pi session.
- **2026-08-29** — Agents work in an isolated git worktree (AGENTS.md #5). After merge, remove the tree and the branch.
- **2026-08-29** — Keepalive debt paid: the tray's `wsl.exe` child now lives in a job object with `KILL_ON_JOB_CLOSE`, so the kernel tears it down however the tray dies — `taskkill /F` and crashes included, where no deferred Go code runs. Established first that killing the Windows-side `wsl.exe` does end the Linux process (launched a `sleep 99999`, killed its wsl.exe, watched it go), because the whole fix rests on that. Then proved it end to end on the machine: before, a force-killed tray left `sleep infinity` behind; after, `taskkill /F` leaves nothing. `startDetached` became `startSupervised`, and a failure to supervise is non-fatal — a keepalive that can be stranded beats no keepalive.
- **2026-08-29** — Tray icon is now the browser favicon (`web/public/favicon.svg`): the blocky Pi in white on `#09090b`, transcribed as rectangles in `mkicon.go` and drawn at 4x then boxed down so 16px stays legible. `icon_test.go` reads the SVG's fills and asserts the committed ICO carries both — verified it actually fails by regenerating with the old indigo. Restarted the tray on the owner's machine with the new mark (PID 29532); doctor still reports everything ok on both halves.
- **2026-08-29** — **PiCode Desktop is live on the owner's machine.** Ran `picode-desktop.exe` through WSL interop: `doctor` showed the mkcert CA already trusted from an earlier `setup-cert.sh`, leaving only the logon task; `install` elevated (one UAC) and registered `PiCodeDesktop` — `At logon time`, `Run As User: cfpp`, `"…\picode-desktop.exe" --tray`. The tray runs (PID 30320) with its `sleep infinity` keepalive holding the distro open. Every step now reports ok on both halves. Fixed a fourth bug the run exposed: `doctor` summarised only the distro, so a fully set-up machine was still told to run install; `printWindowsSteps` now reports whether its half is finished and the summary weighs both.
- **2026-08-29** — Ran the real provision on the owner's machine (Linux half). One change: `Linger=no` → `yes`. `/etc/wsl.conf` came back with the **same md5** and no `.picode.bak`; PID 447165 and all three tmux sessions untouched, because the unit was already enabled and running so nothing restarted. The root pass then exposed a third bug: `systemctl --user` always answers for the *calling* account, so from root it reported goat's running service as "present but not enabled" — confident and wrong. `Env.Acting` now records who the process actually is, `OnBehalf()` compares it to the target, and the service check returns **blocked** instead of guessing. The summary line also stopped counting a skip as "would change". Windows half (logon task, CA trust, keepalive) still not run — it needs the `.exe` on Windows plus one UAC click.
- **2026-08-29** — First `picode-desktop doctor` run found two real bugs no test had. (1) The root pass invoked a bare `picode`, which is on **no** PATH but the owner's — ADR-0018 installs to `~/.local/bin`, reachable only through that account's profile. `PicodePath` now resolves it once via the owner's **login** shell (`sh -lc`) and hands both passes the absolute path. (2) Far worse: `cmd/picode` fell through to `serve()` for any unrecognised argument, so the *old* installed picode, asked to `provision`, silently started a second server **as root** in `/root/.picode` on port 8446. Killed it and removed the directory; the owner's picode on 8445 was untouched. `dispatch` now owns the command list and a bare word it does not know exits 2.
- **2026-08-29** — Desktop **M5**: release plumbing (ADR-0020, plan complete). Tag-triggered `release.yml` builds picode for 5 platforms plus `picode-desktop-windows-amd64.exe` into one release; `version.Version` became a var so `-X` can stamp it (verified: a stamped build reports 9.9.9). `install.LatestReleaseFor` extracted so both binaries share one update check. A test reads the workflow and fails if the asset names drift from what `update` looks for — the kind of break that is silent forever. Tray icon: a pi generated by `mkicon.go` (`//go:build ignore`, `go:generate`) at six sizes, BMP payloads; the first draft merged its legs into a "T" at 16px, so the geometry was fixed against a rendered preview. Elevation is runtime `ShellExecuteW`/`runas`, **not** a manifest — the same binary is the tray, and `requireAdministrator` would elevate that too.
- **2026-08-29** — Desktop **M4**: the clean-machine path (ADR-0020). A stage machine derived from observed state, not saved progress, so an interrupted install resumes and a finished one is a no-op: install WSL `--no-distribution` → restart → install the distro `--no-launch` → create the account → provision. Both flags dodge the interactive account setup plain `wsl --install` opens. Exit **3010** is "succeeded, restart pending", not a failure; `RunOnce` resumes at the next logon and deletes itself. The Linux account is named after the Windows one (accent-folded, sanitised) with the password left **locked** — PiCode reaches root via `wsl -u root` and never needs sudo. Writing the 3010 test found that POSIX shells truncate exit status to 8 bits (3010 arrives as 194) while Windows does not, so the rule is tested through an interface rather than a real process. Live-WSL test confirms this machine is classified as ready and skips every stage.
- **2026-08-29** — Desktop **M3**: `cmd/picode-desktop` + `internal/desktop` (ADR-0020). Drives the distro with two `picode provision --json` passes (root, then owner) and merges them so whichever pass resolved a step wins — the rule that keeps "skipped for lack of privilege" from masking "fixed". Windows side: `onlogon` task at `/rl limited` (an elevated tray cannot reach the notification area), mkcert CA import gated on a count so logon does not re-import, `sleep infinity` keepalive against the idle timeout, `CREATE_NO_WINDOW` on every child. `wsl.exe` output is UTF-16LE **with no BOM** — decoding is decided by inspecting bytes, and the real 136-byte output is a base64 fixture. `make desktop` cross-compiles from WSL: 6.5 MB PE32+ GUI, `CGO_ENABLED=0`, no C compiler. Live-WSL tests (skip in CI) confirm it picks `Ubuntu` (WSL 2) and reads the account `goat`. New dep: `fyne.io/systray` (pure Go on Windows). **Not executed against the machine** — no `install` run.
- **2026-08-29** — Desktop **M2**: `picode provision` (ADR-0020) converges six steps — wsl.conf, systemd, linger, cert, unit, health — with `--dry-run` and `--json`, and root vs user scopes so the Windows side can drive it in two calls. `EnsureKey` merges `/etc/wsl.conf` by line: the owner's real file (comment, key order, `generateResolvConf = false` spacing) is a test fixture asserted byte-identical, and the fix writes no backup when nothing changed. Writing the `Run` decision table caught a real bug: blocked steps were reported as "planned" in a dry run, promising a fix no run could deliver. Extracted `tlsutil.LocalNames` so the self-signed and mkcert paths issue for the same hosts. Dry run on the owner's machine: 4 ok, 2 to fix (linger, unit) — matching the plan, with `/etc/wsl.conf` verified unchanged (same md5, no `.picode.bak`).
- **2026-08-29** — Track 3 live cwd: Ctrl+click asks tmux `#{pane_current_path}`. File-preview roadmap closed. Tests: PaneCwd + GET `/api/terminals/{id}/cwd` after `cd`.
- **2026-08-29** — Chat file cards sit under the turn file names (one per click). visual-review: PASS (file-chat-cards-inline.png).
- **2026-08-29** — Terminal: Shift+drag select, Ctrl+C copy if selected, Ctrl+V paste. visual-review: PASS (pref-term-copy.png).
- **2026-08-29** — Sidebar Terminals list flush like Pins. visual-review: PASS (terms-flush.png).
- **2026-08-29** — **ADR-0020** accepted: PiCode Desktop provisions the distro from Windows (tray `.exe` at the WSL boundary, `picode provision` inside). Supersedes ADR-0018, whose "Alternatives considered" had rejected both the logon task and linger; the linger objection is answered by enabling it on install and never disabling it. Carries a preservation contract (`~/.picode` untouched, tmux survives, `wsl.conf` line-merged after backup, cert reissued only near expiry). Docs only — no code, no user-visible change, so no CHANGELOG entry.
- **2026-08-29** — Chat file card: click a path → closable card in the thread; Open in tab → same `#/file/a/…` as the terminal. Split FilePane removed. visual-review: PASS (file-chat-card.png, file-chat-tab.png).
- **2026-08-29** — File preview track 1: png, pdf, md, audio, video, glb/gltf (model-viewer). visual-review: PASS (file-png-preview.png, file-md-preview.png, file-pdf-preview.png, file-audio-preview.png, file-video-preview.png, file-glb-preview.png, file-bin-raw.png, file-bin-gone.png).
- **2026-08-29** — Ctrl+click on a terminal path eats mousedown/mouseup so tmux SGR (`<16;NaN;NaNm`) does not land in the Pi composer.
- **2026-08-29** — Terminal Ctrl+click also matches bare `App.jsx` / `foo.js` (not only paths with `/`).

## Recent activity (archived 2026-08-29)

- **2026-08-29** — File V1.1: `.svg` / `.mmd` open Preview \| Raw \| Save (one chip group). Empty: “Nothing to preview.” Bad mermaid: “Can't draw this diagram.” visual-review: PASS (file-svg-preview.png, file-svg-raw.png, file-mmd-preview.png, file-mmd-empty.png, file-mmd-error.png).
- **2026-08-29** — ADR-0019: Ctrl/Cmd+click a path in the terminal opens `#/file/…` on the tab strip (text only). http(s) → browser. visual-review: PASS (file-tab.png, file-tab-gone.png, file-tab-outside.png). Chat FilePane unchanged.
- **2026-08-29** — Reconnect covers fast restarts: health `bootId` + WS-close kick → tab reloads itself (proved: restart → page age reset). Shift+Enter trusted-key E2E: bytes `[27;2;13~` only, multiline composer confirmed; earlier "submit" reading was a bad pane-tail interpretation. keypress guard added for browsers that fire it after a canceled keydown.
- **2026-08-29** — Shift+Enter three-layer fix: ground truth via session JSONL proved the user's Windows Chrome submits (a stray `\r` after the canceled keydown). `termDataFilter` on `term.onData` swaps/drops that `\r` (120ms window). Keydown tracker lives on the xterm textarea (capture).
- **2026-08-29** — Terminal Ctrl+C copies if text is selected (Warp / Windows Terminal); else interrupt. Keys list in Preferences → Terminal; `/hotkeys` shows them.
- **2026-08-29** — Settings **Keys**: search / Add (press a key) / Reset writes `~/.pi/agent/keybindings.json`. visual-review: PASS (settings-keys.png, settings-keys-empty.png, settings-keys-listen.png).
- **2026-08-29** — Terminal scroll: tmux `mouse on` so xterm.js enables mouse tracking (#426); screen `margin-right` so the scrollbar is clickable (#1751); stop capturing wheel (that blocked SGR). visual-review: PASS (term-scrollbar.png, `enable-mouse-events` true).
- **2026-08-29** — Tabs flush (no left pad); drag to reorder. Terminal: debounce resize 150ms; wheel sends SGR to TUI transcript; Shift+Enter newline (Preferences → Terminal). visual-review: PASS (tabs-reorder.png, first tab gap 0).
- **2026-08-29** — Terminal: ResizeObserver fits the pane as soon as the sidebar/split changes (not only window.resize). Wheel capture: scroll xterm, else PageUp/PageDown to the TUI. visual-review: PASS (term-resize.png; screen 1000→1168px when sidebar 244→80).
- **2026-08-29** — Terminal wheel on a TUI (Pi) pages/scrolls the view instead of the composer cursor. Stopped stretching `.xterm-screen` to 100% (broke hit-testing + scrollbar). visual-review: PASS (term-wheel-scrolled.png).
- **2026-08-29** — Composer chips joined into button groups (session+New; provider/model/thinking/mode/kind). visual-review: PASS (composer-chip-groups.png, composer-chip-groups-run.png).
- **2026-08-29** — Machine menu: Theme/Layout are one left-aligned capsule (radio semantics); entries have icons. Fixed segment svg shrinking to 0 (`flex:none` + `flex:auto`). visual-review: PASS (user-menu.png).
- **2026-08-29** — Composer Expand + More menu; sketch Cancel/Insert 28px. visual-review: PASS (composer-pages.png, composer-more.png, composer-sketch.png).
- **2026-08-29** — Preferences → Terminal is two columns (controls + live xterm preview). visual-review: PASS (pref-terminal.png).
- **2026-08-29** — `make deploy` / `picode deploy` = repo → systemd. `picode update` = GitHub release for a normal user.
- **2026-08-29** — `picode install` / `uninstall` (systemd --user, ADR-0018). No Windows task.
- **2026-08-28** — Rename terminals (pencil / double-click). visual-review: PASS (term-rename.png, term-renamed.png). Terminal Light/Dark in Preferences (pref-term-theme.png), independent of the app theme.
- **2026-08-28** — File pane header compact (24px); Save, Close, Expand last. visual-review: PASS (file-header.png, file-expanded.png, overlayAudit ok).
- **2026-08-28** — File pane left edge resizes (persists). visual-review: PASS (file-resize.png).
- **2026-08-28** — File pane syntax colors follow GUI light/dark. visual-review: PASS (file-theme-light.png, file-theme-dark.png, overlayAudit ok).
- **2026-08-28** — Track E4 turn file names. visual-review: PASS (turn-files.png, turn-files-open.png, overlayAudit ok).
- **2026-08-28** — Track E3 Keep/Undo on edit diffs. visual-review: PASS (hunk-keep.png, hunk-kept.png, hunk-undo.png, overlayAudit ok).
- **2026-08-28** — Track E2 CodeMirror Save. visual-review: PASS (file-edit.png, file-discard.png, overlayAudit ok).
- **2026-08-27** — Track E1 open file beside chat. visual-review: PASS (file-open.png, file-gone.png, file-binary.png, overlayAudit ok).
- **2026-08-27** — ADR-0015 + Track E plan (file pane, Save, hunk Keep/Undo). Herdr study: runtime peer, not the editor bar.
- **2026-08-27** — Voice V1 owner dogfood: Chrome Windows mic works. No V1.1 unless a quality gap shows up.
- **2026-08-27** — Track D5 package updates (badge + Update). npm only. visual-review: PASS (pkg-update.png, pkg-update-menu.png, overlayAudit ok).
- **2026-08-27** — Track D4 Prompts timeline (Now + continue in a new session). visual-review: PASS (prompts-tree.png, prompts-continue.png, overlayAudit ok).
- **2026-08-27** — Track D3 `@` mentions (agent / skill / file). visual-review: PASS (at-mention.png, at-mention-empty.png, overlayAudit ok).
- **2026-08-27** — Track D2 cost on the session chip. visual-review: PASS (session-cost.png, overlayAudit ok).
- **2026-08-27** — D1 loop: hash apply no longer depends on selectedId (tabs vs URL freeze). visual-review: PASS (agent-url-switch.png, overlayAudit ok).
- **2026-08-27** — Track D1 `#/agent/<id>`. visual-review: PASS (agent-url.png, agent-url-gone.png, overlayAudit ok).
- **2026-08-27** — Plan: `@agent` / `@skill` are mentions (context in this prompt). Agents talking to each other is the broker item, later.
- **2026-08-27** — Track C2 draft persistence (text + kind per agent). visual-review: PASS (chat-draft-reload.png, overlayAudit ok).
- **2026-08-27** — Follow-up queue: Edit / Remove on the bubble (held until idle). visual-review: PASS (chat-queue-edit.png, overlayAudit ok).
- **2026-08-27** — Track C3 visible queue: Send while busy/waiting; prompt→follow-up; abort drops Steer. visual-review: PASS (chat-queued.png, overlayAudit ok).
- **2026-08-27** — Track C1 waiting: extension dialogs in the conversation (`POST /api/agents/{id}/ui`). Notify is a toast. visual-review: PASS (chat-waiting.png, chat-ask-cancel.png, overlayAudit ok).
- **2026-08-27** — Track C roadmap: waiting → queue → draft (`docs/design/conversation-control-roadmap.md`). Next-roadmap gaps recorded there (rewind, cost, `#/agent/<id>`, extra `@`, hunk accept, broker, ACP, IDE). Backlog unchanged.
- **2026-08-27** — Cleared MCP dogfood: `picode-dogfood-*` out of Claude/Codex/Grok; Notion/Linear tokens out of the keyring; `~/.picode/mcp-auth` job files gone. Claude keeps `context7`.
- **2026-08-27** — MCP layout: Servers (list or “No servers yet.”) then Add with **Use from…** as first chip. Dropped “Using Claude Code”. visual-review: PASS (mcp-named.png, mcp-use-from.png, overlayAudit ok).
- **2026-08-27** — Signed-in rows do not also say “Signed in”; **Sign out** is the state. visual-review: PASS (mcp-signout.png).
- **2026-08-27** — MCP **Sign out** forgets the keyring login (Off does not). Linux secret-tool. visual-review: PASS (mcp-signout.png, mcp-signout-confirm.png, overlayAudit ok).
- **2026-08-27** — After Sign in the toast fired but the row still said Sign in: overlay ends on callback before keyring write, and Idle list does not poll without a running agent. Optimistic **Signed in** + retry load. visual-review: PASS (mcp-linear-signed.png).
- **2026-08-27** — Linear Sign in was sending `redirect_uri=https://mcp.linear.app/callback` (Linear homepage, overlay hung). Now we register a localhost callback. Authorize URL verified `127.0.0.1` + refuse non-loopback.
- **2026-08-27** — Add GitHub still saves the row; Copilot DCR failure is a toast, not a failed Add. Dogfood Linear in Claude (`picode-dogfood-linear`). visual-review: PASS (mcp-linear-list.png).
- **2026-08-27** — MCP overlay ends on callback (not after ~40s keyring write). Signed-in OAuth rows say **Signed in** and hide Sign in. visual-review: PASS (mcp-signed-in.png). Signed-in detection is Linux secret-tool (WSL); macOS/Windows later.
- **2026-08-27** — MCP Sign in: GUI opens the authorize tab (Pi/PowerShell does not) so the PiCode callback can `window.close()` like Claude/Codex. visual-review: PASS (mcp-signin-auto.png, overlayAudit ok).
- **2026-08-27** — MCP callback page is the same PiCode success HTML as providers (close + return to `#/mcps`), not the adapter's "return to pi".
- **2026-08-27** — MCP Sign in no longer uses `/mcp-auth` paste UI. Headless `authenticate()` (callback only) writes a result file; overlay ends when that file is ok. visual-review: PASS (mcp-signin-auto.png).
- **2026-08-27** — MCP Sign in opened two Notion tabs (GUI window.open + Pi open on WSL). GUI no longer opens a tab. visual-review: PASS (mcp-signin-auto.png, overlay unchanged).

## Recent activity (archived 2026-08-27)

- **2026-08-26** — MCP Sign in is automatic (no paste): GUI opens the tab, adapter callback auto-closes it, overlay ends on success notify. visual-review: PASS (mcp-signin-auto.png).
- **2026-08-26** — MCP Sign in overlay stayed up after Notion Authorization Successful (callback did not unblock `/mcp-auth` UI). Now notify success finishes the wait; Paste is always there.
- **2026-08-26** — MCP Sign in opened two Notion tabs (GUI window.open + Pi open()). GUI no longer opens a second. visual-review: PASS (mcp-signin-wait.png, overlay unchanged).
- **2026-08-26** — MCP Sign in waits for the browser callback (no paste by default). Paste address is fallback. visual-review: PASS (mcp-signin-wait.png).
- **2026-08-26** — MCP Sign in uses a short pi when no agent is running. Add/On on OAuth starts Sign in. visual-review: PASS (mcp-signin-short.png). Dogfood: Notion login opened with no agent.
- **2026-08-26** — Sign in is a button next to On (not a SIGN-IN tag). Off has no Sign in. visual-review: PASS (mcp-signin-btn.png).
- **2026-08-26** — MCP Sign in starts `/mcp-auth` (RPC + paste dialog). visual-review: PASS (mcp-signin.png).
- **2026-08-26** — MCP GET redacts env/header values. Dogfood in Codex/Grok left for later.
- **2026-08-26** — Diff cards in conversation use JetBrains Mono + Fira Code (same as source fences). visual-review: PASS (chat-diff-font.png).
- **2026-08-26** — Remove on a Use-from overlay no longer unmasks the import (stays Off). Dogfood Claude servers deleted. List is A–Z. visual-review: PASS (mcp-list-az.png).
- **2026-08-26** — MCP list is A–Z by name (live poll no longer reshuffles).
- **2026-08-26** — Conversation source uses JetBrains Mono + Fira Code ligatures. visual-review: PASS (chat-code-font.png).
- **2026-08-26** — MCP live status (Idle / Live / Failed / Sign in). visual-review: PASS (mcp-live-idle.png).
- **2026-08-26** — MCP Add More: env / headers / Sign in / Token. visual-review: PASS (mcp-add-more-url.png, mcp-add-more-env.png, mcp-add-more-error.png).
- **2026-08-26** — MCP card: agent icon + name at top; scope pill is This agent again. visual-review: PASS (mcp-this-agent.png).
- **2026-08-26** — Use from is a tree (app → servers). Pick per server; Off the rest. visual-review: PASS (mcp-use-from-tree.png).
- **2026-08-26** — Dogfood MCP in Claude/Codex/Grok globals (`picode-dogfood-*`). Use from lists counts. visual-review: PASS (mcp-use-from-dogfood.png).
- **2026-08-26** — Import renamed **Use from…** (mirror, not copy). Empty hosts hidden. visual-review: PASS (mcp-use-from.png).
- **2026-08-26** — MCP Import is a picker, not import-all. visual-review: PASS (mcp-import-pick.png).
- **2026-08-26** — MCP B3 Import (adapter `imports` only). visual-review: PASS (mcp-import.png).
- **2026-08-26** — Agent context is the first line in the MCP/Packages card, not under the title. visual-review: PASS (mcp-card-ctx.png).
- **2026-08-26** — MCP/Packages name the agent (title + pills). Sidebar click from a pane opens that agent. visual-review: PASS (mcp-named.png).
- **2026-08-26** — MCP empty redesigned (one line + Open packages). UI skills now load-before-JSX; visual skip = quality-gate FAIL. visual-review: PASS (mcp-blocked.png).
- **2026-08-26** — MCP manager: list/add/toggle/remove on adapter files (machine / folder / this agent). B3 import next.
- **2026-08-26** — Composer `!cmd`: RPC bash in the agent folder, inline block + Stop. Track A done.
- **2026-08-26** — Toolbar clip attaches workspace files (image → chip, else `@path`); reads stay inside the folder.
- **2026-08-26** — Click a composer/chat thumbnail to preview the image.
- **2026-08-26** — Composer image chip 64px; `@` list has a filter and hides dotfiles until typed.
- **2026-08-26** — Composer paste/drop images (RPC `images[]`). Next: `!`.
- **2026-08-26** — Composer `@` file picker (agent cwd). Next: images, then `!`.
- **2026-08-26** — Roadmap: composer files then MCP (`docs/design/composer-mcp-roadmap.md`). Auth/llama parked.
- **2026-08-26** — Restore walks the same job overlay (stop agents → db → pins → sessions) and asks to reload.
- **2026-08-26** — Reveal uses host Explorer on WSL. Backup job steps animate. Motion + optimistic UI is a gate.
- **2026-08-26** — Backup schedule is explicit (off until Schedule). Preferences split into tabs.
- **2026-08-26** — Folder picker on WSL: Home / C: / E: chips; accepts `C:\\` paths.
- **2026-08-26** — Backup V1: Preferences folder + interval/retention. `VACUUM INTO` + hardlink snapshots. Restore refuses newer schema.
- **2026-08-26** — Decision table is a quality gate when conditions change the outcome (AGENTS.md).
- **2026-08-26** — Delete agent/workspace: confirm may offer session + work-folder purge (last occupant only). All workspace agents stopped first.
- **2026-08-26** — Pin V3 sketches (Excalidraw, lazy). Blank or annotate image.
- **2026-08-26** — Pin V2.1 TipTap editor (markdown on disk).
- **2026-08-26** — Pin attachments V2 (image + file). Sketch/Excalidraw is V3.
- **2026-08-26** — Pin studio is a route (`#/pins/new` / `#/pins/:id`). List stays in the sidebar.
- **2026-08-26** — `npm:pi-agent-browser-native` + skill shrunk. Fix: IconPin crash (blank app).
- **2026-08-26** — Pins V1: title, tags, markdown body. Flat list, `+` on title bar.
- **2026-08-26** — Sidebar tabs Agents / Pins. QR → user menu.
- **2026-08-26** — Conversation polish: blockquotes + ```diff``` hunks. Images + Mermaid + KaTeX + tables.
- **2026-08-26** — Source **Run** (bash/python/js/go) in the agent cwd. Not a browser sandbox.
- **2026-08-26** — Conversation source renderer (fenced code: lang + copy + highlight).
- **2026-08-26** — Codex DID reply; chat ignored `message_end` (no `text_delta`). Free-agent Sessions listed the wrong folder (0).
- **2026-08-25** — Chose `npm:pi-web-search` (this machine). Chat search cards from tool sources. Full packages-cycle dogfood deferred.
- **2026-08-25** — Packages **This agent** + optional isolate (skip machine/folder). `pi -e` every start / every session.
- **2026-08-25** — llama GUI installer reverted. Setup stays in `www/guide/llama.md`; dialog is URL + link. Continuity → backlog.
- **2026-08-25** — `/llama` dialog on the agent (not Providers redirect). HF download, wait-for-load, default `127.0.0.1:8080`.
- **2026-08-25** — Slash TUI 24 all **ui**. Skills/templates picker. `/export` `/import` `/share` (gist) `/hotkeys` `/changelog`.
- **2026-08-25** — ADR-0013 multi-account vault. OAuth re-login updates same account; click name to rename.
- **2026-08-25** — Device-code OAuth: Copilot / Kimi / xAI. Claude + Codex stay loopback.
- **2026-08-25** — SW never caches `index.html`. Sidebar tree: 12px indent, shared chevron|icon|label grid.
- **2026-08-25** — Providers GUI: no docs copy (guide is public). Voice V1 shipped; owner dogfooding.
- **2026-08-25** — Relicensed PolyForm Noncommercial. Public docs VitePress on Pages (no in-app iframe).

## Recent activity (archived 2026-08-25)

- **2026-08-25** — Public docs: VitePress Markdown, new-tab slash hints
  (`/commands#{id}`). No in-app docs/iframe.
- **2026-08-25** — Public docs (`www/` → GitHub Pages). `#/docs/{cmd}`
  iframes them (later removed). `/tree` click remains fork (pi#8645).
- **2026-08-25** — ADR-0011: sidebar **Free** vs **Workspaces**, many agents
  per folder (own model). Selected entity is the agent id.
- **2026-08-25** — Packages: This machine vs This workspace (`-l`).
  Session/`pi -e` still deferred (This run). ADR-0010 amended.
- **2026-08-25** — Optimistic UI is a bar (`docs/philosophy.md` §7).
  Packages gallery uses layout skeletons on first load; refetch keeps
  last hits. Blank wells while fetching are FAIL.
- **2026-08-25** — Voice V1: dictation + Grok-style voice composer
  (`docs/design/voice-mode.md`). Web Speech API, no Realtime fork.
- **2026-08-24** — Desktop/mobile shells in one Vite app (`web/src/desktop`,
  `web/src/mobile`). Boot picker by viewport or `?desktop=1`/`?mobile=1`.
- **2026-08-24** — Phone QR: prefer current LAN IP. Drawer lists lan/tailnet
  targets; QR only for addresses on the cert.
- **2026-08-24** — Adopted AgentDeck's product-benchmark set: Cursor +
  t3code + paseo. Studies in `docs/benchmarks/`.
- **2026-08-24** — Route split: Settings = PiCode system; `#/providers`
  and `#/mcps` are first pi-facing routes.
- **2026-08-24** — Agent provider/model/thinking moved onto the agent tab
  bar (auto-save).
- **2026-08-24** — **ADR-0009 + M3 v1**: catalog from pi, auth via `/login`
  in the TUI, MCP status-only, agent config flags on start, exclusive lock.
- **2026-08-24** — M2 closed: inline diffs and Ctrl+K palette. Accept/reject
  hunks deferred.
- **2026-08-24** — **ADR-0008**: UI React + Vite + Tailwind. Source in `web/`.
- **2026-08-24** — Dock: `[hidden]{display:none !important}`; single pane
  owned by the active agent tab (no inner tab strip).
- **2026-08-24** — IDE-style agent tabs; dock opens only by explicit action.
- **2026-08-24** — Exploratory QA (agent-browser): real-pi prompt stream,
  port rebind 8445→8446→8445, theme sweep.
- **2026-08-24** — `agent-browser` skill added (agentdeck port).
- **2026-08-24** — User-menu popover SyntaxError; Cache-Control no-cache +
  `cmd/uicheck`. JS-syntax gate mandatory after app.js edits.
- **2026-08-24** — ADR-0007 shipped: HTTPS default, port rebind, server.json.
- **2026-08-24** — UI redesign after owner feedback (conversation-hero,
  tool pills, rounded composer, terminal dock).
- **2026-08-24** — M2 core shipped (ADR-0006): rpc + runtime + delivery,
  mode-switch, /ws/agent, agent panel. Verified against real pi.
- **2026-08-24** — ADR-0005 shipped: SQLite store, schema v1, migrations,
  legacy JSON import.
- **2026-08-23** — UI copy de-documentarized, Vercel-style user menu,
  settings route, live statusbar. M1 visually validated by owner (PASS).
- **2026-08-23** — CI `-race` term-bridge shutdown race; single-owner pty.
- **2026-08-23** — M1 complete: screenshot tooling, tmux, WS↔PTY, terminal
  grid, ADR-0004.
- **2026-08-23** — Language policy: English is the repository language.
