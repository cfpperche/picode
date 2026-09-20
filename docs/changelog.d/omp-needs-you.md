### Fixed

- **Agent CLIs: Omp terminals now report Needs you.** The omp reporter only
  knew pi's event set, and omp never fires those — so an approval or a
  question waited in the terminal while the row said Working. The extension
  now listens to omp's own events (`tool_approval_requested`/`resolved`
  and the ask card's `tool_execution_start`/`tool_result`, verified against
  the 18.2.6 bundle), files the same Inbox item the other CLIs file, and
  only settles on `agent_end` when the run is really over (`willContinue`
  marks a mid-run turn).
