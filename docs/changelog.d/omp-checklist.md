### Added

- Omp terminals now show their todo plan as the sidebar checklist — the
  current step on the agent card, the full list in the disclosure — the
  way Pi and Claude Code already do. A checklist-mirror extension rides
  the same wrapper injection as omp's terminal-state extension and POSTs
  the native `todo` tool's committed phases to the terminal's checklist
  routes; omp registers no new tool and nothing is gated.
