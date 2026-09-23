# picode-mcp (ADR-0154)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

PiCode's own tools, spoken over the Model Context Protocol so a CLI agent
(Claude Code, Codex, OpenCode, …) gets what a pi agent gets from the
packages under `packages/`. The server is `picode mcp <family…>`: a stdio
JSON-RPC process inside the daemon's binary, stateless, that translates
`tools/call` into one request to the daemon's route and the answer back
into the pi package's words. Plan: `docs/plans/picode-mcp.md`.

## Who decides what

| Question | Decided by | Where |
|---|---|---|
| Which tools exist | the server's catalog | `internal/mcptool.Families()`: `computer`, `browser`, `inbox` (`notify_human`, `ask_human`), `checklist`, `delivery` (ADR-0171) |
| Who is calling | the environment the launch carries | `PICODE_AGENT_ID`, else `PICODE_TERM_ID` → `term:<id>`; neither → the tool answers `NoIdentity` and never dials |
| Where the daemon is | `PICODE_URL`, `PICODE_TERM_URL`, else `server.json` | `mcptool.ResolveURL`; unreachable at start is not fatal — the tool answers with the reason |
| The credential | the install token, read per call | `mcptool.ReadToken`: `PICODE_TOKEN` or `<data>/token`; a rotation needs no restart |
| May they act | the daemon | computer/browser use ADR-0143/0148/0134 grants and tiers; delivery uses ADR-0171 launch/repository ownership |
| What the model reads | `internal/mcptool/{computer,browser}.go` | ported from `packages/pi-*/src/logic.ts`; the package tests are the goldens |

## What each CLI hands its servers

The server is the CLI's child, so the identity arrives only if that CLI passes
its environment on — or PiCode writes it into the CLI's own server config.
Measured 2026-09-21 with an env probe in a real session of every catalog CLI,
and each one went on to declare a delivery through the tool:

| CLI | Identity reaches `picode mcp` because |
|---|---|
| Claude Code | the launch environment is inherited |
| OpenCode | the launch environment is inherited |
| Grok | the launch environment is inherited |
| Omp | the launch environment is inherited |
| Antigravity (`agy`) | the launch environment is inherited |
| Codex | it is written into the launch: `-c mcp_servers.<name>.env.PICODE_TERM_ID=…` and `PICODE_AGENT_ID` when the terminal is bound to an agent, plus `PICODE_TERM_URL` and `PICODE_DATA` — Codex hands a stdio server **no** environment, so an agent's tool calls answered `no identity` until the values were written (`internal/server/cli_tools.go`, `toolIdentityEnv`) |
| Hermes Agent | it has to be written into the CLI's own server entry (`env` in its `mcp_servers` block, the Connectors pane's env field): Hermes passes an allowlist of thirteen variables and none of them is PiCode's |
| Muse Code | the same for its `mcpServers.<name>.env` (Muse passes eight); the values must be literal — Muse does not expand `${VAR}` |

A connector card (`ToolPresets`, the Connectors pane) writes command and args
only, so for Hermes and Muse a card alone is not enough: the entry needs the
identity written into its environment field, and one entry can serve only one
terminal — their config is a single file with no workspace layer (PiCode
refuses the workspace scope for both, measured 2026-09-21) and neither CLI
exposes a per-launch config override, so the launch cannot write the value the
way `toolIdentityEnv` does for Codex. Recorded as a debt in
`docs/handoff/open/delivery-flow.md`; the user-facing steps are in
[the guide](../../docs-site/guide/picode-mcp.md).

## The wire

```
CLI ──stdio JSON-RPC──▶ picode mcp computer ──POST /api/computer/tool──▶ daemon ──▶ shell
     initialize · tools/list · tools/call · ping        {agent, term, action, params}
```

`tools/call` answers `{content: [text, image?], isError?}`; a refusal is a
result the model reads, never a JSON-RPC error. A capture travels as an
`image` block and is also written to `<data>/var/captures/<principal>/`
(last 50 per principal) with the path in the text, for a client that
renders files but not blocks. Unknown methods answer `-32601`; a
notification gets no line; nothing but protocol goes to stdout.

## Three scopes

| pi package scope | Guest | Mechanism |
|---|---|---|
| machine | user file | ADR-0150 drivers; catalog presets `picode-computer`, `picode-browser` (`mcp.ToolPresets`, hidden from pi's catalog) |
| workspace | project file | same drivers, same cards |
| agent | **launch** | `clilaunch.Config.Tools`; `internal/server/cli_tools.go` folds the servers into the launch options the way ADR-0106 does: Claude Code one generated `tools.mcp.json` per launch (the peer-communication file merged in, one `--mcp-config`), Codex `-c mcp_servers.<name>.{command,args}`, OpenCode `OPENCODE_CONFIG_CONTENT` merged. Grok, Hermes, Muse, Antigravity, Omp: refused at save time with the Connectors pane as the way |

The server name is the same in every scope so a launch entry and a file
entry collapse where the CLI merges by name. `cliView.toolsCapable` tells
the form which CLIs show the group; the PUT routes refuse the rest.

## ask_human over MCP

pi ends its turn and the human's reply rides pi's receiver back; a CLI agent
has no receiver. So `ask_human` files the question and **waits**: it polls
`GET /api/inbox/{id}` every 2 s for up to `PICODE_ASK_WAIT` seconds (8 h
by default) and returns the answer as the tool result. A client's tool
timeout is wall-clock plus an idle window (Claude Code: ~28 h, and a call
that stays silent is aborted), so the server runs each `tools/call` on its
own goroutine, sends `notifications/progress` with the client's
`progressToken` every 15 s while it waits, and honours
`notifications/cancelled`. Only at the deadline or on a cancel does it
name the item so the model can call `ask_human` again with `item`. On the daemon, `Deps.AnswerTerminalQuestion` is the one rule for both the
inbox route and the Inbox app: a terminal running pi gets the reply
delivered; any other terminal has the answer recorded on the item.

Since ADR-0184 every CLI launch is an agent, so most questions arrive
agent-sourced, and `Deps.AnswerAgentQuestion` (`internal/server/agent_answer.go`)
is the matching rule for those — again one rule for the route and the app,
first match wins:

| Condition when the human answers | Door |
|---|---|
| An asker polled the item with `?wait=1` in the last 30 s (`ask_human`, `picode inbox ask --wait`) | Recorded on the item; the poller returns it. Nothing is typed, so a waiting agent never hears the answer twice |
| Pi, or Omp whose question carries a session and whose terminal is running | ADR-0060: the receiver (Omp loads Pi's receiver with `-e`), else a verified paste; the JSONL row is the proof |
| Pi not in a terminal | The durable `follow_up` queue |
| Any other CLI (or Omp without a session), terminal running | Recorded, then typed into the TUI with a bracketed paste and a verified Enter |
| Any other CLI, not running — or the paste did not land | Recorded, and a note on the item says the agent was not told |

An MCP ask cannot name its conversation, so `POST /api/inbox` stamps an
agent's question with the session its receiver last reported; delivery
still checks the path (`resolveReplySession`, which also accepts
`omp-sessions/<agent>`). Before 2026-09-22 the agent path assumed Pi, and every
reply to a non-Pi agent's `ask_human` failed with "could not be identified
safely". `checklist` publishes under the terminal (or agent)
exactly as pi-checklist does; pi's mutation gate and reminder have no MCP
equivalent and are not pretended.

## Refuse

- No identity from the environment: the tool says so; the server never
  invents an agent or terminal id.
- A daemon route refusal (403, 502): passed through verbatim as `isError`.
- `picode mcp` with no or an unknown family: usage on stderr, exit 2; it
  never falls through to the server default (`cmd/picode/mcp_test.go`).
- A tool family a config names that the catalog lacks: the PUT refuses,
  and the launch refuses if a stored config carries one.

## Tests

`internal/mcptool` (framing goldens, schema equals the pi package's, calls
through a fake daemon, identity and discovery rules, capture pruning),
`cmd/picode` (dispatch), `internal/clilaunch` (Tools resolve/validate),
`internal/mcp` (presets), `internal/server/cli_tools_test.go` (the per-CLI
decision table and the routes), `web/shared/domain/cliLaunch.test.js`.

## Delivery declarations

`picode mcp delivery` and `picode delivery` share `CallDelivery` and the same
[delivery contract](delivery.md). Delivery has no Pi-package equivalent or
execution authority. Unset tool defaults in the three supported managed CLI
launchers include the family; explicit selections remain explicit. The UI tool
picker is unchanged and does not offer this family yet. No messages connection
is enabled or used.
