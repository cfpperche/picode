---
description: The agent writes a plan before it changes anything; the sidebar follows it.
---

# Checklist

Make an agent write its plan before it changes anything, and follow that
plan from the sidebar without opening the agent.

- **Where:** after you install `pi-checklist` ([Packages](/guide/packages)), the plan is a line on the agent in the sidebar.
- **Not this:** not a [canvas](/guide/canvas) pin or note. Optional package — without it, no tool, no gate, no sidebar line.

`pi-checklist` is an **optional pi package — an extension, not part of
PiCode core**. The tool and the rule live in the package; PiCode only
renders what the agent reports and stores the obligation level you pick
per agent.

Install `packages/pi-checklist` from the PiCode repository. Guide for
install targets: [Packages](/guide/packages).

```sh
pi install /path/to/picode/packages/pi-checklist
```

With it, the agent has a `checklist` tool and a rule: before its first
edit, write or shell command in a task, it writes 2–8 steps and marks the
one it starts. Until it does, changes are refused and the refusal tells
it what to call. As steps finish it updates the list.

## Where it runs

| | What you get |
|---|---|
| **Pi TUI** (terminal) | The tool, the rule and the step card in your own terminal |
| **PiCode chat** | The same tool and rule, plus the sidebar line and the step cards |
| **PiCode core** | The surfaces only: the sidebar line, the chat cards, the phone row, and the per-agent Level setting that the package reads |

The split runs the other way than usual here: the package holds the
behavior, PiCode holds the mirrors. The agent posts its plan to PiCode;
delete the package and the mirrors have nothing to show.

The mirrors cover agent CLI terminals too: a pi you launch in a PiCode
terminal reports its plan under that terminal, so the terminal's sidebar
card shows the same line a managed agent's card does. The terminal pane
is the TUI itself — it already draws the step card, so PiCode does not
repeat it above the pane. Other CLIs (Claude Code, Codex, Grok, Hermes Agent, OpenCode) have no
checklist package, so their terminals never show a line.

## What you see

- **Sidebar**: one line under the agent — `(2/4) the step it is on`.
  When a plan is finished it reads `(4/4)`. Nothing known, and a task
  without a plan, shows no line at all (ADR-0092).
  Agent CLI terminal cards carry the same line.
- **Chat**: each `checklist` call is a card with every step —
  `☐` pending, `◐` in progress, `☑` done.
- **Phone**: the agent row's second line is the current step.

## Level, per agent

Agent settings → **Checklist**:

| Level | Meaning |
|---|---|
| **Before changes** (default) | a plan before the first change; a question or a read-only answer needs none |
| **Always** | every task, read-only answers too — a turn without a plan gets a reminder, up to three |
| **Never** | the tool stays, nothing is required |

A read-only agent is never asked: it cannot change anything. The level
travels to the package as an environment variable on the agent's
process; a plain `pi` outside PiCode gets the default (before changes).

## Notes

- Each new prompt is a new task: the agent writes or updates the plan
  again before its next change. Small fixes get a one-line plan.
- The plan lives in the agent's session; a resume or a fork keeps it.
- There is no config file — the obligation level in agent settings is
  the only knob.
