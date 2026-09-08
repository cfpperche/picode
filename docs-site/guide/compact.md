# Compact earlier

Pi waits until the session is almost out of room before it summarizes.
On a large model that can be hundreds of thousands of tokens too late,
and the summary itself can fail.

`pi-compact` fixes that. It is an **optional pi package — an extension,
not part of PiCode core**. PiCode never compacts differently on its own:
the behavior below exists only where you install the package, and it
stays dormant until you write a config file. Uninstall it (or delete the
config) and every session is back to Pi's stock behavior, byte for byte.

Install `packages/pi-compact` from the PiCode repository. Guide for
install targets: [Packages](/guide/packages).

```sh
pi install /path/to/picode/packages/pi-compact
```

While there is no config file, nothing happens — Pi's own compaction
runs exactly as before, and the status line reads
`compact: not configured · /compact-edit`.

Once you save a config (`/compact-edit`, `/compact-model`, or a hand-written
`.pi/compact.json`), the package:

- compacts when the session hits **100 000 tokens** or **half the window**,
  whichever comes first (never below 32 000) — always at the end of a run,
  never mid-flight
- summarizes with your configured model first, then its fallback models —
  `gemini-3.6-flash`, then Haiku, then the session model by default —
  thinking off; if a link fails, the next one takes over
- keeps the recent work Pi already keeps (the last ~20 000 tokens)
- leaves Pi's last-resort compact on

## Where it runs

| | What you get |
|---|---|
| **Pi TUI** (terminal) | The commands below, early compact, the cheap summarizer on every compaction |
| **PiCode chat** | Same summarizer on `/compact` (PiCode's summarize button); the wizard runs in the agent terminal |
| **PiCode core** | Nothing — the daemon has no compact logic of its own; it only tells the package which agent it is (see below) |

## Commands

| You type | What happens |
|---|---|
| `/compact` | Pi's own compact-now (uses this package's summarizer when configured) |
| `/compact-edit` | Wizard: when, which model, fallback, thinking, where to save |
| `/compact-model` | Only the summarizer chain |
| `/compact-on` / `/compact-off` | Pause or resume early compact for this session (configured only) |

Why the dash? Pi's terminal owns `/compact` itself — its input loop handles
`/compact …` before any extension command, so the wizard lives one dash away.
Bare `/compact` keeps working exactly as Pi built it.

## Config and scopes

The code is global (installed once into pi); **the config is per file,
never per machine**. Two layers, the agent on top:

| Layer | File | Who reads it |
|---|---|---|
| Workspace | `<workspace>/.pi/compact.json` | every agent in the folder |
| Agent overlay | `<workspace>/.pi/compact/<agent>.json` | that one agent, wins over workspace |

Either file turns the package on. `/compact-edit` shows exactly what it
will save; in PiCode it asks **Save to**: this agent, or the whole folder.
A `pi` you start yourself in a terminal writes the workspace file.

A minimal hand-written config:

```json
{ "atTokens": 100000, "model": "google/gemini-3.6-flash", "thinking": "off" }
```

All keys — `atTokens`, `atPercent`, `model`, `fallback`, `thinking`,
`cooldownTurns` — are optional and documented in
[`compact.schema.json`](https://github.com/cfpperche/picode/blob/main/packages/pi-compact/compact.schema.json);
delete the file to go back to Pi's stock behavior.

## How you know it worked

When a compaction runs, the agent shows
`Summarizing with <model> (<thinking>)…`. The session JSONL records the
proof: the compaction entry carries `"from": "pi-compact"` plus the
`summarizer` and `thinking` that produced the summary. No such fields —
that compaction was Pi's own.
