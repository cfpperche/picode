# Pi Settings pane — refinement proposal

Status: **proposed** (not implemented). Study and measurements:
[../benchmarks/2026-09-12-cli-settings-ux.md](../benchmarks/2026-09-12-cli-settings-ux.md).
Keeps every row of [cli-native-settings.md](cli-native-settings.md) and
[cli-settings-recovery.md](cli-settings-recovery.md) true — this changes how
the pane is arranged, not what it reads, writes or refuses. No ADR for the
layout; the one API field it wants (`reset`) extends ADR-0101's own patch
contract and updates `docs/architecture/cli-settings.md`.

## Goal

One layer at a time, its own file named, and the keyboard map out of the
settings scroll. Today the pane is 6014 px in a 1000 px viewport, the same six
controls appear twice (Global and Workspace), a third layer hides below them,
and 75 % of the page is 89 key rows that belong to the machine, not to the
layer you are editing.

## Target shape

```
Pi        Settings ······················································   ← pane tabs (unchanged)
┌ Settings │ Keys ┐                        ← sub-tabs (packages `pkg-tabs`)
│ Edit:  (This machine)(QA)(Atlas)         ← one layer at a time
│ Writes ~/.pi/agent/settings.json         ← the file this layer owns
│
│ Auto-compact                     [switch]
│ Steering                         [select]
│ Follow-up                        [select]
│ Defaults                         [provider ▾][model ▾][thinking ▾]
│ Scoped models                    [claude-* ×][Add]
│ Tools                            [✓ read][✓ bash]…
└──────────────────────────────────────────
```

- **Layer switcher** — `pkg-scope` radios: *This machine*, the workspace, the
  agent. Same control language as the Packages install bar and the pi-roles
  editor (`PackagesConfig.jsx`), same place: above the form, labelled.
- **One body** — the selected layer's rows only. The agent layer keeps its
  three rows (Model, Tools, Checklist) and its `PATCH /api/agents/{id}` path,
  including the existing restart-on-tool-mode-change behavior.
- **The layer's file is named** under the switcher (`rep.global.path`,
  `rep.project.path`) — the honest answer to "where does this go", and the
  reason the pi-roles editor prints it.
- **Keys is a sub-tab** — `pkg-tabs` (Installed | Marketplace in Packages):
  *Settings* holds the layers, *Keys* holds today's section unchanged
  (category groups, filter, Add → press a key, Reset). It is machine-wide
  (`/api/pi-keys`), so it is not wrapped in a layer.
- **`settings-ctx` line removed** — the embedded `PageFrame` context
  ("Atlas"), glued 0 px under the tab rule, goes the way of Packages and
  Connectors; the pane owns the 12 px gap.
- **The route names the layer** — `#/clis/pi/settings?agentId=…&layer=project`
  (and `&view=keys`), so a reload, a bookmark or a Palette command lands on
  the same layer. `cliSettingsHash`/`cliSettingsLocation` gain both fields;
  the existing `focus=scoped-models` deep link selects the layer that holds
  the row before scrolling.

## Provenance and undo (the part that makes layers honest)

The API already returns `layer.has.{key}` per key; the UI throws it away.

- A row **set in this layer** gets the modified marker (a 2 px accent bar at
  the row's left, VS Code's "colored bar on the left of the setting") and a
  compact **Use inherited** action beside its control.
- A row that **is not set** shows the resolved value with a muted
  `From This machine` / `From this folder` and no marker. It stays editable —
  editing creates the override.
- **Use inherited** removes exactly that key from this layer
  (`PUT /api/pi-settings {layer, patch: {reset: ["steeringMode"]}}`), so the
  parent applies again. This is the one backend gap: `pisettings.Patch` only
  sets fields today and `merge()` never deletes. The change is small
  (`internal/pisettings/pisettings.go`, its tests, `docs/architecture/cli-settings.md`)
  and confined to the layer file Pi already owns.
- Without it (owner's call, question 3) P1 ships the markers read-only and
  reset stays "edit the JSON" — the pane then *says* which value cannot be
  undone.

## Findability

- A `Filter settings…` field over the rows, drawn when the layer shows more
  than 8 rows, and a **Modified** toggle that shows only rows set in this
  layer (VS Code's search bar and `@modified`).
- The agent layer (3 rows) gets neither — progressive disclosure.

## Phases

| Phase | Deliverable | Files |
|---|---|---|
| **P0 — IA** | sub-tabs (Settings / Keys), layer switcher + one layer body, layer file named, ctx line gone, `layer`/`view` on the route | `web/desktop|mobile/src/components/PiSettings.jsx`, `CliSettings.jsx`, `web/shared/domain/cliSettings.js`, both `styles/app.css`, both `components/agent-clis.css` |
| **P1 — provenance** | `has`-driven markers, `Use inherited`, `Patch.Reset`, tests | both `PiSettings.jsx`, `internal/pisettings/pisettings.go` + tests, `internal/server/pi_settings_test.go`, architecture doc |
| **P2 — findability** | filter + Modified toggle; `focus` selects its layer | both `PiSettings.jsx`, `styles/app.css` |
| **P3 — mobile + docs** | mobile sub-tabs/switcher (the quick sheet keeps `agentOnly`), docs-site guide, changelog | `web/mobile/src/components/PiSettings.jsx`, `docs-site/guide/*.md` |

P0 alone takes the machine layer from 6014 px to ~600 px and puts the 89 key
rows behind one tab.

## Decision table

| Conditions | Action | Coverage |
|---|---|---|
| Canonical URL, no agent | Machine layer only; no pills for layers that do not exist; Keys sub-tab available | `qa-cli-settings.mjs` (extended) |
| Agent in the route | Switcher shows machine + workspace + agent; default layer = the agent | extended script |
| Workspace untrusted | Workspace pill selectable; body keeps "This folder is not trusted." + Trust action; no write | existing untrusted row |
| Agent layer selected | Model/Tools/Checklist rows and the existing PATCH/restart behavior | `qa-cli-settings-recovery.mjs` (existing matrix) |
| Row set in this layer | Accent bar + **Use inherited** | new browser assertions + unit tests |
| Use inherited | `reset: [key]` deletes only that key, other keys survive, parent value returns, toast confirms | new `pisettings` unit tests + browser |
| Row inherited | Resolved value + muted source; editing creates the override | extended script |
| Read fails / context refresh fails / save fails | Unchanged: the recovery plan's rows stay true (editor kept, writes blocked, rollback) | `qa-cli-settings-recovery.mjs` |
| `?layer=project` without a workspace, or unknown value | Fall back to the machine layer; never write another layer | unit test on `cliSettingsLocation` + browser |
| `?focus=scoped-models` | Select the layer that holds the row, then scroll to it | extended script |
| `?view=keys` | Keys tab; no layer switcher | extended script |
| Narrow pane / mobile | Switcher wraps; sub-tabs scroll; overlays contained (audit) | screenshots 1024 + 390 |

## Verification

1. `make ci-scoped`, then `make close`; `make ci` on `main`.
2. Extend `scripts/qa-cli-settings.mjs` (one script, not a third): layer
   switcher present per route, one body only, `layer` in the hash after a
   click and after a reload, Keys tab hides the switcher, provenance markers
   match `has`, Use inherited round-trip, untrusted workspace unchanged.
   `scripts/qa-cli-settings-recovery.mjs` must stay green untouched.
3. Measure in the DOM: machine layer ≤ ~900 px, no duplicate labels in one
   body, every control at `--ctl-h`, `__picodeOverlayAudit()` ok in each
   capture.
4. Screenshots read (`var/screenshots/cli-settings-ux/`): machine, workspace,
   agent, keys, untrusted, 1024 px, mobile 390.
5. Go: `pisettings` tests for `reset` (only that key, unknown key refused,
   other keys preserved, live-apply still fires for the patched fields).

## Out of scope

- New knobs, the settings schema, or editing `settings.json` in the pane.
- The agent runtime semantics (restart on tool-mode change) and the
  live-apply path.
- The pi-roles editor, the mobile quick sheet's content, the Packages and
  Connectors panes.
- Keyboard capture itself (only its address changed).

## Open questions (owner)

1. **Default layer with an agent in the route**: the agent (recommended — that
   is the link's intent) or *This machine* (VS Code's default)?
2. **Keys**: a sub-tab (recommended, packages already use that pattern inside
   a pane) or one collapsed section in the same scroll?
3. **`Use inherited`**: approve the small `reset` addition to the settings
   patch (recommended) or ship provenance read-only and defer the undo?
