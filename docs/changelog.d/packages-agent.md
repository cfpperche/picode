### Added

- **Omp agents can carry their own extensions.** The packages pane's "This
  agent" scope used to be Pi's alone; it now works for an Omp agent too. An
  entry added there is PiCode's own list on that agent, and the agent's next
  start passes it to Omp as `-e` — so the extension loads only for that agent,
  and "only this agent's packages" switches off what Omp would otherwise
  auto-load. Nothing is written into Omp's own configuration.
