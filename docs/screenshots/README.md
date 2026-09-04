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
