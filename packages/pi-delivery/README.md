# pi-delivery

Delivery declarations for [pi](https://github.com/badlogic/pi-mono): the
`delivery` tool registers a change in the repository the agent was launched
in, asks the human for review, or reads what is already declared.
[PiCode](https://github.com/cfpperche/picode) shows it in the project's
**Git ▸ Delivery** view (ADR-0171).

- **`delivery {action, …}`** — `capabilities`, `register`, `update`,
  `request-review`, `withdraw-review`, `show`, `list`.
  `register` names the title, the local branch, the full commit id and the
  target branch; the answer carries the delivery `id` and `version` every
  later call needs.
- **The repository is the launch.** PiCode derives it from the folder the
  agent was started in, and the identity from `PICODE_AGENT_ID` (managed
  agent) or `PICODE_TERM_ID` (a pi in a PiCode terminal). Neither can be
  chosen by the model, and no vendor credential is involved.
- **Declarations only.** A review request is an agent's statement that the
  human should look — never an approval, a passed check, a merge, a queue
  or a deploy. Queues and execution are not implemented, and the daemon
  answers `capability unavailable` if a client asks for them.
- **Retries replay.** A mutation needs a retry key: the tool derives one
  from the session, the action and the payload, so repeating the call
  replays the same declaration instead of registering a second one. Pass
  `requestId` explicitly to force a distinct key.
- **An update clears the review request.** After changing the deliverable,
  call `update` with the returned `id` and `version`; the Delivery view
  goes back to "no review request" until you ask again.
- **Soft failure.** With no reachable PiCode the tool returns an
  explanatory text result (never a thrown error to retry against); with no
  identity it refuses and names the way out.

## Install

`packages/pi-delivery` from the PiCode repository — Agent CLIs ▸ Packages
(**This agent**, `-e`, or `pi install -l` for the whole folder). Guide:
[Delivery](/guide/delivery) and [Packages](/guide/packages).

## Command line

The same contract is `picode delivery <action>` in a PiCode terminal, and
`picode mcp delivery` for an agent CLI that takes MCP servers (the launch
"PiCode tools" switch). One daemon route serves all three:
`POST /api/delivery/tool`.
