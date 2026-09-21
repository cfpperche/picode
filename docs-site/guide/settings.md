---
description: Pi JSON for the selected agent, one layer at a time. Not Preferences.
---

# CLI settings

Two different screens. Do not mix them.

- **Where:** **Agent CLIs**, pick **Pi**, then the **Settings** pane. Composer `/settings` keeps the selected agent in the URL.
- **Not this:** not Preferences (theme, server port). That is PiCode chrome. This pane is pi JSON. Overview: [Configure](/guide/configure).

| Hash | What | Writes |
|---|---|---|
| `#/clis/pi/settings` | **pi** JSON for the selected agent | `~/.pi/agent/settings.json` (this machine), `<cwd>/.pi/settings.json` (workspace, if trusted) |
| `#/clis/pi/keyboard` | the keyboard map | `~/.pi/agent/keybindings.json` (one map per machine) |
| `#/preferences` | **PiCode** chrome | theme, server port |

The pane edits **one layer at a time**. **Edit** picks it — this machine, the
workspace, or the agent — and the line under it is the file that layer writes.
A row says where its value comes from: **Set here** when this layer sets it
(accent bar on the left), otherwise **Pi default** or **From This machine**.
A row this layer sets has **Use inherited**, which hands the value back to the
layer below instead of freezing a copy of it.

**Keyboard** is the pane next to Settings: the whole keyboard map of this
machine, one row per action. The **Filter keys** field narrows by action name,
group or chord, and **Find by key** takes a chord you press and shows the
actions that answer to it. The facets count what they filter — **Changed**, **Shared** (a key two
actions use; pi's contexts overlap on purpose), **Off** (unbound). One
**Add key** per row records the next chord you press; **Reset** returns that
row to pi's default, and **Reset all** returns every changed row at once. The
pane says which file it writes and which bindings *this machine* has: nine
actions differ on Windows and WSL, and the rows that use a chord a browser
keeps are labeled, because those never reach the terminal inside PiCode.

It has no layer — Pi keeps one map per machine — and the link remembers which
agent and layer you came from, so going back lands where you left.

Contextual links use `?agentId=<id>`; the pane adds the layer it is editing
(`?layer=global|project|agent`) to the URL, so a reload or a bookmark lands on
the same view. Old `#/settings` and mobile `#/more/settings` links redirect
here, and a `?tab=keys` link from the day the map was a sub-tab lands on the
Keyboard pane. Workspace values override machine
values; agent values override both.


Workspace writes require the folder in pi's `trust.json`. Untrusted → the
workspace layer says so and offers **Open agent to trust**; the same write
returns 409. Run `/trust` in the TUI.

## Related pi documentation

Canonical: [pi Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md)

| | pi TUI `/settings` | PiCode `#/clis/pi/settings` |
|---|---|---|
| File | **global only** | machine + workspace + agent, one layer at a time |
| Project file | `pi config` / `pi install -l` | the workspace layer, if trusted |
