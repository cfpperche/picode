# Settings

Two different screens. Do not mix them.

| Hash | What | Writes |
|---|---|---|
| `#/settings` | **pi** JSON for the selected agent | `~/.pi/agent/settings.json` (global) and `<cwd>/.pi/settings.json` (workspace, if trusted) |
| `#/preferences` | **PiCode** chrome | theme, server port |

Composer `/settings` opens `#/settings`. Depth is like skills: workspace beats global, agent beats both.

Workspace writes require the folder in pi's `trust.json`. Untrusted → 409; run `/trust` in the TUI.

## Related pi documentation

Canonical: [pi Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md)

| | pi TUI `/settings` | PiCode `#/settings` |
|---|---|---|
| File | **global only** | global + workspace + agent |
| Project file | `pi config` / `pi install -l` | workspace card, if trusted |
