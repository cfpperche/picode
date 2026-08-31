# pi-roles

Opt-in model roles for [pi](https://pi.dev). Three automatic behaviours plus
named presets. Works in the pi TUI and, via the same commands and dialogs, in
PiCode.

**This directory is MIT.** The rest of the PiCode repository is
PolyForm Noncommercial — see the [repo licensing](../../LICENSING.md).

## Install

```bash
# this workspace
pi install -l /absolute/path/to/picode/packages/pi-roles

# this run only
pi -e /absolute/path/to/picode/packages/pi-roles
```

npm publish is not wired yet. Local path is the supported M1 install.

## Configure

Create `<workspace>/.pi/roles.json`, or let `/roles add` / `/roles edit` write it.
Missing file = routing is dormant until the file exists.

PiCode agents also get `PI_ROLES_AGENT=<agent-id>`. Then `/roles` reads the
workspace file and overlays `<cwd>/.pi/roles/<id>.json` (slots in the overlay
win). A `pi` you start yourself in a terminal has no overlay unless you export
that env. `/roles remove` in an overlay only deletes presets this agent added.

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

`model` is `provider/id`. `thinking` is optional; if omitted, thinking is left
alone on that switch. Unset builtin slots fall through (omp-style empty `—`).

Schema: [`roles.schema.json`](roles.schema.json). Unknown keys are ignored.

## What it does

| Role | Trigger |
|---|---|
| `default` | Auto mode, text-only input. **Not** applied at session start — that would fight PiCode's per-agent `--model`. |
| `vision` | Auto mode when the input has attached images or a `.png`/`.jpg`/… path in the text. `/vision` locks it. |
| `plan` | `/plan` locks the model **and** appends plan-mode instructions (no edits until the user approves). |
| custom | `/role <name>` or `/roles` picker. |

`/auto` returns to content routing. Locks win over content.

A path that merely *mentions* `screenshot.png` is treated as vision. Prefer
attaching the image; the path heuristic is a fallback.

## Commands

| Command | Effect |
|---|---|
| `/auto` | Content routing |
| `/vision` | Lock vision |
| `/plan` | Lock plan + system prompt |
| `/role <name>` | Lock a preset (`auto` is accepted) |
| `/roles` | Pick a configured role |
| `/roles edit [name]` | Provider → model → thinking, then save |
| `/roles add [name]` | Create a custom preset |
| `/roles remove [name]` | Delete a custom preset |

## Tests

```bash
npm test
```

Zero runtime dependencies. Tests cover the routing decision table in
`test/logic.test.ts`.
