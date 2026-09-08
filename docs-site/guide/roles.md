# Model roles

One agent, one model — until the work changes. A text-only refactor does
not need the vision flagship, and a plan deserves max thinking. Switching
by hand every time breaks the flow.

`pi-roles` routes automatically. It is an **optional pi package — an
extension, not part of PiCode core**. PiCode never switches models on its
own: routing exists only where the package is installed, and stays
dormant until you write a `.pi/roles.json`. Your per-agent model choice
always survives — the package never overrides the model an agent starts
with.

Install `packages/pi-roles` from the PiCode repository. Guide for install
targets: [Packages](/guide/packages).

```sh
pi install -l /path/to/picode/packages/pi-roles
```

While there is no config file, nothing happens — the session model stays
whatever it is.

## What it does

| Role | Trigger |
|---|---|
| `default` | Auto mode, text-only input. **Not** applied at session start — that would fight the agent's own model. |
| `vision` | Auto mode when the input has attached images or a `.png`/`.jpg`/… path in the text |
| `plan` | `/plan` locks it **and** appends plan-mode instructions (no edits until you approve) |
| custom | `/role <name>` or the `/roles` picker |

`/auto` returns to content routing. A lock always wins over content.

## Commands

| You type | What happens |
|---|---|
| `/auto` | Content routing |
| `/vision` | Lock vision |
| `/plan` | Lock plan + system prompt |
| `/role <name>` | Lock a preset (`auto` accepted) |
| `/roles` | Pick a configured role |
| `/roles edit` / `add` / `remove` | Manage presets, with a Save to choice |
| `/roles clear` | Delete a whole roles file (confirmed) |

## Where it runs

| | What you get |
|---|---|
| **Pi TUI** (terminal) | The commands and auto routing in your own terminal |
| **PiCode chat** | Same routing; a composer chip shows the active role; the wizard asks **Save to**: this agent or the whole folder |
| **PiCode core** | Nothing — the daemon never routes; it only tells the package which agent it is and renders the chip |

## Config and scopes

The code is global (installed once into pi); **the config is per file,
never per machine**. Two layers, the agent on top:

| Layer | File | Who reads it |
|---|---|---|
| Workspace | `<workspace>/.pi/roles.json` | every agent in the folder |
| Agent overlay | `<workspace>/.pi/roles/<agent>.json` | that one agent; its slots win |

Either file turns routing on. A minimal workspace file:

```json
{
  "builtin": {
    "default": { "model": "zai/glm-5.3", "thinking": "medium" },
    "vision": { "model": "xai/grok-4.6", "thinking": "high" },
    "plan": { "model": "zai/glm-5.3", "thinking": "max" }
  },
  "custom": [
    { "name": "redteam", "model": "kimi-coding/k3", "thinking": "low" }
  ]
}
```

`model` is `provider/id`; `thinking` is optional (omit it and the switch
leaves thinking alone). Unset builtin slots fall through. All keys are
documented in
[`roles.schema.json`](https://github.com/cfpperche/picode/blob/main/packages/pi-roles/roles.schema.json);
delete the file to go back to one static model.

## How you know it worked

In PiCode, the composer shows a chip with the active role
(`vision — xai/grok-4.6 · high`); a lock survives a session restart. In
the TUI, the `/roles` picker lists each role with its model and thinking.
No chip and no picker entries — the package is not installed or the
config file is missing.
