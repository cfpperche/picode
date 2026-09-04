# Compact earlier

Pi waits until the session is almost out of room before it summarizes.
On a large model that can be hundreds of thousands of tokens too late,
and the summary itself can fail.

Install `packages/pi-compact` from the PiCode repository so an agent
compacts **sooner**, with a cheaper model, and overrides the summarizer
Pi's own `/compact` uses. Guide for install targets: [Packages](/guide/packages).

```sh
pi install /path/to/picode/packages/pi-compact
```

Without it, nothing changes. With it, the package stays **dormant until you
configure it** — no defaults, ever. While there is no config file, Pi's own
compaction runs exactly as before, and the status line reads
`compact: not configured · /compact-edit`.

Once you save a config (`/compact-edit`, `/compact-model`, or a hand-written
`.pi/compact.json`), the package:

- compacts when the session hits **100 000 tokens** or **half the window**,
  whichever comes first (never below 32 000) — always at the end of a run,
  never mid-flight
- summarizes with `gemini-3.6-flash`, then Haiku, then the session model —
  thinking off; if a link fails, the next one takes over
- keeps the recent work Pi already keeps (the last ~20 000 tokens)
- leaves Pi's last-resort compact on

Any of these knobs can be changed or removed in the file — the wizard shows
exactly what will be saved.

## Commands

| You type | What happens |
|---|---|
| `/compact` | Pi's own compact-now (uses this package's summarizer when configured) |
| `/compact-edit` | When, which model, fallback, thinking, where to save |
| `/compact-model` | Only the summarizer chain |
| `/compact-off` | Pause early compact for this session (configured only) |

Why the dash? Pi's terminal owns `/compact` itself — its input loop handles
`/compact …` before any extension command, so the wizard lives one dash away.
Bare `/compact` keeps working exactly as Pi built it.

In PiCode chat, `/compact` is PiCode's summarize button. It still uses
this package's cheaper summarizer. To change when and how, open the
agent terminal and run `/compact-edit`, or edit `.pi/compact.json`.
Early compact happens on its own.

## Per agent

PiCode sets `PI_COMPACT_AGENT` on every agent. `/compact-edit` then asks
**Save to**: this agent, or the whole folder. A `pi` you start yourself
in a terminal has no overlay unless you export that environment variable.

A config file is what turns the package on. `/compact-edit` writes
`.pi/compact.json` (or `.pi/compact/<agent>.json`); delete the file to go
back to Pi's stock behavior.
