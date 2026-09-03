# Checklist

Make an agent write its plan before it changes anything, and follow that
plan from the sidebar without opening the agent.

Install the `pi-checklist` package for the machine, the folder or one
agent (see [Packages](/guide/packages)); the path is
`packages/pi-checklist` in the PiCode repository. Without it nothing
changes.

```sh
pi install /path/to/picode/packages/pi-checklist
```

With it, the agent has a `checklist` tool and a rule: before its first
edit, write or shell command in a task, it writes 2–8 steps and marks the
one it starts. Until it does, changes are refused and the refusal tells
it what to call. As steps finish it updates the list.

## What you see

- **Sidebar**: one line under the agent — `(2/4) the step it is on`.
  When a plan is finished it reads `(4/4)`. If the task required a plan
  and the agent did not write one, the line says **No checklist**.
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

A read-only agent is never asked: it cannot change anything.

## Notes

- Each new prompt is a new task: the agent writes or updates the plan
  again before its next change. Small fixes get a one-line plan.
- The plan lives in the agent's session; a resume or a fork keeps it.
- A plain `pi` outside PiCode with the package gets the tool, the rule
  and the card in its own terminal.
