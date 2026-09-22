# ADR-0184: One door for agent CLIs

- **Status**: accepted (owner session, 2026-09-22)
- **Date**: 2026-09-22
- **Boundary**: persistence — every terminal that launches a catalog CLI
  on behalf of a user is bound to an `agents` row (existing unbound rows
  migrate); security model — `term:<id>` stops being a principal that can
  hold a browser, computer or delivery grant, since the only terminals left
  without an agent are shells and sign-in terminals; protocol —
  `POST /api/clis/{cli}/terminals` is removed from the public API.
- **Amends**: ADR-0160 (its line "Unbound CLI terminals (`#/clis/new`,
  project shells) stay terminals" — unbound *CLI* terminals end; shells
  stay), ADR-0143 (terminal principal grants).

## Context

ADR-0160 made every launchable CLI an agent, but kept a second door:
`createCLITerminal` (`internal/server/cli_launch.go`) opens a terminal with
a CLI launch and no agent. It is reached from Agent CLIs → **New terminal**
and a profile's **Use** (both apps), the palette's `cli-new` outside a
workspace, the Sessions tab's resume for non-Pi CLIs, the cross-CLI session
handoff, and the credential sign-in. Such a terminal carries its own
identity, `term:<id>`, which browser and computer grants, delivery, Inbox
and peer messages each special-case. A CLI typed into a PiCode shell has
no identity at all. The owner wants one door, controlled by PiCode.

Measured 2026-09-22 on the owner's instance: 11 terminals bound to agents,
1 unbound CLI terminal (`checklist-mirror`, omp).

## Decision

A catalog CLI launched for a user always runs as an agent. Every user-facing
launch — New agent, a profile, the palette, resuming a session, a handoff —
creates an agent (in a workspace, or free) whose terminal carries the launch.
The public `POST /api/clis/{cli}/terminals` is removed. Two terminals without
an agent remain: **shells**, and **sign-in terminals**, which the server
creates internally, are visible on the credential card that opened them (not
in the sidebar or a CLI's Terminals list), are closed by the server when the
credential appears, when their process exits, after an idle limit, and at
boot, and can hold no grant. A catalog CLI observed running in a PiCode shell
offers **Make agent**, which binds that terminal to a new agent in the
shell's workspace; nothing is adopted without the click. CLIs run outside
PiCode are out of scope. Existing unbound CLI terminals migrate to agents,
and their `term:<id>` grants are rewritten to the agent id.

## Consequences

Easier: one identity for grants, delivery, Inbox, automations and messages;
`term:` special cases shrink to shells; `#/clis` becomes a runtime surface
(install, settings, providers, sessions) and stops being a launcher.
Harder: a throwaway CLI run now enters the fleet — removing it is the
agent's Remove. Sign-in terminals become invisible outside their card, so
their reaping is the contract: a leak there is a process nobody sees, and
tests cover every close path. If we are wrong about a caller, the failure is
a launch that 404s, not a CLI running with a grant nobody owns.

## Decision table

| Conditions | Action |
|---|---|
| New agent / profile Use / palette, any launchable CLI | agent row (workspace or free) + bound terminal with launch |
| Resume a non-Pi session | agent in the session's workspace (free if none), launch args = resume args |
| Cross-CLI handoff | agent for the destination CLI, launch args = handoff args |
| Credential sign-in | internal sign-in terminal, no agent, no grant, shown on the card |
| Sign-in: credential stamp changes / process exits / idle limit / boot | server closes the terminal and its tmux session |
| Shell, foreground = catalog CLI | "Make agent" offer; click binds terminal to a new agent |
| Shell, CLI exits or offer ignored | stays a shell, no grant |
| `POST /api/clis/{cli}/terminals` | 404 |
| Grant request as `term:<shell or sign-in>` | refuse |
| Existing unbound CLI terminal | migrate: agent row bound to it; `term:<id>` grants → agent id |
| CLI outside PiCode | out of scope |

## Alternatives considered

- **Keep the terminal door (ADR-0160 as written).** Lost: two identities
  for one gesture, and a CLI with grants outside the fleet.
- **Sign-in as an agent.** Lost: a seconds-long, workspace-less process
  would appear in the fleet, Inbox and automations and then vanish.
- **Adopt a CLI in a shell automatically.** Lost: `claude --version` would
  create an agent; a click costs one gesture and never surprises.
- **An "ephemeral agent" kind for throwaway runs.** Deferred: nobody has
  asked; Remove covers it.
