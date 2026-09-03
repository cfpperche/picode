# pi-checklist

Opt-in internal checklist for [pi](https://github.com/badlogic/pi-mono):
the agent writes its plan with the `checklist` tool before its first
change and keeps it updated; [PiCode](https://github.com/cfpperche/picode)
shows the current step on the sidebar (ADR-0055).

- **`checklist {items: [{text, status}]}`** — the whole list each call.
  Statuses: `pending` (default), `in-progress`, `completed`. The TUI
  prints it as `☐ ◐ ☑` lines; the plan survives a resume or a fork.
- **The gate.** With a level that requires a plan, `bash`, `powershell`,
  `edit`, `write` and `multiedit` are refused until `checklist` has run
  for the current task. The refusal names the tool to call; the model
  writes the plan and retries.
- **The reminder.** Under `always`, a turn that ends without a checklist
  gets a follow-up (at most three per task).

## Level

`PICODE_CHECKLIST` in the process env — PiCode sets it per agent from the
agent's settings; a raw `pi` without it is `changes`.

| Level | Meaning |
|---|---|
| `changes` | a checklist before the first change of a task; a read-only answer needs none |
| `always` | every task, read-only answers too |
| `never` | the tool stays, nothing is required |

## How it reaches PiCode

On every change the extension POSTs
`/api/agents/<PICODE_AGENT_ID>/checklist` — `server.json` (or
`PICODE_URL`) and the install token, re-read per call, loopback TLS
accepted — never awaited by the turn. With no reachable PiCode the tool,
the gate and the TUI card still work; only the sidebar stays quiet.

## Known limit

pi 0.84.4's print mode (`pi -p`) hangs on any blocked tool call — a
five-line extension that blocks `edit` once reproduces it — so a
`pi -p` run whose model skips the plan and goes straight to a change
sits until killed. Interactive and RPC modes (what PiCode runs) recover
as designed: the refusal is shown, the model writes the list, retries.

## Install

```sh
pi install -l /path/to/picode/packages/pi-checklist   # this workspace
pi install /path/to/picode/packages/pi-checklist      # this machine
pi -e /path/to/picode/packages/pi-checklist/extensions/checklist.ts  # this run
```

Nothing is installed by default (ADR-0010). MIT licensed — see LICENSE;
the rest of the PiCode repository is licensed separately.

## Tests

```sh
node --test test/*.test.ts
```
