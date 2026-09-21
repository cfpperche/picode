# ADR-0176: packages-unification

- **Status**: proposed
- **Date**: 2026-09-21
- **Boundary**: process and protocol — one package subsystem replaces two
  engines that are chosen by *which CLI* a row is; the HTTP surface becomes one
  family, and the per-agent scope stops being a Pi-only branch.
- **Supersedes**: ADR-0167's engine clause ("Pi keeps `internal/pipkg` … and
  the others get what they expose"). Amends ADR-0010's Pi-only framing.

## Context

Two engines serve one concept. `internal/pipkg` reads Pi's own settings files
(`~/.pi/agent/settings.json`, `<ws>/.pi/settings.json`) plus the agent's list on
its row, mutates through `pi install/remove`, and carries the gallery, the
update check and the config descriptors (ADR-0099/0119). `internal/clipkgs`
holds one `spec` per guest CLI — a vendor argv builder per verb (nil meaning
"the CLI does not do this"), a tolerant reader over the vendor's `--json`, a
catalog, and mutation through the durable job lane (ADR-0087) — behind a
capability struct, with routes at `/api/cli-packages*` and a second pane.

That split was deliberate in ADR-0167 (one vendor at a time, no invented
semantics) and honest then. What changed is the axis: ADR-0160 made a CLI
runtime an agent, so "pi vs guest" no longer describes anything a user can
observe. The driver shape is already the house pattern elsewhere — ADR-0174
gives every guest CLI a declaration and a driver for its own key maps — which
leaves packages as the one subsystem still split by vendor identity. The cost
is visible: an extension installed for a workspace by a project setting appears
in Pi's pane and not in Omp's, although both load it — the reader is chosen by
the CLI, not by what the CLI's mechanism can do.

## Decision

Packages are one subsystem with one model and one interface, `internal/pkgs`:
`Row`, `Report`, `Caps`, `Scope`, and a `Driver` whose verbs are `List`,
`Available`, `Install`, `Remove` — with `Update`, `Toggle` and `Inspect`
reachable only where the driver's `Caps` declares them. Scope is a declared
capability (`machine | workspace | agent`), not a Pi-only branch; the agent
scope means "this CLI takes an injection at launch" and names the mechanism
(`-e` for Pi and Omp, the ADR-0154 MCP lane for Claude Code, Codex and
OpenCode). Mutations run through `internal/clijob` for every driver, and a
`Command` carries the exact line the user sees.

Pi becomes one driver among nine: its settings reader, validation, gallery,
update check and config descriptors move behind the interface unchanged, and
the eight guest specs keep their argv builders, parsers and fixtures. The HTTP
surface becomes `/api/packages*` with `cli` as a parameter (`pi` when absent);
`/api/cli-packages*` answers as an alias for one release. One pane renders
every CLI, its controls gated by `Caps` — the Pi pane's words ("This machine /
This workspace / This agent", "Open the Marketplace", "Configure", "Describe
config…") and the guests' ("Plugins go to", "Copy command", the vendor's
refusal) both survive.

## Consequences

Easier: a capability is declared once and the missing-control problem
disappears for every CLI at the same time — the Omp `extensions` list, invisible
today, becomes a row because its driver declares the reader. New CLIs land as
one driver file and a fixture. The routes and the panes stop being two of each.

Harder, and accepted as cost: the Pi pane is the richer one, so the model must
express "this CLI has no catalog" and "this CLI cannot toggle" as capabilities
rather than as hidden controls — a control that cannot work is still refused,
now by data instead of by a branch. The job lane becomes every mutation's shape,
which changes Pi from a synchronous call to a queued job unless the owner keeps
`Async` as a capability. Route churn is absorbed by one release of aliases.

If we are wrong: the parity suite (the non-regression table in
`docs/plans/packages-unification.md`) fails in slice 1, and the branch is
dropped — the two engines are untouched until that suite is green.

## Alternatives considered

- **Extend `clipkgs` with a `pi` spec and delete `pipkg`.** Refused: Pi's
  gallery, descriptors and agent list are not a vendor argv; the Pi driver is
  bigger than a spec row, and deleting `pipkg` is how the parity suite would
  fail silently.
- **Keep two engines and add a reader for Omp's `extensions` only.** Refused:
  it fixes the symptom and leaves the axis that produced it — the next CLI with
  a settings-level extension repeats the work.
- **One shared store for every CLI.** Refused (ADR-0167 stands): the CLIs' own
  files are the truth; PiCode keeps no package database.
