# Project plan: picode-mcp (PiCode's tools for every agent CLI)

Status: design approved by the owner in session on 2026-09-18 (one server per
pi package; scope and toggle are the CLI's; a guest's per-agent scope is
launch injection). Executes ADR-0154 (proposed).

## Objective

A CLI that speaks MCP — Claude Code first, then Codex, OpenCode, Grok,
Hermes Agent, Muse Code, Antigravity and Omp — gets the same tools a pi agent
gets from the packages under `packages/`: `computer`, `browser`,
`notify_human`/`ask_human`, `checklist`. Same names, same parameters, same
answers and refusals. The daemon keeps deciding identity and grants; the MCP
server is a stateless translator that ships inside the `picode` binary.

## Engineer decisions (the owner may override any)

1. **`picode mcp <family…>`**: a stdio MCP server as a subcommand of the
   binary already on every terminal's PATH. Go, zero dependencies,
   dispatched before the binary's server default (a test guards that).
2. **One server per family**, mirroring the pi package boundary: `computer`,
   `browser`, `inbox`, `checklist`. Several families in one process when the
   user registers `picode mcp computer browser`; the presets register one.
3. **Identity from the inherited environment** (`PICODE_AGENT_ID`, else
   `PICODE_TERM_ID` → `term:<id>`, else none). URL from `PICODE_TERM_URL`,
   else `server.json`. Credential: the install token, read per call. Outside a
   PiCode terminal the tool answers "no identity — open the CLI in a PiCode
   terminal"; nothing is invented.
4. **Vocabulary 1:1 with the pi packages.** Rendering lives in
   `internal/mcptool`, one file per family, with the package's `logic.test.ts`
   cases ported as Go goldens so the two stay equal.
5. **Images as MCP image blocks and as files**: the PNG also lands under
   `<dataDir>/var/captures/<principal>/<seq>.png` with the path in the text.
6. **Three scopes, three mechanisms**: machine = the CLI's user file,
   workspace = its project file (both via the ADR-0150 drivers, from catalog
   presets "PiCode · Computer", "PiCode · Browser", "PiCode · Inbox",
   "PiCode · Checklist"); per-agent for guests = launch injection from the
   CLI's Launch settings ("PiCode tools" group, per-terminal override), by
   the ADR-0106 mechanisms (Claude Code `--mcp-config` — one generated file
   per launch merged with the peer-communication one; Codex process-local
   overrides; OpenCode `OPENCODE_CONFIG_CONTENT`). Same server names in
   every scope. A CLI without a launch mechanism says "workspace scope only".
7. **Pi unchanged.** Native packages stay; pi does not consume the server.
8. **No new route, port or credential** in the daemon.

## Spec

### The server

- Transport: stdio, newline-delimited JSON-RPC 2.0; protocol version
  negotiated in `initialize` (accept the client's, answer the latest this
  server knows); `tools/list` (tool schemas as JSON Schema, generated from
  the same tables the pi packages use), `tools/call` (content blocks +
  `isError`), `ping`; unknown methods answer `-32601`. Notifications are
  ignored. No resources, prompts or sampling.
- Server info: `{name: "picode-<family>", version: <binary version>}`;
  with several families `picode`.
- `tools/call` → one `POST` to the family's route with `{agent, term, …}` and
  `Authorization: Bearer <token>`; TLS on loopback accepts the daemon's
  self-signed certificate as the pi packages do (`rejectUnauthorizedFor`).
  HTTP 4xx becomes `isError: true` with the daemon's message verbatim.
- Families and routes: `computer` → `/api/computer/tool`; `browser` →
  `/api/browser/tool`; `inbox` → `/api/inbox` (`ask_human` polls the item
  until answered, capped, and says so when it gives up); `checklist` →
  `/api/terminals/{id}/checklist` (agent form when `PICODE_AGENT_ID` is set).

### Registration

- Catalog presets in `internal/mcp` (`Entry{Command: "picode", Args:
  ["mcp", "<family>"]}`), grouped "PiCode tools" at the top of the Add
  connector dialog for every CLI; the pane's existing "Save to" chooses
  machine or workspace; Pi's own catalog hides them (pi has the packages).
- Launch settings: `clilaunch.Config` gains `Tools []string`; the plan
  inspection (`Plan.Args`/`ManagedEnv`) shows what will be injected;
  `launchCLITerminal` adds the per-CLI mechanism; the terminal's launch
  override can add or remove families. UI: a "PiCode tools" group with one
  switch per family on the CLI's Launch settings page and in the terminal's
  launch override.

### Docs

`docs-site/guide/picode-mcp.md` (what it is, the three scopes, one page of
per-CLI registration lines, the honest per-CLI table), a paragraph in
`guide/computer-tool.md`, `guide/browser-tool.md` and `guide/mcp.md`;
`docs/architecture/picode-mcp.md`; changelog fragment.

## Roadmap

| Milestone | Scope | Gate | Owner acts |
|---|---|---|---|
| **N0 — server, presets, launch injection** (`feat/picode-mcp`) | ADR-0154; this plan; `cmd/picode` `mcp`; `internal/mcptool` (framing, `initialize`, `tools/list`, `tools/call`, `ping`; `computer`, `browser`); catalog presets; `clilaunch.Config.Tools` + injection for Claude Code, Codex, OpenCode; Launch settings UI; docs | Go: framing goldens, `tools/list` equals the pi schemas, `tools/call` against an `httptest` daemon, identity absent/terminal/agent, subcommand never starts the daemon; `make ci-scoped`. Real: Claude Code in a PiCode terminal, `/mcp` lists the tools, `computer screenshot` arrives as an image, the terminal's switch in Settings ▸ Computer turns it on and off | Approves the ADR; first real run with Claude Code |
| **N1 — inbox and checklist** | `inbox` and `checklist` families; the terminal card shows a guest's plan line | Per-family tests; an `ask_human` from Claude Code appears in the Inbox and the answer returns | Tests the Inbox |
| **N2 — the other guests** | Launch injection for Grok, Hermes, Muse, Antigravity, Omp where the vendor offers a mechanism; honest "workspace only" elsewhere | One test per driver | Picks the order |
| **Later, if it hurts** | Streamable HTTP endpoint in the daemon for CLIs off this machine; `docker_*` family; grant inherited from Launch settings (M5 of the computer plan) | | Decides |

Every milestone: `make worktree NAME=<branch>`, `make ci-scoped` while
iterating, `make close`, fast-forward on `main`, one handoff note (≤ 25
lines), a visual proof on a scratch instance when UI is touched. Deploy is
the owner's, in batches.

## Validation

- Go: `go test ./internal/mcptool/... ./internal/clilaunch/... ./internal/mcp/... ./cmd/picode/...`.
- Scratch (N0): `scripts/qa-scratch.sh start mcp`; a `claude` in a PiCode
  terminal with the Computer tool in Launch settings; no grant → refusal with
  the Settings path; grant → screenshot as image, `windows` + `focus` +
  `type` in Notepad; off → refusal again; every call a `computer.step` row
  with principal `term:<id>`; the same terminal with the preset saved at
  workspace scope shows the server once, not twice.

## Risks

- Image blocks in MCP results: Claude Code renders them; Codex and the others
  are confirmed in N0, the file path is the fallback.
- A blocking `ask_human` holds that CLI's MCP server until answered: capped
  and documented; the CLI's own tool timeout may fire first.
- Two `--mcp-config` sources (peer communication + tools) for one Claude
  launch: merged into one generated file; a test covers both present.
- `picode mcp` must never fall through to the daemon default (the binary's
  default is the server on the production data dir): dispatched first,
  tested.
- Same-name entries in launch and file scopes: collapse where the CLI merges
  by name; where it does not, the pane says the name already exists.

## Critical files

- Server: `cmd/picode/mcp.go` (new), `internal/mcptool/{server,computer,browser,inbox,checklist}.go` (new); moulds: `packages/pi-*/src/logic.ts` and their tests.
- Registration: `internal/mcp/mcp.go` (presets), `internal/connectors/*.go` (unchanged drivers), `internal/clilaunch/config.go` (`Tools`), `internal/server/{terminals,cli_plan,cli_profiles}.go`, `internal/communication/launch.go` (merge of the Claude file).
- Web: the Launch settings page and the terminal launch override (`web/browser/src/components/…`), `web/shared/domain/integrations.js` (preset group).
- Docs: `docs/decisions/0154-picode-mcp.md`, `docs-site/guide/picode-mcp.md`, `docs-site/guide/{computer-tool,browser-tool,mcp}.md`, `docs/architecture/picode-mcp.md`, `docs/changelog.d/picode-mcp.md`.
