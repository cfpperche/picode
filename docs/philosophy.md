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

- The embedded terminal is the **real Pi TUI**, not a reimplementation.
- Anything configurable in the GUI is inspectable as Pi config files.
- If PiCode disappeared tomorrow, agents keep running in tmux + Pi sessions.

A GUI that becomes a bottleneck between the user and their agent has failed.
We measure ourselves against the fear we're removing — never adding new
"walled garden" fear back.

## 3. Simplicity (inherited from Pi)

- One binary. One command. Browser opens. Working.
- Standard library first; dependencies are decisions, not conveniences.
- Fewer concepts, honestly named. If a setting needs a manual to explain,
  it's a design bug.

## 4. Modularity (inherited from Pi)

Every PiCode capability maps onto Pi primitives users already know:

| PiCode feature | Pi primitive underneath |
|---|---|
| Agent config | `.pi/settings.json`, extensions, skills |
| Tasks & steering | `steer` / `follow_up` semantics |
| Auth wizard | `/login` OAuth flows |
| Session tree | session JSONL (tree via `id`/`parentId`) |
| Inter-agent chat | extension tools → HTTP API |

We extend Pi's ecosystem; we never fork it.

## 5. Agents first

PiCode is developed **by** Pi agents as much as **for** Pi users. The repo is
a Pi-native workspace: `AGENTS.md` is the operating contract, skills encode
quality gates, `docs/handoff.md` carries state across sessions, and ADRs keep
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
