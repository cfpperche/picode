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

M0 placeholder only — first real captures land with M1 (terminal grid).
