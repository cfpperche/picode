# ADR-0081: The internal checklist follows the agent into its terminal

- **Status**: accepted
- **Date**: 2026-09-06
- **Extends**: ADR-0055 (internal checklist), ADR-0069 (Agent CLI terminals)

## Context

ADR-0055 gave every managed agent an internal plan: `pi-checklist` publishes
the list and PiCode shows one operator line per agent. The data plane keys on
`PICODE_AGENT_ID` — a managed agent's spawn env — and lands on
`/api/agents/{id}/checklist`, stored per agent (`agent_checklists`) and
announced as `agent.checklist`.

Agent CLIs (ADR-0069) changed who runs where. A pi launched in a CLI terminal
is the same pi with the same package — its TUI renders the checklist card —
but it has no `PICODE_AGENT_ID`, so `publish()` silently returned: PiCode
never heard about the plan, and the terminal's sidebar card showed nothing
while the agent's chat card did. The owner saw the gap while supervising a
terminal pi: "managed agents show the checklist, agent CLI terminals don't."

What a terminal already has is `PICODE_TERM_ID` — injected into every tmux
session at creation (shell and CLI terminals alike), which is how the
lifecycle hooks (ADR-0056 tier 1) correlate their reports. Terminal state
itself is ephemeral by design; the checklist is not presence but content, the
latest list the agent published, and managed agents' rows survive a daemon
restart in SQLite. Dropping a terminal's line on restart would make the two
cards disagree for no honest reason.

## Decision

The checklist follows the agent, not the spawn mechanism. `pi-checklist`'s
publish target is: `PICODE_AGENT_ID` when set (managed agents, unchanged);
else `PICODE_TERM_ID` when set, posting to `POST /api/terminals/{id}/checklist`
(new); else silent (a raw pi outside PiCode — no channel, no line).

The server stores terminal rows in `terminal_checklists` (one per terminal,
same shape and validation as agents'), announces `terminal.checklist` as a
durable feed event, and folds the row into every terminal view next to the
live state — the `GET /api/terminals` list stays the one boot fetch. A reset
marker, a removed terminal, and a deleted agent's checklist all behave like
their agent-side siblings. The shells render the same one-line projection on
terminal cards and above terminal panes that agent cards already carry; no
channel still means no line.

The obligation level stays a managed-agent setting: a terminal pi runs at the
package default (`changes`), because a terminal belongs to the user, not to a
per-agent configuration that does not exist there.

### Decision table

| Conditions | Action / observable result |
|---|---|
| `PICODE_AGENT_ID` set (any other env) | POST to the agent route; agent card shows the line (unchanged) |
| No agent id, `PICODE_TERM_ID` set | POST to the terminal route; terminal card and pane show the line |
| Neither | No publish; nothing shown anywhere (raw pi outside PiCode) |
| Terminal row valid / absent / reset | Line / "No checklist" / no line — same vocabulary as agent cards |
| POST for an unknown or removed terminal | 404; no row is created |
| Terminal removed | Row dies with it inside `DeleteTerminal` (no orphan, no extra event) |
| Daemon restart | Rows survive (SQLite), like agent checklists; live presence still does not |

## Consequences

One supervision language for every pi PiCode can see, whatever launched it.
The cost: a second keyed table and route alongside the agents', and one rule
to remember — `PICODE_AGENT_ID` wins over `PICODE_TERM_ID` if a process ever
carries both (a managed agent never does). Terminal checklists for CLIs other
than pi remain future work: other CLIs have no checklist package; their
terminal cards simply never receive a line, which is the honest state.

## Alternatives considered

- **Set `PICODE_AGENT_ID=<terminal-id>` on terminal launches.** Rejected: it
  would make terminals impersonate agents on routes that mean "managed agent"
  (inbox, roles), and the agents route 404s unknown ids anyway.
- **Observe checklist calls in the injected terminal-state extension.**
  Rejected: duplicates the package's reconstruction and normalization logic in
  a Go string template, and only covers terminals with integration enabled.
- **Keep terminal rows ephemeral like terminal state.** Rejected: the
  checklist is content, not presence; a restart would silently drop lines the
  agent's session still holds.

## References and adaptation

- ADR-0055 supplied the data plane, the vocabulary and the "no channel means
  no line" rule; this ADR reuses all three under a second identity.
- ADR-0069 supplied `PICODE_TERM_ID` correlation, proven by the lifecycle
  hooks since 2026-09-04.
