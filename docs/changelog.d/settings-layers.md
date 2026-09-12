### Changed

- **Agent CLIs → Settings edits one layer at a time.** A labelled switcher
  (**Edit: This machine / workspace / agent**) picks the layer, the line under
  it names the file that layer writes, and the body shows only that layer's
  rows — instead of stacking the same six knobs for the machine and the
  workspace and hiding the agent's below them. The chosen layer and the tab
  travel on the URL (`?layer=…&tab=…`), so a reload or a bookmark lands on the
  same view; an agent link opens that agent's layer.
- Rows say where their value comes from: **Set here** (with an accent bar) when
  this layer sets it, **Pi default** or **From This machine** when it inherits.
  A row this layer sets offers **Use inherited**, which hands the value back to
  the layer below instead of freezing a copy of it.
- **Keys** is a sub-tab of its own: the whole 89-row keyboard map keeps its
  filter and its Add-then-press-a-key flow, without sitting at the end of the
  settings scroll. The global settings pane went from ~6000 px of scroll to
  under 1000 px.

### Fixed

- A settings value can now be *un-set*: `PUT /api/pi-settings` accepts
  `patch.reset[]` and removes exactly those keys from the layer's file
  (other keys, including ones PiCode does not know, stay). An unknown name is
  refused rather than reported as inherited. After a reset, a running agent
  adopts the effective compaction/steering/follow-up values instead of keeping
  the override that just left the file.
- A malformed `~/.pi/agent/settings.json` no longer disables the agent's own
  Model/Tools/Checklist fields on the mobile quick sheet.
