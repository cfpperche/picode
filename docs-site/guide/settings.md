# CLI settings

Two different screens. Do not mix them.

| Hash | What | Writes |
|---|---|---|
| `#/clis/pi/settings` | **pi** JSON for the selected agent | `~/.pi/agent/settings.json` (global), `<cwd>/.pi/settings.json` (workspace, if trusted), and **Keys** (`~/.pi/agent/keybindings.json`) |
| `#/preferences` | **PiCode** chrome | theme, server port |

Open **Agent CLIs**, pick **Pi**, then the **Settings** pane. Composer `/settings` preserves the selected agent in the URL. A direct link without an agent opens Global and Keys.

Contextual links use `?agentId=<id>`. Old `#/settings` and mobile `#/more/settings` links redirect here. Workspace values override global values; agent values override both.


Workspace writes require the folder in pi's `trust.json`. Untrusted → 409; run `/trust` in the TUI.

## Related pi documentation

Canonical: [pi Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md)

| | pi TUI `/settings` | PiCode `#/clis/pi/settings` |
|---|---|---|
| File | **global only** | global + workspace + agent |
| Project file | `pi config` / `pi install -l` | workspace card, if trusted |
