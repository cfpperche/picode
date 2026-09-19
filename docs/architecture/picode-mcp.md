# picode-mcp (ADR-0154)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

PiCode's own tools, spoken over the Model Context Protocol so a guest CLI
(Claude Code, Codex, OpenCode, …) gets what a pi agent gets from the
packages under `packages/`. The server is `picode mcp <family…>`: a stdio
JSON-RPC process inside the daemon's binary, stateless, that translates
`tools/call` into one request to the daemon's route and the answer back
into the pi package's words. Plan: `docs/plans/picode-mcp.md`.

## Who decides what

| Question | Decided by | Where |
|---|---|---|
| Which tools exist | the server's catalog | `internal/mcptool.Families()`: `computer`, `browser`, `inbox` (`notify_human`, `ask_human`), `checklist` |
| Who is calling | the environment the CLI inherited | `PICODE_AGENT_ID`, else `PICODE_TERM_ID` → `term:<id>`; neither → the tool answers `NoIdentity` and never dials |
| Where the daemon is | `PICODE_URL`, `PICODE_TERM_URL`, else `server.json` | `mcptool.ResolveURL`; unreachable at start is not fatal — the tool answers with the reason |
| The credential | the install token, read per call | `mcptool.ReadToken`: `PICODE_TOKEN` or `<data>/token`; a rotation needs no restart |
| May they act | the daemon | exactly ADR-0143/0148/0134: the grant, the tier, the audit rows |
| What the model reads | `internal/mcptool/{computer,browser}.go` | ported from `packages/pi-*/src/logic.ts`; the package tests are the goldens |

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

pi ends its turn and the human's reply rides pi's receiver back; a guest CLI
has no receiver. So `ask_human` files the question and **waits**: it polls
`GET /api/inbox/{id}` every 2 s for up to `PICODE_ASK_WAIT` seconds (90 by
default, under the client's own tool timeout), and returns the answer as
the tool result. At the deadline it names the item so the model can call
`ask_human` again with `item` to keep waiting. On the daemon, `Deps.AnswerTerminalQuestion` is the one rule for both the
inbox route and the Inbox app: a terminal running pi gets the reply
delivered; any other terminal has the answer recorded on the item. `checklist` publishes under the terminal (or agent)
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
