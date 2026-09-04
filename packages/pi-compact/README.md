# pi-compact

Opt-in compaction policy for [pi](https://pi.dev). Compacts earlier than
Pi's window-edge default, summarizes with a cheap model (thinking off),
and overlays `/compact` so the command users already know still works.
Works in the pi TUI and, via the same commands and dialogs, in PiCode.

**This directory is MIT.** The rest of the PiCode repository is
PolyForm Noncommercial — see the [repo licensing](../../LICENSING.md).

## Install

```bash
# this workspace
pi install -l /absolute/path/to/picode/packages/pi-compact

# this run only
pi -e /absolute/path/to/picode/packages/pi-compact
```

npm publish is not wired yet. Local path is the supported M1 install.

## Defaults (no config file required)

Unlike `pi-roles`, a missing file does **not** disable the package.
Loaded means:

| Knob | Default |
|---|---|
| Early trigger | **100 000 tokens or 50% of the window**, whichever first |
| Floor | 32 000 tokens (short sessions stay untouched) |
| Summarizer | Flash → Flash-Lite → Haiku → session model |
| Thinking | `off` |
| Recent tail | Pi's own cut (`keepRecentTokens`, default 20k) |
| Pi overflow compact | still on |

Create `<workspace>/.pi/compact.json`, or let `/compact edit` write it.

PiCode agents also get `PI_COMPACT_AGENT=<agent-id>`. Then the workspace
file is overlaid by `<cwd>/.pi/compact/<id>.json` (keys in the overlay
win). A `pi` you start yourself in a terminal has no overlay unless you
export that env.

```json
{
  "enabled": true,
  "atTokens": 100000,
  "atPercent": 0.5,
  "floorTokens": 32000,
  "model": "google/gemini-2.5-flash",
  "fallback": ["anthropic/claude-haiku-4-5"],
  "thinking": "off",
  "instructions": "",
  "cooldownTurns": 2
}
```

`model` is `provider/id`. `atTokens` / `atPercent` of `null` turns that
knob off. Schema: [`compact.schema.json`](compact.schema.json). Unknown
keys are ignored.

## Commands

This package **replaces** Pi's `/compact`. Empty `/compact` still
triggers compaction (`ctx.compact()` — same pipeline, including overflow
recovery). Sole-word subcommands:

| Command | Effect |
|---|---|
| `/compact` | Compact now |
| `/compact <instructions>` | Compact now, with that focus |
| `/compact edit` | Wizard: when, model, fallback, thinking, save to |
| `/compact model` | Change the summarizer chain |
| `/compact on` / `/compact off` | Session lock for the *early* trigger |

`/compact edit the summary` is instructions, not the wizard — a reserved
word is a subcommand only when it is the entire argument.

In **PiCode chat**, `/compact` is PiCode's own summarize action (it still
runs this package's summarizer via `session_before_compact`). `/compact
edit` lives in the agent TUI, or edit `.pi/compact.json` by hand. Early
trigger does not need a command.

Under `PI_COMPACT_AGENT`, `/compact edit` and `/compact model` end with a
**Save to** select: *this agent* writes the overlay, *workspace* writes
`.pi/compact.json`. Cancel (Esc in the TUI) aborts the whole flow; `‹
back` walks one field.

`/compact off` does not persist and does not disable Pi's overflow
compact or this package's summarizer on a manual `/compact`.

## What it does not do

- Drop Pi's recent-token tail (the live reads/edits).
- Turn off `compaction.enabled` in Pi settings.
- Rewrite the session JSONL beyond the `CompactionEntry` Pi already writes.

## Tests

```bash
npm test
```

`npm test` type-checks against the pinned pi types (`tsc`) and runs the
pure decision table (`test/logic.test.ts`) plus handler tests with narrow
fakes (`test/extension.test.ts`). Zero runtime dependencies.
