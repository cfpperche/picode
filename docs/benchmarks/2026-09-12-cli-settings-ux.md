# Study: the Pi Settings pane — three layers stacked, and 4.5 k px of keybindings

- **Date:** 2026-09-12
- **Sources:** owner screenshot of `#/clis/pi/settings` (2026-09-12 11:41,
  2559×1599, ~150 %) and a live read of the same pane on a `qa-scratch.sh`
  instance (1600×1000, trusted workspace, agent in the route) with the DOM
  measured; live docs fetched 2026-09-12 —
  [VS Code, "User and workspace settings"](https://code.visualstudio.com/docs/configure/settings).
  In-house bars: [benchmarks.md](../benchmarks.md) (one page width, density,
  the UI copy rule), [plans/cli-native-settings.md](../plans/cli-native-settings.md)
  and [plans/cli-settings-recovery.md](../plans/cli-settings-recovery.md)
  (the acceptance tables this must keep true), ADR-0101, ADR-0103,
  the pi-roles layer editor (`PackagesConfig.jsx`), and the two surfaces the
  owner just approved: the Packages install bar
  (`docs/handoff/2026-09-11-pkg-target-bar.md`) and
  [Connectors](2026-09-12-connectors-ux.md).
- **Scope:** the information architecture of `#/clis/pi/settings` in the
  desktop app (and its mobile twin where it differs). The APIs, the files
  (`~/.pi/agent/settings.json`, `<folder>/.pi/settings.json`, the agent
  fields), trust, live-apply and the recovery behaviour stay as they are.

## What VS Code does (the same problem, solved)

| Pattern | Receipt | Why it matters here |
|---|---|---|
| **Scope is a tab, not a stack** | "Select the User tab in the Settings editor"; "Select the Workspace tab" | User and Workspace settings are edited one layer at a time; the editor never renders both forms. |
| **Scope of one value is visible** | "Not all user settings are available as workspace settings" | The editor tells you where a value *can* live before you change it. |
| **Modified marker per row** | "You can identify settings that you modified by the colored bar on the left of the setting" | Provenance is a property of the row, not of the page. |
| **Reset per row** | "The gear icon alongside the setting … options to reset a setting to its default value" | An override can be removed, not only changed. |
| **Search filters the list** | "When you search using the search bar, the Settings editor filters the settings" | A long settings list is searched, not scrolled. |
| **Keys are a different surface** | Keyboard Shortcuts is its own editor, not a section of Settings | PiCode currently appends 89 key rows to the settings scroll. |

## The problem, measured

Desktop, 1600×1000, trusted workspace with an agent, from the live pane:

| Measured | Value | Why it hurts |
|---|---|---|
| Whole pane (`#pi-settings-view` inside `.pane-view.cli-page`) | **6014 px** in a **1000 px** viewport | Six screens for the settings of one CLI. |
| Section **Keys** | **4533 px, 89 rows, 12 groups** | **75 %** of the page. It is machine-wide (`/api/pi-keys`, "This machine … Same map as Pi") yet it is the tail of a per-layer form. |
| Section **Global** | 414 px, 6 rows (`Auto-compact, Steering, Follow-up, Defaults, Scoped models, Tools`) | – |
| Section **Workspace** | **431 px, the same 6 labels again** | The same form twice in one scroll, resolved values only: a workspace row that inherits looks exactly like one that overrides. |
| Section **Agent** | 202 px, 3 rows (Model, Tools, Checklist) | A third layer, again stacked, written through a different API (`PATCH /api/agents/{id}`) behind the same-looking form. |
| Layer provenance in the DOM | none — `[data-layer]` has no styling and `LayerKnobs` renders resolved values (`resolveLayer(layer, parent)`) | The API already returns `layer.has.{key}` (per-key provenance) and the UI throws it away. |
| Undo an override | impossible in the UI (`pisettings.Patch` only sets fields; `merge()` never deletes) | An inherited value that was once touched can be changed but never handed back to the parent. |
| First line of the pane | `settings-ctx` ("Atlas"), 0 px below the tab rule | The third surface with the same glued line (Packages and Connectors already shed theirs). |
| `?focus=scoped-models` | scrolls to the **global** row only (`id` is set for `prefix === "g"`) | The Palette shortcut points at one layer; with layers it must name the layer too. |

Untrusted folder: the Workspace section collapses to a 139 px notice
("This folder is not trusted." + one action) — the one state today that reads
well, because the layer says what it is and offers exactly one action.

## What we refuse

- **A settings search that invents a schema.** Pi's settings file is open
  (unknown keys are preserved); the pane edits the keys it understands. A
  search box filters *those* rows — it does not promise coverage of the file.
- **Editing `settings.json` inside the pane.** The JSON escape hatch is the
  file itself (Pi's own file, opened in the Files view); a second editor in
  the pane is a competing writer. VS Code can afford it; we have two writers
  already (PiCode and Pi).
- **Keyboard-shortcut capture beyond what exists.** The Add-then-press-a-key
  flow, the group filter and Reset already work; this study only asks where
  the section lives.
- **A second width or a new page** (ADR-0103): the pane keeps its geometry.

## What follows

The refinement proposal — one layer at a time, provenance per row, Keys as its
own sub-tab, and the one small API gap (`reset`) it needs — lives in
[../plans/cli-settings-ux.md](../plans/cli-settings-ux.md).
