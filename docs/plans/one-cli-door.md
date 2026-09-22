# One door for agent CLIs (ADR-0184)

Every user-facing CLI launch becomes an agent; the terminal door closes.
Four branches, in order. Each one leaves the app working: slice 1 moves
every caller, slice 3 closes the route only once no caller is left.

Inventory (2026-09-22, `main` at 04571898):

| Caller of `createCLITerminal` | Where |
|---|---|
| Agent CLIs → New terminal, profile Use (`#/clis/new/<cli>`) | `web/{browser,mobile}/src/components/AgentClis.jsx` (423 / 365), `CliProfiles.jsx:16` |
| Palette `cli-new` outside a workspace | `web/browser/src/App.jsx:4363` |
| Sessions → resume a non-Pi session | `web/browser/src/components/SessionsView.jsx:320` (and mobile's) |
| Cross-CLI handoff | `internal/server/cli_handoff.go:508` |
| Credential sign-in | `internal/server/credentials.go:760` |

`term:<id>` consumers: `browser_policies.go:69,125`, `computer.go:192`,
`delivery.go:87`, `apps/inbox.go:494,503`, `PeerMessages.jsx` (Terminal
controls → `#/clis/<cli>/terminals`), the CLI Terminals tab.

## 1. `feat/cli-door-agents` — every launch creates an agent

- Server: one helper, `createCLIAgent(deps, r, cli, workspaceID, name, cwd,
  overrides)`, built on the workspace/free agent create paths
  (`handleAddWorkspaceAgent`, `handleAddFreeAgent`) so the agent's bound
  terminal carries `overrides` in its launch. The handoff calls it instead
  of `createCLITerminal`; the response gains `agentId`.
- UI (both apps): New terminal → **New agent** (opens the create form with
  the CLI and profile preset); palette `cli-new` → free agent; Sessions
  resume → `POST /api/agents` / workspace agents with `overrides.args`;
  navigate to the agent, not `termHash`.
- The CLI's Terminals tab lists that CLI's **agents** (links to them).
- Tests: handoff creates an agent; resume creates an agent with resume args;
  both apps' create flows per CLI. Visual review: Agent CLIs, Sessions.

## 2. `feat/cli-door-migrate` — unbound CLI terminals become agents

- Store migration: each `terminal_launches` row whose terminal no agent
  binds gets an `agents` row (`cli` = launch CLI, workspace = terminal's,
  free if none, name = terminal name) bound to it; one event per row.
- Grants: `term:<id>` keys (browser policies, computer grant) for those
  terminals are rewritten to the agent id in the same transaction.
- Delivery, Inbox, PeerMessages: drop the `term:` fallback for CLI
  terminals; a `term:` that is a shell or sign-in is refused.
- Tests: migration idempotent; grant rewrite; `TestEveryMutationAppendsAnEvent`.
  Scratch with a copy of the owner's DB (1 row: `checklist-mirror`, omp).

## 3. `feat/cli-door-close` — close the route, confine sign-in

- Remove `POST /api/clis/{cli}/terminals` (OpenAPI regenerated).
- Sign-in terminal: created internally, flagged `kind=signin` on the
  terminal row, hidden from the sidebar and the Terminals list, shown on the
  credential card with its state and a Close button.
- **Reaping — nothing runs unseen.** The server closes a sign-in terminal
  (row + exact tmux session) when: the credential stamp changes; its process
  exits; it has been idle past a limit (15 min); at boot for any left over.
  A sign-in terminal can hold no grant.
- Tests: one per close path, plus "grant request as sign-in → refuse".

| Sign-in terminal state | Action |
|---|---|
| stamp changed | close, card says signed in |
| process exited, stamp same | close, card offers Retry |
| idle > limit | close, card says timed out |
| boot finds one | close |
| card Close | close |

## 4. `feat/cli-door-adopt` — Make agent from a PiCode shell

- Detection: the shell's foreground command (`pane_current_command`,
  `internal/tmux`) matches a catalog CLI executable; published on the
  terminal's live state, no new poll (reuse the existing tmux status pass).
- UI: the shell's header shows "Claude Code is running here · Make agent".
  Click binds the terminal to a new agent in the shell's workspace (store
  mutation + event); the tab turns into the agent's.
- Not adopted: ignored offer, CLI exits (offer disappears), terminals
  outside PiCode.
- Open: latency from CLI start to the offer, measured before done.
