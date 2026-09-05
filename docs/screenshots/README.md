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
