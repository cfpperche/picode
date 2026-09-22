### Added

- Omp terminals now show their todo plan as the sidebar checklist — the
  current step on the agent card, the full list in the disclosure — the
  way Pi and Claude Code already do. The new `packages/omp-checklist` is
  an installable omp package: it watches omp's native `todo` tool and
  mirrors the committed phases to the checklist routes at whatever scope
  you install it — Global, This workspace, or an agent's package list.
  Omp registers no new tool and nothing is gated.
