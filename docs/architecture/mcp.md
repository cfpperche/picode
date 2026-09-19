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
placeholders verbatim, writes atomically, and reports honestly: a guest row
claims live state only when its vendor exposes a headless signal (below);
otherwise the row stays honestly "configured".

- **Claude Code** — project scope is the workspace `.mcp.json`, edited
  directly as JSON; user scope is Claude Code's own store, so the vendor CLI
  (`claude mcp add/remove/list … --scope user`) is the only read/write path
  and `~/.claude.json` is never opened. The `add` argv follows the shape the
  real CLI accepts (measured 2.1.277): name first, flags behind it, `--`
  before a stdio command — the vendor's `--env`/`--header` are variadic and
  swallow positionals that follow them. `claude mcp list` prints lines, not
  JSON; the parser keeps name + target, strips the health tail (glyphs and
  wording moved between releases) and marks URL-shaped targets as remote
  rows. Claude has no per-server switch, so driver `toggle: "none"` removes
  the pane's switch and a toggle request is refused ("Claude Code connectors
  turn on and off in Claude Code; remove instead"). Sign-in is likewise
  Claude Code's: the pane shows the vendor command (`claude mcp login
  <name>`) as copyable text instead of a PiCode-driven flow.
- **Codex** — `~/.codex/config.toml`, the one file the vendor loads
  (measured codex-cli 0.155.0: `codex mcp list` resolves only CODEX_HOME's
  config, so the old workspace layer was a fiction and project scope
  refuses). Edits are a surgical splice: only the `[mcp_servers.<name>]`
  table span is replaced or deleted — comments, unrelated tables and
  placeholders survive byte-for-byte (golden-tested) — the result is
  re-parsed before it lands, and `enabled` is the real per-server switch
  (`toggle: "entry"`; the vendor's own list reports the flag as
  enabled/disabled).

Since phase 2 (2026-09-18) two more JSON-codec drivers share the
merge-by-key rewrite (`map[string]any`, 2-space indent, re-read before
write, atomic rename) and never shell out — both scopes are plain files:

- **Omp** — `~/.omp/agent/mcp.json` and `<workspace>/.omp/mcp.json`. The
  document's own keys (`$schema`, `disabledServers`, `enabledServers`) and
  unknown entry fields (`timeout`, `auth`, `oauth`, …) survive every write;
  the entry's `enabled` field is the toggle. Omp's OAuth is TUI-only, so
  the sign-in hint is a copyable TUI command (`/mcp reauth <name>`), not a
  terminal one.
- **Antigravity** (`agy`) — `~/.gemini/config/mcp_config.json`, the one file
  the vendor resolves (measured agy 1.2.6: `agy mcp add/list` know no
  workspace config, so the old `.agents/` layer was unverifiable and
  project scope refuses). Remote servers ride `serverUrl`;
  `disabled` is the toggle — the same flag `agy mcp disable` writes — and
  Antigravity's own fields
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

- **OpenCode** — `$XDG_CONFIG_HOME/opencode/opencode.json[.c]` (default
  `~/.config/…`) and the workspace's own `opencode.json[.c]`. Measured
  opencode 1.18.31: the vendor merges both files and its own `mcp add`
  writes the `.jsonc`, which also wins a name conflict — so the pane lists
  every existing file as its own layer (jsonc first) and a mutation targets
  the file the name already lives in, else the `.jsonc` when it exists, else
  `opencode.json`. The document's
  `mcp` block holds `{name: {type: "local"|"remote", command: […]|url,
  enabled?, environment?, …}}`: the command array round-trips as an array,
  `environment` carries the entry's env, `enabled` is the toggle, and
  everything outside the block (`$schema`, `theme`, provider settings) plus
  unknown entry fields (`oauth`, `timeout`, …) survive every write. This
  driver manages only these config files — the session config PiCode
  generates for launches through `OPENCODE_CONFIG` is a different document
  and is never touched. Its sign-in hint is the vendor command
  (`opencode mcp auth <name>`).
- **Grok** — `~/.grok/config.toml` and `<workspace>/.grok/config.toml`,
  sharing Codex's surgical TOML splice (one helper in
  `internal/connectors/toml.go`): only the `[mcp_servers.<name>]` span is
  replaced or deleted, so comments, unrelated tables and
  `${VAR}`/`${VAR:-default}` placeholders survive byte-for-byte, and
  Grok's own fields (`startup_timeout_sec`, `tool_timeout_sec`, …) survive
  an update. `enabled` is the per-server switch (`toggle: "entry"`,
  measured grok 1.0.34): Toggle mirrors what `grok mcp disable/enable`
  write — the table flag plus, in the user file, the top-level
  `disabled_mcp_servers` overlay (the vendor's mechanism for project-scope
  entries; it never rewrites project configs, and neither does the driver's
  project toggle). Remove strips the name from the overlay of the file it
  edits, exactly what the vendor leaves behind. New entries carry
  `enabled = true`, the shape of the vendor's own add. OAuth happens on
  first use inside Grok, so its sign-in hint is plain text.

Since phase 4 (2026-09-18) the last two codecs close the set: every catalog
CLI now has a Connectors pane (ADR-0150 complete). Both keep a single config
file, so project scope refuses with a pointer to it instead of inventing a
per-workspace variant:

- **Muse** — the one JSON settings file at `~/.config/muse/settings.json`.
  Its `mcp_servers` block holds `{name: {transport: "stdio"|"streamable_http",
  command, args, env | url, headers, enabled?, mode?, …}}`: a URL entry
  rides `transport: "streamable_http"`, `enabled` is the toggle, and
  `mode` and every unknown document/entry key survive
  every write (same JSON codec as the other drivers). Muse's loader refuses
  a settings file without `schema_version` (measured Muse Code 1.3.0:
  "missing field `schema_version`"), so every write path keeps the key
  present — a file PiCode creates gains `schema_version: 1`, an existing
  value is never overwritten. Both the block key and the transport names
  were verified against the vendor's own `mcp login` command. Its sign-in
  hint is the vendor command (`muse mcp login <name>`).
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

## Live vendor parity (2026-09-18, `internal/connectors/live_test.go`)

Every driver was verified against the real vendor binaries installed on the
development machine, not only against vendor docs: add/toggle/remove round
trips in a sandbox HOME (`PICODE_CONNECTORS_LIVE=1`, sandbox equals the
process HOME, so nothing outside it is touched), with the vendor's own
commands as the oracle — `codex mcp get/list`, `agy mcp list`,
`opencode mcp list`, `grok mcp list`, `hermes mcp list`, `muse mcp login`,
and, for Omp, a marker file a spawned stdio server touches (Omp has no MCP
CLI; the marker proves it loads `~/.omp/agent/mcp.json` and the workspace
file, and honors `enabled`). The measured divergences became driver fixes
with fixture tests: Claude's add argv and line parsing, Codex's and AGY's
fictional workspace layers, Grok's real toggle, OpenCode's `.jsonc` store,
Muse's mandatory `schema_version`. What cannot be exercised headlessly
(OAuth completion, TUI-only flows, auth-gated sessions) is recorded in
`docs/handoff/open/connectors-parity.md`, never faked.

Reading and writing are held to different bars (ADR-0150, connectors-codec
robustness). Every guest `List` reads leniently and writes strictly: the
JSON codec tries strict JSON first, then strips the JSONC escapes vendors
hand-write (one trailing comma before a closer, `//` and `/* */` comments —
OpenCode leaves both in `opencode.json`) and retries; a rewrite through Add
still lands canonical strict JSON. A file that exists but parses under no
form blocks only its own layer, never the pane: the layer reports
`exists: true` plus a short `error` reason ("is not valid JSON" — the same
degradation for TOML and YAML), contributes no servers, and healthy layers
keep listing, so `GET /api/mcp` answers 200. The pane renders one blocked
line naming the file (`blockedLayers` in `integrations.js`) with **Open**
(`POST /api/mcp/reveal` — the server re-derives the path from the driver's
own layer list, so no path travels from the client) and **Retry**; every
write on top of an unreadable file still refuses — corruption over silence.

`web/shared/domain/integrations.js` `CONNECTOR_DRIVERS` declares each CLI's
honest capability set (status `live` vs `configured`), and a request naming a
CLI without a driver fails loudly instead of writing Pi's files. There is no
driven sign-in flow: auth points at the vendor's own `… mcp login <name>`
command. Definition-file import and host-config discovery are removed
(ADR-0157) — the Pi adapter keeps its own import, and with every CLI's native
config managed here, the marketplace replaces "use from another app".

## Guest live status (ADR-0150 decision 4, 2026-09-19)

Three vendors expose a headless status signal, and their drivers now surface
it on every pane load — the vendor's own verdict, never invented:

- **claude-code** — `claude mcp list` health-checks servers; the ✓ / ✗ / ✘
  tails the driver already parsed for display become the row's
  Live / Failed state (zero extra cost).
- **OpenCode** — `opencode mcp list` reports per-server status; ● rows are
  live, ✗ rows failed, disabled rows show no connection claim.
- **Hermes** — `hermes mcp list`'s status column: ✓ live, ✗ failed,
  "✗ disabled" is the enabled flag's own off state, not health.

Probes run per pane load with a 15s budget and a 60s cache (vendor CLIs fork;
a hung or missing CLI is skipped and the row stays "configured" — degradation
over invention). Codex, Grok, AGY, Muse and Omp expose no headless health and
stay "configured"; guest live state refreshes on pane load, not through the
feed (that stream remains the adapter's).

## Marketplace (ADR-0157, 2026-09-18)

The pane is **Installed | Marketplace** subtabs, the Packages pattern:
Installed is the roster as built above; Marketplace is the add surface.

- `internal/mcpcatalog` serves the catalog behind
  `GET /api/connectors/gallery?q=`: the **seed** — the hand-picked presets
  and PiCode's own `picode-*` connectors, featured first — plus a filtered
  sync of the **official MCP Registry** (status active, a description, and a
  streamable-HTTP remote; stdio beyond the seed stays hand-curated). The
  sync paginates the registry's v0 API, caches the merged list in the data
  dir, refreshes in the background at most daily, and degrades to the seed
  when the registry is unreachable — the pane answers offline.
- Cards carry capability badges (Remote / Local command / Sign-in required);
  **Add** builds the same POST `/api/mcp` body the old dialog rows built and
  runs it through the driver of the selected CLI. **Added** (disabled) marks
  a name already in the selected scope. **Docs** (when the registry sent a
  `websiteUrl` or repository URL, or the seed has a cookbook page) opens that
  page in the browser; a card without a URL hides the link. **Custom
  server…** stays as the manual escape hatch.
- The definition-import endpoint (`POST /api/mcp/import`), the file reader,
  and `Report.Found`/`Report.Imports` are gone, as are the guests'
  "arrive in a later phase" refusals — there is no later phase.
- The connector-packages section (packages declaring `pi.mcp`) stays on the
  Installed side, unchanged.

`#/mcps`, `#/integrations*` and both apps' `ConnectorsPane` render this view
embedded — there is no separate "MCPs" page.

A configured row carries the status (only while an agent runs — `Live`,
`Failed`, or the `Sign in` next action; with the agent stopped the pane says so
once instead of showing `Idle` per row), the scope tag whose title is the file
it lives in, the target, a Radix switch for enable/disable, and an overflow
menu (**Sign out**, **Remove**). Removal confirms by naming that file. Status
words come from the adapter's live report (`mcp.updated`) and the signed-in
store; the pane never claims a tool list it did not fetch.
