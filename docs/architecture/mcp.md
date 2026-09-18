# MCP (Model Context Protocol) support

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Pi has **no native MCP** — a deliberate design choice (tool definitions burn
context; Pi prefers CLI tools/Skills). PiCode adopts MCP through the
community **`pi-mcp-adapter`** extension (`pi install npm:pi-mcp-adapter`):

- One proxy tool (~200 tokens) instead of hundreds of definitions;
  lazy server startup; on-demand discovery.
- Reads standard configs (`.mcp.json`, `~/.config/mcp/mcp.json`,
  `~/.agents/mcp.json`) and **imports host configs** (Cursor, Claude Code,
  Codex) — a migration path for users arriving from Cursor.
- PiCode's value-add (M3–M4): a visual MCP Server Manager per workspace and
  per agent (enable/disable, precedence layers) writing the same config
  files the adapter reads. We orchestrate the ecosystem; we don't fork it.

## Connectors pane (2026-09-12, `docs/plans/connectors-ux.md`)

Every agent CLI gets this pane at `#/clis/<cli>/connectors` (ADR-0150):
one per-CLI driver behind `/api/mcp?cli=<id>` manages that CLI's native MCP
config — Pi's driver is the adapter implementation below. Since phase 1
(2026-09-17) the guest drivers for **Claude Code** and **Codex** live in
`internal/connectors` and answer through the same handlers with the same
`Report` JSON shape: the pane does not branch on the CLI. Each driver
mirrors `internal/mcp`'s types, preserves unknown keys and `${VAR}`
placeholders verbatim, writes atomically, and reports honestly: no headless
status signal, so guest rows never claim live state.

- **Claude Code** — project scope is the workspace `.mcp.json`, edited
  directly as JSON; user scope is Claude Code's own store, so the vendor CLI
  (`claude mcp add/remove/list … --scope user`) is the only read/write path
  and `~/.claude.json` is never opened. Claude has no per-server switch, so
  driver `toggle: "none"` removes the pane's switch and a toggle request is
  refused ("Claude Code connectors turn on and off in Claude Code; remove
  instead"). Sign-in is likewise Claude Code's: the pane shows the vendor
  command (`claude mcp login <name>`) as copyable text instead of a
  PiCode-driven flow.
- **Codex** — `~/.codex/config.toml` and `<workspace>/.codex/config.toml`
  (the folder is read only when it exists; Add may create it). Edits are a
  surgical splice: only the `[mcp_servers.<name>]` table span is replaced or
  deleted — comments, unrelated tables and placeholders survive
  byte-for-byte (golden-tested) — the result is re-parsed before it lands,
  and `enabled` is the real per-server switch (`toggle: "entry"`).

Since phase 2 (2026-09-18) two more JSON-codec drivers share the
merge-by-key rewrite (`map[string]any`, 2-space indent, re-read before
write, atomic rename) and never shell out — both scopes are plain files:

- **Omp** — `~/.omp/agent/mcp.json` and `<workspace>/.omp/mcp.json`. The
  document's own keys (`$schema`, `disabledServers`, `enabledServers`) and
  unknown entry fields (`timeout`, `auth`, `oauth`, …) survive every write;
  the entry's `enabled` field is the toggle. Omp's OAuth is TUI-only, so
  the sign-in hint is a copyable TUI command (`/mcp reauth <name>`), not a
  terminal one.
- **Antigravity** (`agy`) — `~/.gemini/config/mcp_config.json` and
  `<workspace>/.agents/mcp_config.json`. Remote servers ride `serverUrl`;
  `disabled` is the toggle and Antigravity's own fields
  (`authProviderType`, `oauth`, `disabledTools`) are preserved. An entry
  keeping its URL under the legacy `url` key still displays but reports
  `owned: false`, and Toggle/Remove refuse it ("legacy entry managed in
  Antigravity"); an Add over the same name converts it to `serverUrl`. Its
  sign-in hint is plain text — no command exists to copy.

Guest sign-in hints are declared per driver (`AuthHint` in Go,
`signIn` in `CONNECTOR_DRIVERS`): copyable for claude-code, codex, omp,
opencode, muse and hermes, plain text for agy and grok.

Since phase 3 (2026-09-18) two more codecs complete the set of shapes so
far, one JSON, one TOML:

- **OpenCode** — `$XDG_CONFIG_HOME/opencode/opencode.json` (default
  `~/.config/…`) and the workspace's own `opencode.json`. The document's
  `mcp` block holds `{name: {type: "local"|"remote", command: […]|url,
  enabled?, environment?, …}}`: the command array round-trips as an array,
  `environment` carries the entry's env, `enabled` is the toggle, and
  everything outside the block (`$schema`, `theme`, provider settings) plus
  unknown entry fields (`oauth`, `timeout`, …) survive every write. This
  driver manages only these two files — the session config PiCode generates
  for launches through `OPENCODE_CONFIG` is a different document and is
  never touched. Its sign-in hint is the vendor command
  (`opencode mcp auth <name>`).
- **Grok** — `~/.grok/config.toml` and `<workspace>/.grok/config.toml`,
  sharing Codex's surgical TOML splice (one helper in
  `internal/connectors/toml.go`): only the `[mcp_servers.<name>]` span is
  replaced or deleted, so comments, unrelated tables and
  `${VAR}`/`${VAR:-default}` placeholders survive byte-for-byte, and
  Grok's own fields (`startup_timeout_sec`, `tool_timeout_sec`, …) survive
  an update. Grok has no per-server switch, so driver `toggle: "none"`
  removes the pane's switch and a toggle request is refused ("remove and
  re-add instead"); OAuth happens on first use inside Grok, so its
  sign-in hint is plain text.

Since phase 4 (2026-09-18) the last two codecs close the set: every catalog
CLI now has a Connectors pane (ADR-0150 complete). Both keep a single config
file, so project scope refuses with a pointer to it instead of inventing a
per-workspace variant:

- **Muse** — the one JSON settings file at `~/.config/muse/settings.json`.
  Its `mcp_servers` block holds `{name: {transport: "stdio"|"streamable_http",
  command, args, env | url, headers, enabled?, mode?, …}}`: a URL entry
  rides `transport: "streamable_http"`, `enabled` is the toggle, and
  `schema_version`, `mode` and every unknown document/entry key survive
  every write (same JSON codec as the other drivers). Its sign-in hint is
  the vendor command (`muse mcp login <name>`).
- **Hermes** (`hermes`) — the one YAML config file at
  `~/.hermes/config.yaml`. Its `mcp_servers` block holds `{name: {command,
  args, env | url, headers, enabled?, auth?, tools?}}`; `enabled` is the
  toggle. YAML has comments and key order worth keeping, so this codec
  edits the document as a `yaml.Node` tree (`gopkg.in/yaml.v3`, ADR-0150
  decision 3) instead of a generic map: untouched keys keep their position
  and comments, entries' own fields (`auth`, `tools.include/exclude`, …)
  survive an update, `${VAR}` placeholders travel as plain scalars and are
  never interpreted, and a malformed file refuses with a clear error before
  anything is written. Its sign-in hint is the vendor command
  (`hermes mcp login <name>`).

`web/shared/domain/integrations.js` `CONNECTOR_DRIVERS` declares each CLI's
honest capability set (status `live` vs `configured`), and a request naming a
CLI without a driver fails loudly instead of writing Pi's files. Guest
host-config imports and the driven sign-in flow arrive in later phases; the
API refuses them with instructions (imports → "arrive in a later phase",
auth → the vendor's own `… mcp login <name>` command).

`#/clis/pi/connectors` is one surface in three bands: the header (the intro
line and the single primary action, **Add connector**), the configured roster,
and the connector-packages section when a package declares `pi.mcp`. `#/mcps`,
`#/integrations*` and both apps' `ConnectorsPane` render this view embedded —
there is no separate "MCPs" page.

The Add flow is one `ResponsiveDialog` (desktop) / `MobileSheet` (mobile): a
searchable service list over the catalog and the host imports
(`connectorTabs(found)`), a labelled **Save to** control, and two secondary
entries — the custom-server form (`mcpAddSchema`) and the definition-file
import, which keeps its review-and-confirm step. **Save to** is the pane's
scope: it writes `scope=user|project|agent` onto the route so a reload keeps
the target, and the add request carries the same value. Adding from a host app
keeps its confirmation.

A configured row carries the status (only while an agent runs — `Live`,
`Failed`, or the `Sign in` next action; with the agent stopped the pane says so
once instead of showing `Idle` per row), the scope tag whose title is the file
it lives in, the target, a Radix switch for enable/disable, and an overflow
menu (**Sign out**, **Remove**). Removal confirms by naming that file. Status
words come from the adapter's live report (`mcp.updated`) and the signed-in
store; the pane never claims a tool list it did not fetch.
