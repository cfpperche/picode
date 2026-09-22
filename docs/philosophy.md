# Philosophy

> Why PiCode exists and the values that arbitrate every design fight.

## 1. The moat: control from creation

Coding agents are becoming team members. Most tools treat the agent as a
chat window that already exists. PiCode treats the agent as an **asset with
a lifecycle**: created (wizard: workspace, model, skills, provider),
configured (profiles, extensions, auth), tasked (queue, steer, follow-up),
coordinated (broker), and audited (sessions, diffs, costs).

Owning that lifecycle — from the very first "New Agent" click — is the moat.

## 2. The browser is a door, not a cage

Every GUI convenience must have a terminal escape hatch:

- The embedded terminal is the **real TUI** of whichever CLI the agent runs
  (Pi, Claude Code, Codex, …), not a reimplementation.
- Anything configurable in the GUI is inspectable as that CLI's own config
  files.
- If PiCode disappeared tomorrow, agents keep running in tmux and each CLI
  keeps its own sessions.

A GUI that becomes a bottleneck between the user and their agent has failed.
We measure ourselves against the fear we're removing — never adding new
"walled garden" fear back.

## 3. Simplicity

- One binary. One command. Browser opens. Working.
- Standard library first; dependencies are decisions, not conveniences.
- Fewer concepts, honestly named. If a setting needs a manual to explain,
  it's a design bug.

## 4. Modularity

Every PiCode capability maps onto primitives the CLI's users already know,
and PiCode never replaces them:

| PiCode feature | What sits underneath |
|---|---|
| Agent config | each CLI's own settings, extensions and skills files |
| Providers | the CLI's own auth; one vault feeds them (ADR-0165) |
| Sessions | each CLI's own session files; handoff translates between them (ADR-0088) |
| Tasks & steering (Pi managed mode) | pi's `steer` / `follow_up` semantics |
| Inter-agent chat | `picode-communication` over MCP or the `picode messages` command → HTTP API, connected automatically for Pi, Claude Code, Codex, OpenCode, Grok and Hermes Agent |

We extend each CLI's ecosystem; we never fork one. Pi is where PiCode
started and the one CLI with a managed mode today; it is not a dependency
(ADR-0179).

## 5. Agents first

PiCode is developed **by** coding agents — Pi, Claude Code, Omp and the
others — as much as **for** their users. The repo is an agent-native
workspace: `AGENTS.md` is the operating contract, skills encode
quality gates, `docs/handoff/open/` carries state across sessions (the board
`docs/handoff.md` is a generated view of it, ADR-0123), and ADRs keep
decisions honest. If our own agents can't thrive here, the product is a lie.

## 6. Respect the terminal-averse user

Our second audience didn't choose the terminal and shouldn't be punished for
it. That means: no unexplained jargon, no "just run this command" dead ends,
progressive disclosure, friendly empty states, and safety rails on
destructive actions — while still being a power tool for people who *do*
live in the terminal. Both audiences, one tool, no dumbing down.

## 7. Optimistic UI (never a blank wait)

A waiting screen is a broken screen. While data is in flight the UI shows
the **shape** of the result (skeletons that match the loaded layout) or
the **last good data** (stale-while-revalidate). Spinner-only and empty
white wells are defects.

This is not license to invent content. Skeletons are chrome. Status stays
truthful (philosophy of deference): we never paint fake packages, fake
search hits, or fake progress bars. Pending *actions* (Install…) live on
the control that was clicked.
