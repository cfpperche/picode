# Screenshots — committed visual evidence

This directory holds **curated screenshots that document the UI's evolution**.
GitHub renders them inline in PRs and Markdown — they are how visual review
verdicts (`.pi/skills/visual-review`) become auditable evidence.

## Rules

- Naming: `<milestone>-<view>-<state>.png` — e.g. `m1-termgrid-home-empty.png`,
  `m2-agentpanel-diff-streaming.png`.
- Only curated evidence lands here (working captures stay in gitignored
  `var/screenshots/`). Keep files <500KB; PNG only.
- Every PR that changes user-visible UI should include before/after
  screenshots when feasible, plus the `visual-review:` verdict line.
- The accumulated history of this directory is the visual changelog of
  PiCode — treat it as documentation, not attachment dumping.

## Current state

Connector catalog tabs: `connector-tabs-catalog.png` shows the fixed Catalog
Custom card plus presets; `-custom-dialog.png` the moved server form;
`-host-added.png` / `-mobile-added.png` the Claude Code tab with an imported
server's Added state (desktop and mobile).
Inspector rail (ADR-0078): `inspector-*` (dark unless named `-light`, 1440×900
plus the 1200 and 700px layouts) covers Changes with counts, a change opened
as a center Diff tab, Files with the light theme and the empty filter, the
empty This-agent scope, the blocked terminal with Follow, the non-git folder,
the no-anchor line, the More menu and the shrunk/narrow layouts. All were read;
overlay and row audits passed. The data is the seeded docs fixture; the
acceptance groups are recorded in `inspector-qa.json`. `inspector-pr-*` adds
the PR tab (ADR-0078 phase 2): the open pull request card, the no-PR line with
**Create in terminal**, the terminal with `gh pr create --fill` pre-typed and
unsubmitted, the not-logged-in and no-remote lines. Those states come from a
scripted `gh` first on the fixture's PATH (`FAKE_GH_MODE_FILE` selects `ok`,
`none`, `unauth`, `noremote`; `FAKE_GH_JSON` is the `pr view` answer); the
runner drives it when `PICODE_QA_GH_MODE_FILE` names that file.
`inspector-git-fetch-typed-*` and `inspector-commit-*` add the Git actions
menu (ADR-0078, Commit stage 1): Fetch typed into an idle terminal of the
folder, the commit form refusing an empty message, the form with its command
preview, and the prepared commit sitting unsubmitted in the prompt while
`gitstatus` still reports the four changes; the branch chip reads `main ↑1`
from the seeded upstream. `inspector-run-*` adds the run-when-idle mode
(stage 2): the note that a second terminal is running `sleep` and the command
was prepared instead (the toast steps left of the rail), the commit form in
"Run in terminal" wording, and the terminal after the commit ran with
`gitstatus` at zero changes before the runner restores the seed.

Integrations (ADR-0075): `integrations-desktop-*` covers empty webhooks,
missing adapter, validation overlay, failed delivery/retry and the connector
catalog. `integrations-mobile-*` covers installed package metadata, the scrolled
catalog/form and webhook editing. Read screenshots and final overlay/row audits
passed. All data belongs to an isolated daemon; real HTTP/MCP results are recorded
in `docs/plans/integrations.md`, not inferred from these images.

Tool captures (ADR-0076): `tool-capture-desktop-*` (light, 1280×900) and
`tool-capture-mobile-*` (dark, 390×844) cover no capture, running capture,
blocked source, decode failure/Retry and the existing image viewer. All ten
images were read; overlay audits passed. The embedded docs fixture plus
`scripts/fixtures/tool-preview-init.js` supplies synthetic socket events and
successful history reads delayed by 1.6s, not aborted requests. This proves
host rendering/reconciliation, **not** a real browser-package emitter.
Mobile reconnect, rapid agent switches and final/late-update checks are local
`var/tool-preview/*.json` evidence. See the [plan](../plans/browser-preview.md)
for remaining integration acceptance and fixture setup. These ten captures
were refreshed against combined main during merge review. The mobile composer
now wraps rather than clipping Send; 320px evidence and audits are retained in
`var/capture-merge/` alongside the 390px state checks.

Working captures stay in `var/screenshots/`. Devices footer (list + centred Pair/Copy): `devices-foot.png`. Curated MCP: empty/blocked, Use from, Add More (url / env / error).
Provider Usage (ADR-0031): `usage-accounts.png` (Usage on each vault row),
`usage-windows.png`, `usage-empty.png`, `usage-error.png`, `usage-auth.png`,
`usage-openrouter.png`. `usage-loading.png` is V1 skeleton.
Mobile v2 (ADR-0044, 390×844): `mobile-now-empty.png`, `mobile-now-needs-you.png`
(a blocking inbox question, a live terminal and an agent under Running, today,
results), `mobile-now-dark.png`, `mobile-agent-ask.png` (pushed agent screen with
the open select card), `mobile-inbox-respond.png` (stacked inbox, reply form),
`mobile-work-workspaces.png` (Work → Workspaces card with agents and the
+ Agent / + Terminal row), `mobile-terminal.png` (pushed terminal with the key
bar), `mobile-more.png`.
Git graph worktrees (ADR-0073): `adr0073-gitgraph-worktrees-dirty.png`
(three dirty worktrees — owner with "this worktree", a detached checkout and a
sibling branch — with the sibling's file list open, dark),
`adr0073-gitgraph-branch-picker.png` (local branches marked with the worktree
they are checked out in).
Dashboard v2 (ADR-0042): `dashboard-v2-7d-light.png` (tiles + daily chart +
first breakdown row), `dashboard-v2-breakdowns-dark.png` (tokens, tools,
reliability, top sessions), `dashboard-v2-empty.png` (Today with no activity).
Transient TUI replies (ADR-0059): `adr-0059-burst-receiving.png`,
`adr-0059-burst-processing.png`, `adr-0059-burst-failed.png`,
`adr-0059-burst-terminal-unavailable.png` (explicit TUI restart), and
`adr-0059-burst-mobile-receiving.png` (390×844).

Docker (ADR-0065): `docker-desktop-empty.png`, `docker-desktop-blocked.png`,
`docker-desktop-error.png`, `docker-desktop-dark.png`, and
`docker-mobile-confirm.png` cover the shared App host with synthetic data
and one disposable QA container.

Docker groups (ADR-0066): `docker-groups-desktop-light.png`,
`docker-groups-desktop-dark.png`, `docker-groups-mobile-light.png` and
`docker-groups-mobile-long.png` show project disclosure and readable long
identities. `docker-groups-empty.png`, `docker-groups-blocked.png`,
`docker-groups-error.png` and `docker-groups-mobile-confirm.png` cover the
non-happy states. All use synthetic Docker data; real inventory grouping,
keyboard navigation, search and fold persistence were also exercised.

Docker width correction: `docker-width-desktop.png` (1920px) and
`docker-width-mobile.png` (390px) show cards filling the app canvas up to
its padding. Both use synthetic data. Empty/blocked/error and confirmation
states were also recaptured and read in `/tmp/picode-docker-width-*.png`.

Docker v3 (ADRs 0067/0068): `docker-v3-*.png` covers full-width groups, resource
consumers, reviewed project/procedure plans, running/completed/failed jobs,
empty/blocked resources and mobile confirmation/monitoring. These are synthetic
Engine fixtures. Separate real Engine QA used only disposable project members,
an image, a network and a volume; all were removed afterward.

Agent CLIs v1 (ADR-0069): `cli-v1-*.png` covers the desktop catalog, terminal
inventory, empty/missing-executable/network-error states, mobile light/dark
layouts and a named Stop confirmation. `cli-v1-adopt-terminal.png` shows a
manual Claude terminal adopting launch defaults without restarting its process.
All terminals and configuration changes belong to an isolated QA daemon;
native installed CLIs opened without model turns. Screenshots were read and
settled menu/dialog audits passed. QA terminals and their private launch files
were removed afterward; native CLI data was retained.

Agent CLIs v2 (ADR-0070): `cli-v2-desktop-defaults.png` and
`cli-v2-desktop-injection.png` show visible defaults and shared launch details.
`cli-v2-profile-editor.png`, `cli-v2-terminal-pending.png` and
`cli-v2-workspace-menu.png` cover copied profiles, pending settings and contextual
creation. `cli-v2-desktop-blocked.png` / `cli-v2-desktop-error.png` cover recovery;
`cli-v2-discard-confirm.png` / `cli-v2-restart-confirm.png` cover named decisions.
`cli-v2-mobile-dark.png` / `cli-v2-mobile-discard.png` show readable paths and
the mobile discard sheet without empty scroll space. Screenshots were read;
settled overlay audits passed. QA used isolated fixture data and disposable
terminals; version checks did not start conversations. This is not evidence
of every native vendor lifecycle event.

Independent web applications (ADR-0072): `split-mobile-*.png` covers the initial
mobile copy, empty Inbox/Changes, unavailable Docker, form validation, chunk
load recovery and an always-sheet wide preview. `split-desktop-small-*.png`
shows responsive desktop navigation at 390px. All were read, using an isolated
embedded fixture; overlay audits passed. Package-chunk failure was injected
and Try again recovered after the interception was removed. No model turns
were started. Physical PWA installation/push delivery remains separate QA.

### Mobile review corrections (2026-09-05)

Synthetic embedded fixture, no model turns. Screenshots were read; settled
sheet/popover audits passed. Light phone and wide dark sheet; save failure
rolls back controls, and closing Settings preserves the draft and attachment.

- [Agent Settings](mobile-review-settings-light.png)
- [Save failure](mobile-review-settings-error.png)
- [Retained draft and image](mobile-review-draft-retained.png)
- [Tool selector above the sheet](mobile-review-mode-selector.png)
- [Wide dark sheet](mobile-review-settings-wide-dark.png)

File Tree v2 (ADR-0074): `filetree-v2-editor-dark.png`,
`filetree-v2-editor-light.png`, `filetree-v2-diff-dark.png` and
`filetree-v2-image.png` show one local detail panel for content and changes.
`filetree-v2-unsaved-dialog.png`, `filetree-v2-save-error.png`,
`filetree-v2-conflict.png`, `filetree-v2-removed.png` and
`filetree-v2-root-changed.png` cover draft protection and recovery.
The empty, clean-changes, blocked, unsupported and tree-error captures show
their next actions. Narrow, preview-narrow, compact and stacked captures cover
900, 720, 550 and 390px desktop windows. Screenshots were read after correcting narrow
preview controls; overlay and toolbar-boundary audits passed.

`filetree-v2-qa.json` records all 15 browser acceptance groups. Reproduce with
`scripts/qa-filetree-v2.mjs` against a private `picode-docs-fixture -addr
127.0.0.1:18746` instance, using the current UI build. Set `PICODE_QA_BASE` to
that address, `PICODE_QA_CDP` to the CDP URL of a dedicated browser session,
and `PICODE_PLAYWRIGHT_MODULE` to an installed Playwright module. Start the
agent-browser session with `AGENT_BROWSER_NO_AUTO_DIALOG=1` so the test can
observe and dismiss the native unsaved-changes dialog. The runner checks the
synthetic workspace identity before writing fixture files and removes its
disposable terminal afterward. Public docs use a separate unmodified fixture;
`docs-shots.mjs` now closes only its own browser session.


llama.cpp manager (ADR-0080): `llama-desktop-*` and `llama-mobile-*` capture
empty, blocked, authentication error, download, model replacement, model list,
light theme and Providers entry states. All 16 images were read; overlay
and alignment audits pass (`llama-qa.json`). Reproduce with
`scripts/qa-llama.mjs`, `PICODE_QA_BASE` pointing to a private docs fixture and
`PICODE_PLAYWRIGHT_MODULE` pointing to an installed Playwright module.
`PICODE_QA_CHROME` optionally selects an existing Chromium executable. The
runner refuses non-fixture workspaces and mocks only llama API responses;
real model/runtime acceptance remains pending in the plan.
