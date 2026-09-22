### Changed

- **One packages pane for every CLI.** Which pane opened used to depend on the
  CLI's name: Pi got the rich one, the other eight a simpler one. There is one
  pane now — it asks the CLI's own engine what that CLI can do and draws exactly
  that, so each control appears only where the CLI has the mechanism behind it
  and a verb the CLI lacks reads as one line in the CLI's own words. Pi's
  controls, wording, transcript and gallery are unchanged.
- **A CLI's marketplace and updates, in the same pane.** Where a CLI has its own
  update command, opening the pane compares its installed list with the CLI's
  own catalog, and **Update** appears only on the rows that catalog says are
  behind; a catalog PiCode cannot read leaves every row unmarked with the
  reason. Where the CLI manages marketplace sources, the **Marketplace** tab
  lists them with **Add source** beside per-source **Update** and **Remove**.

### Added

- **Omp agents can carry their own packages.** The **This agent** install target
  used to be Pi's alone. It now works for an Omp agent too: those entries are
  PiCode's own list on that agent, and Omp receives them at its next start
  (`-e`). "Only this agent's packages" is offered for Pi and Omp — the two CLIs
  whose launch can be handed a per-agent list.
- **One architecture file for packages.** `docs/architecture/packages.md`
  describes the engine, the report the pane renders, the agent layer and the
  measured vendor facts; the older split description is gone, and the
  architecture index, the routes reference and the public guides point at it.
