# Sidebar checklist expansion — click a card's plan line, read the whole plan

Owner request (2026-09-06): the sidebar checklist line should expand on
click into the full list — ☑ on finished steps, a braille spinner on the
step being executed, ☐ on the rest. This plan covers that feature; the
companion refinements shipped with it (muted counter, absent renders as
silence) are in ADR-0082. Status: **planned, awaiting owner go-ahead**.

## Why the data is already there

No server work is needed. The full list already reaches both shells:

| Owner | Source | Live updates |
|---|---|---|
| Managed agent | `checklists[agentId].items` (App state: one boot fetch of `/api/checklists` + `agent.checklist` feed events) | yes, per step change |
| Terminal pi | `term.checklist.items` (folded into `GET /api/terminals` + `terminal.checklist` events, ADR-0081) | yes |

`currentStep`, `countDone` and `GLYPH` (`web/shared/domain/checklist.js`)
already encode the projection rules; lists are bounded at 30 items
(extension) / 50 (store). An absent or unknown checklist shows no line at
all (ADR-0082), so the disclosure only ever opens on a real list.

## Design

| Area | Design |
|---|---|
| Component | `ChecklistDisclosure({ id, check })` in `WorkspaceRows.jsx`, used by `AgentRow` and `TermRow` in place of `ChecklistLine`. `ChecklistLine` stays the pure one-line renderer used by the terminal pane strip (the strip never expands — the pane is already the detail view) |
| Collapsed line | Unchanged: `(2/4) current step`, muted counter (ADR-0082), one-line ellipsis, `title` holds the full step |
| Toggle | The line becomes a real `<button>` (native control; no homemade widget): `aria-expanded`, `aria-controls="chk-<ownerId>"`, Enter/Space work. Click stops propagation so the row's own select does not fire; keyboard focus ring comes from the shared focus style |
| Expanded list | `<ul id="chk-<ownerId>">` under the line: one `li` per step — ☑ completed (dimmed, `--text-secondary`), `PiSpinner` on the in-progress step (the existing braille spinner: same frames ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏, 80 ms, `role="status"`), ☐ pending. Step text clamps to two lines (`-webkit-box`), full text in `title`. Glyph column is `aria-hidden`; the list is plain list semantics |
| Motion | `grid-template-rows 0fr → 1fr` + opacity, 150 ms ease-out; `prefers-reduced-motion: reduce` renders the open state with no transition (same guard the file already uses) |
| Live while open | Nothing to do: items arrive through the feed the row already subscribes to; a completed step re-renders ☐→spinner→☑ in place. No new fetch, no polling (ADR-0048 rule holds) |
| State | Expanded/collapsed is component state, **not persisted** — it is a glance, not a preference; sidebar collapse/remount resets it. Documented choice, revisitable |
| Density | Collapsed rows stay ≤32px; the expanded block adds ~18px per step at 11.5px with 2px gaps, indented to the text column (aligned after the glyph slot) |
| Mobile | Out of scope for phase 1 (rows are single-line context there); the domain helper below makes it a small follow-up |

### Pure logic first (node:test)

`checklistRows(items)` in `web/shared/domain/checklist.js`: maps the raw
items to `{ key, glyph, status, text, current }` rows (same validation
tolerance as `checklistItems`), so the component stays dumb and the
status→glyph/spinner mapping is unit-tested — including the "no in-progress
step" case (all pending / all done render no spinner).

### Benchmark adaptation

Cursor's activity feed: compact collapsible rows that reveal detail in
place, never a wall of text; Linear's deference (the agent's work is the
hero, chrome recedes). PiCode adds: the collapsed line is already the
status, so expansion only ever adds the *remaining* steps — the current
step's text is not repeated inside the list.

## Decision table (implementation must cover every row)

| Conditions | Observable result |
|---|---|
| Line absent / checklist unknown | No line, nothing clickable (ADR-0082) |
| Line present, collapsed | One muted line; button `aria-expanded=false` |
| Click line | List opens in place; agent/terminal row does NOT navigate; `aria-expanded=true` |
| Click again / Escape while focused | List closes; row still does not navigate |
| Step completed while open | Its row flips spinner → ☑ in place, no fetch |
| Current step changes while open | Spinner moves to the new step in place |
| All steps complete while open | List shows all ☑; collapsed line reads `(n/n)`; both live |
| Owner removed while open | Row unmounts with the list (feed `agent.deleted`/`terminal.deleted`) |
| 30-step list, narrow sidebar | Each step clamps to two lines, full text in `title`; no horizontal scroll |
| Keyboard: Tab to line, Enter | Same as click, focus ring visible |
| Terminal pane strip | Never expandable (stays `ChecklistLine`) |

## Acceptance

- `node --test` on `checklistRows` (statuses, unknown status, empty, no current).
- Browser QA on the isolated daemon: expand agent card and terminal card,
  screenshot collapsed/open, spinner visible on the in-progress step, live
  flip while open (POST a step change mid-expansion), click does not
  navigate away, keyboard toggle, `overlayAudit ok`, reduced-motion spot
  check. visual-review verdict in the handoff.

## Out of scope / follow-ups

- Mobile expansion (touch target exists; needs its own row rhythm).
- Expanding from the dashboard (no agent rows there today).
- Persisting open state across reloads.
- Actions on steps (check/uncheck by hand) — the list is the agent's plan,
  not a shared todo; the agent owns it (ADR-0055 contract).

## Addendum: compact-view alignment (2026-09-06, owner-approved)

The owner flagged two defects in the shipped disclosure while reviewing
screenshots of agents and terminals: the `(x/n)` counter's parentheses read
as terminal vocabulary, not list UI, and the line sat flush at the card's
left edge — out of the 31px text column every other sub-line (folder,
branch) uses. A web-UX pass (VS Code's chat todo list, Cursor's Agents
window, Linear/GitHub sub-issue counters, PatternFly's progress guidance)
confirmed the fix: a fixed-width counter at the row's end, not a prefix.

Shipped (desktop `WorkspaceRows.jsx` + `app.css`, mobile `AgentRow.jsx`):

- No parens anywhere the counter appears (sidebar line, disclosure line,
  terminal pane strip, mobile sub-line).
- `.ws-check` and `.ws-check-disclosure` share `.ws-context`'s 31px margin,
  so the plan line, its expanded list, and the folder/branch line all sit
  in one column. The terminal pane strip (no identity-mark gutter) resets
  the margin to flush width.
- The disclosure's chevron reuses `.ws-chev` (the workspace group header's
  own affordance) instead of introducing a new one.
- `:focus-visible` on the disclosure button uses the row's own
  `box-shadow` selection style, replacing an inset outline that read as a
  text field.
- A finished plan (`position === total`, every item completed) gets an
  `is-done` class that dims it to the same weight as a completed step in
  the expanded list, so it stops reading as live activity.
- The plan line moved above the folder/branch line on both `AgentRow` and
  `TermRow`, matching the identity → activity → location rhythm the mobile
  row already used.
- Mobile's sub-line reads `5/8 · text` instead of `(5/8) text`.

Browser QA on an isolated scratch daemon (`.worktrees/checklist-compact-
refine`, seeded via `POST /api/agents/{id}/checklist` and `POST /api/
terminals/{id}/checklist` for an in-progress plan, a fully-completed plan,
and a terminal plan): collapsed alignment, expand/collapse by click,
real-keyboard Tab reaching the button with the accent focus ring, the
`is-done` dimming on the completed plan, the terminal pane strip flush to
the pane, and dark theme — all confirmed on desktop, and the mobile
sub-line format on `/mobile/`. `overlayAudit` clean, no console errors.
No server or domain change; `checklistRows`/`checklistLine` untouched.
