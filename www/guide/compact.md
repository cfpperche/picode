# Compact earlier

Pi waits until the session is almost out of room before it summarizes.
On a large model that can be hundreds of thousands of tokens too late,
and the summary itself can fail.

Install `packages/pi-compact` from the PiCode repository so an agent
compacts **sooner**, with a cheaper model, and keeps the `/compact`
command you already know. Guide for install targets: [Packages](/guide/packages).

```sh
pi install /path/to/picode/packages/pi-compact
```

Without it, nothing changes. With it, and with **no config file**, the
package already:

- compact when the session hits **100 000 tokens** or **half the window**,
  whichever comes first (never below 32 000)
- summarize with Flash, then Haiku, then the session model — thinking off
- keep the recent work Pi already keeps (the last ~20 000 tokens)
- leave Pi's last-resort compact on

## Commands

| You type | What happens |
|---|---|
| `/compact` | Compact now |
| `/compact focus on auth` | Compact now, with that focus |
| `/compact edit` | When, which model, fallback, thinking, where to save |
| `/compact model` | Only the summarizer chain |
| `/compact off` | Pause early compact for this session |

`/compact edit the summary` still means “compact with those words” — a
subcommand is only the word **alone**.

In PiCode chat, `/compact` is PiCode's summarize button. It still uses
this package's cheaper summarizer. To change when and how, open the
agent terminal and run `/compact edit`, or edit `.pi/compact.json`.
Early compact happens on its own.

## Per agent

PiCode sets `PI_COMPACT_AGENT` on every agent. `/compact edit` then asks
**Save to**: this agent, or the whole folder. A `pi` you start yourself
in a terminal has no overlay unless you export that env.

A config file is optional. `/compact edit` writes `.pi/compact.json` (or
`.pi/compact/<agent>.json`).
