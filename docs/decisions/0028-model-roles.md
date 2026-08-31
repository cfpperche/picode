# ADR-0028: Model roles as an opt-in pi package

- **Status**: accepted
- **Date**: 2026-08-30

## Context

Users want per-task model routing (cheap model for text, a vision model for
images, a high-thinking model for plans) without paying the expensive model
on every turn. Oh-my-pi does this with a **fixed vocabulary of ten roles**
whose *semantics* live in that fork's harness; the user only fills in
model + thinking.

Pi has no role system. It does have `pi.setModel` / `pi.setThinkingLevel`,
an `input` event with `event.images`, and per-agent `--model` flags
(ADR-0009). PiCode has no plugin host in the React UI; extension dialogs
already surface as chat cards (`extension_ui_request`). Packages are a pi
surface (ADR-0010): opt-in, installed with `pi install`, not copied into
SQLite (ADR-0005).

The PiCode tree is PolyForm Noncommercial. A routing extension that plain-pi
users install cannot inherit that license without poisoning commercial use
of the package. Per-directory MIT is an established pattern and the
copyright holder can multi-license.

Cursor's custom modes (benchmark: [benchmark-cursor.md](../benchmark-cursor.md))
bind a named mode to a model plus instructions. We bind a small set of
**detectable moments** to a model plus thinking, and leave everything else
as a named preset. We only ship a role when we can detect its moment with
certainty in pi's extension API.

## Decision

PiCode ships **`packages/pi-roles/`**, a pi package:

1. **MIT in that directory only.** Root `LICENSE` stays PolyForm
   Noncommercial. Contributions to `packages/pi-roles/` are MIT (carve-out
   in [LICENSING.md](../../LICENSING.md)). The same rule applies to any
   future installable pi extension we author (including the M4 broker).
2. **Opt-in.** Nothing is installed by default. Users add it with
   `pi install -l <path>` or PiCode `#/packages` (ADR-0010). A missing
   `.pi/roles.json` leaves the extension dormant.
3. **Config is a workspace file** `<cwd>/.pi/roles.json`, schema at
   `packages/pi-roles/roles.schema.json`. PiCode does not copy it into
   SQLite. M2 (not this ADR) may add a GUI that edits that file.
4. **Three builtin behaviours** whose triggers we can detect today:
   - `default` — auto mode, text-only input. **Not** applied at session
     start (that would fight ADR-0009 `--model`).
   - `vision` — auto mode when the input has attached images or an image
     path in the text; `/vision` locks.
   - `plan` — `/plan` locks the model and appends plan-mode instructions.
   Unset slots fall through. Named **custom** presets cover everything
   else (`/role`, `/roles`).
5. **Default is a fallback, not a startup override.** Locks win over
   content. `/auto` returns to content routing.
6. **M3 slots** (`commit`, `task`, `advisor`) are added only when we
   ship the feature that gives them a moment. `tiny` is a non-goal
   (pi-core internals are not interceptable).
7. npm publish is a follow-up: a path-triggered workflow and a
   `pi-roles-v*` tag, so the package is not tied to the Go binary
   release train. Until then, install from the local path.

## Consequences

Easier: the routing decision table is unit-tested without pi; the same
extension runs in the TUI and in PiCode; the GUI (M2) has a file to edit
instead of a new store; commercial users of pi can install the package
under MIT.

Harder: two licenses in one tree. GitHub's badge shows the root license;
the package README and `packages/pi-roles/LICENSE` have to say MIT.
Schema changes after M2 need a coordinated bump. `vision` treats a
mention of `screenshot.png` as an image (documented).

If wrong: extracting the directory to its own repository later is cheap
(`git filter-repo`). Replacing MIT with PolyForm would break anyone who
installed the package; we will not do that.

## Alternatives considered

- **Own repository from day one.** Cleaner legal surface and release
  train. Lost to atomic schema↔GUI commits during M1–M2, one handoff,
  and the MIT carve-out covering the original objection.
- **Fixed roles only, no custom presets.** Rejected: `slow` / `designer`
  in omp are manual selectors, which *are* presets. Custom is that
  mechanism with a user-chosen name.
- **User-defined triggers (rules engine).** Rejected for M1: precedence,
  debugging, and docs cost for a terminal-averse audience. Revisit as M3
  if custom presets are not enough.
- **Copy roles into SQLite per agent.** Rejects ADR-0005/0010. Per-agent
  override via env at spawn is deferred (M2.1).
- **Fork oh-my-pi.** Rejected: we orchestrate pi, we do not fork it
  (architecture non-goal).

## Amendment (2026-08-31): cancel aborts, back is explicit

Dogfooding the PiCode chat stepper showed that cancel is ambiguous over
the one-select-at-a-time extension protocol: the extension used it to
mean "back one field", every GUI Cancel/Stop means "abort", and the web
client had to guess which follow-up select to auto-cancel — producing
stacked cards, dead flows, and steppers that lost earlier pills.

Decision: in every `pi-roles` select that has a previous field, going
back is an explicit `‹ back` **option** (a plain string in `options`, no
pi API change). Cancel — Esc in the TUI, Cancel/Stop in a GUI — always
aborts the whole flow. The PiCode stepper hides the `‹ back` row, makes
prior pills clickable only when the open select offers it, and answers
`‹ back` (repeatedly, if needed) to walk to the clicked field. Selects
that would offer one choice are still skipped; a field that was skipped
has no pill, so the walk can never dead-end.

Consequence for the TUI: Esc no longer steps back one field — it ends
the command; the `‹ back` row replaces it. One select at a time stands.
