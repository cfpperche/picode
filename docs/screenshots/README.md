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

Working captures stay in `var/screenshots/`. Curated MCP: empty/blocked, Use from, Add More (url / env / error).
Provider Usage (ADR-0031): `usage-accounts.png` (Usage on each vault row),
`usage-windows.png`, `usage-empty.png`, `usage-error.png`, `usage-auth.png`,
`usage-openrouter.png`. `usage-loading.png` is V1 skeleton.
Dashboard v2 (ADR-0042): `dashboard-v2-7d-light.png` (tiles + daily chart +
first breakdown row), `dashboard-v2-breakdowns-dark.png` (tokens, tools,
reliability, top sessions), `dashboard-v2-empty.png` (Today with no activity).
