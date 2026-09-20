# ADR-0154: picode-mcp — PiCode's own tools for every agent CLI, over MCP

- **Status**: proposed (design approved by the owner in session,
  2026-09-18: one server per pi package, scope and toggle belong to the CLI,
  the per-agent scope of guests is launch injection)
- **Date**: 2026-09-18
- **Boundary**: protocol — the `picode` binary speaks the Model Context
  Protocol over stdio (`picode mcp <family…>`), a second wire beside the
  pi extension API for the same tools. Process — PiCode injects its own MCP
  servers into the command line and environment of guest CLIs it launches,
  and writes them into the CLIs' native config files through the ADR-0150
  drivers. Security model — unchanged on purpose: identity and grants stay
  in the daemon (ADR-0143, ADR-0148); the server carries none of its own.

## Context

Every tool PiCode gives an agent — `computer` (ADR-0148), `browser`
(ADR-0132/0134), `notify_human`/`ask_human` (pi-inbox), `checklist` — is a
pi extension: a package under `packages/` that calls one daemon route with
`PICODE_AGENT_ID` or `PICODE_TERM_ID` as identity and the install token as
credential. Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code,
Antigravity and Omp do not load pi extensions; they take tools over MCP.
The owner's first real run of computer use (2026-09-18) stalled on exactly
that: the CLI they use is Claude Code.

The daemon side is already CLI-agnostic. `POST /api/computer/tool` and
`POST /api/browser/tool` take `{agent, term, …}`, resolve the principal by
the house rule (`internal/grant.Key`: agent id, else `term:<id>`, else no
identity) and refuse fail-closed. A CLI opened in a PiCode terminal already
carries `PICODE_TERM_ID` and `PICODE_TERM_URL` in its environment
(ADR-0056), and Settings ▸ Computer and Settings ▸ Browser already list
terminals as principals (ADR-0143). What is missing is the last inch: a
process that speaks MCP to the CLI and HTTP to the daemon.

Three facts shape the shape of that process. ADR-0150 (accepted 2026-09-17)
gives PiCode a driver per guest CLI that writes the CLI's native MCP config
at user and project scope, behind one Connectors pane with a catalog of
presets; it rejected a PiCode-owned registry of third-party connectors
injected at launch. ADR-0106 already injects a PiCode-owned MCP server at
launch for peer communication — `--mcp-config` for Claude Code, process-local
config overrides for Codex, `OPENCODE_CONFIG_CONTENT` for OpenCode — so the
launch path exists and is house practice. And pi packages are opt-in per
package at three scopes (machine, workspace, agent); a guest has no "agent"
in PiCode's sense (ADR-0069: guests are terminals), so `guestScope` refuses
the agent scope today.

Measured 2026-09-18: the `picode` binary is on every terminal's PATH
(`~/.local/bin/picode`, 70 MB, subcommands dispatched before the server
default); MCP over stdio is newline-delimited JSON-RPC with four methods
this server needs (`initialize`, `tools/list`, `tools/call`, `ping`); MCP
tool results carry `{type:"image"}` blocks; the pi packages' pure logic is
804 lines of TypeScript across four `logic.ts` files, each standalone by
house rule (no shared runtime module).

## Decision

The `picode` binary gains a `mcp` subcommand: `picode mcp <family…>` is a
stdio MCP server exposing one PiCode tool family per argument —
`computer`, `browser`, `inbox` (`notify_human`, `ask_human`), `checklist` —
with the same tool names, parameters, result wording and refusal messages
as the pi package of the same name, rendered in Go (`internal/mcptool`)
with the package's tests as goldens. It is stateless: identity is the
inherited `PICODE_AGENT_ID`/`PICODE_TERM_ID`, the URL is `PICODE_TERM_URL`
or `server.json`, the credential is the install token read per call, and a
call outside a PiCode terminal answers "no identity — open the CLI in a
PiCode terminal" rather than inventing one. Opt-in mirrors the pi package
boundary exactly: one server per family, registered as an ordinary
connector in the CLI's own config, so the CLI's native scope and enable
mechanism decide who sees the tool — machine is the user file, workspace
is the project file (both through the ADR-0150 drivers, from catalog
presets "PiCode · Computer", "PiCode · Browser", …), and the per-agent
scope of a guest is **launch injection**: the CLI's Launch settings
(ADR-0056) gain a "PiCode tools" group, and a terminal launched with it
receives the servers by the same mechanism ADR-0106 uses — one generated
`--mcp-config` file per launch for Claude Code (merged with the peer
communication file, never two flags), process-local overrides for Codex,
`OPENCODE_CONFIG_CONTENT` for OpenCode; a CLI without a launch mechanism
reports "workspace scope only" in the UI instead of pretending. Server
names are the same in every scope (`picode-computer`, …) so a launch entry
and a file entry collapse where the CLI merges by name. Screenshots travel
as an MCP image block and are also written under `<dataDir>/var/captures/`
with the path in the text, for clients that render files but not blocks.
What the tool may do stays in the daemon: the computer grant, the browser
tier and domains, the `term:<id>` principal and the audit rows are exactly
those of ADR-0143/0148. Pi keeps its native packages; it does not consume
this server.

## Consequences

Every guest CLI gets PiCode's tools with one click in the Connectors pane
or one checkbox in Launch settings, and a prompt written for the pi
`computer` tool works unchanged in Claude Code. The principal a guest
acts as is its terminal, so the switches the owner already knows in
Settings ▸ Computer and ▸ Browser apply without a new page; a new terminal
is a new principal and needs its own switch, as with the browser today
(inheriting the grant from the CLI's Launch settings is a named M5
refinement, not part of this decision). The daemon gains no route, no
credential and no port; the server is a translator, and a rotated token or
a restarted daemon is picked up on the next call.

Costs accepted: the rendering of each family exists twice, in TypeScript
for pi and in Go for MCP, held equal by shared golden tests — converging
both on daemon-side rendering is possible later and deliberately not done
now. One small idle process per family per CLI session. Guest launch
injection is honest only where the vendor offers a mechanism; Grok, Hermes,
Muse, Antigravity and Omp are confirmed in the first milestone and fall
back to workspace scope with a visible note where they do not. ADR-0150's
rejected alternative ("a PiCode-owned registry injected at launch") is
narrowed, not reversed: PiCode injects only its own tools into processes it
launches, as ADR-0056 and ADR-0106 already do; third-party connectors keep
living in the CLIs' files.

If we are wrong: a CLI that does not merge same-name servers shows the tool
twice (harmless: same server, same daemon decision); a client that ignores
image blocks sees the file path and a line saying so; a `picode mcp` run
that ever fell through to the binary's default would start a second daemon
on the production data dir — the subcommand is dispatched before any flag
and covered by a test.

## Alternatives considered

- **A Node package (`packages/picode-mcp`) on the MCP SDK**: rejected. It
  needs an `npm install` at registration time and a Node on PATH, shares no
  code with the pi packages anyway (each is standalone), and adds two
  dependencies where the Go subcommand adds none and is already installed
  everywhere a terminal runs.
- **One server with every tool, opt-in decided by the daemon per terminal
  (`tools/list` from Settings)**: rejected. It is the registry ADR-0150
  refused, cached differently by each CLI, and duplicates the scope and
  toggle every CLI already has.
- **A Streamable HTTP MCP endpoint in the daemon**: deferred. It serves
  CLIs on other machines, which no supported launch path produces today;
  it needs session auth in the MCP handshake and a second listener surface.
  The stdio server does not preclude it.
- **Teaching pi to consume the MCP server too, retiring the packages**:
  rejected. pi's native tools render in its TUI (`renderCall`,
  `renderResult`) and carry `details.preview` for the chat capture; MCP has
  neither.
- **Per-family opt-in through Launch settings only, no file scopes**:
  rejected. It would break the promise of ADR-0150 that a connector added
  for one CLI is a normal entry in its own config, visible to launches made
  outside PiCode, and would make machine and workspace scope PiCode-only.
