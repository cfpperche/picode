# ADR-0150: Connectors (MCP) management for every agent CLI

- **Status**: accepted (owner approved the plan and its four decisions, 2026-09-17)
- **Date**: 2026-09-17
- **Boundary**: persistence — PiCode writes other CLIs' native MCP configuration files; process — PiCode runs vendor CLI commands (`claude mcp add`, `codex mcp login`, …) on the user's behalf; security model — credentials stay in each CLI's own store and are never read, copied or moved by PiCode.

## Context

Connectors today means Pi only: `internal/mcp` edits `pi-mcp-adapter` config
files and the Connectors pane lives at `#/clis/pi/connectors`
(`CLI_CONNECTORS` in `web/shared/domain/integrations.js` is a one-entry list).
The product direction is a multi-CLI ADE (ADR-0069), and every managed CLI —
Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity and
Omp — already supports MCP natively. The gap is management, not protocol:
each CLI keeps servers in one of four shapes (JSON `mcpServers`, TOML
`[mcp_servers.*]`, YAML `mcp_servers`, the `opencode.json`/`settings.json`
`mcp` block) with divergent field names (`serverUrl` vs `url`, `command[]` vs
`command`), scopes and toggle mechanisms.

Two accepted decisions stand in the way and are hereby amended:

- ADR-0069: "PiCode does not implement other CLIs' authentication, package
  managers or agent protocols." Editing their native MCP config and shelling
  out to their own auth commands is an explicit, bounded exception — it
  changes no protocol and implements no auth; the vendor binary stays the
  authority.
- ADR-0075: "Native MCP config remains authoritative, no second connection
  database." That principle is extended, not replaced: each CLI's own config
  is authoritative for that CLI. PiCode still keeps no connection database.

Owner-approved decisions (2026-09-17):

1. PiCode writes other CLIs' native config files with merge-by-key, atomic
   writes, re-read-before-write, and refusal when the file changed underneath.
2. For Claude Code's user scope (the live `~/.claude.json`), the write path is
   the vendor's own command (`claude mcp add/remove … --scope user`) executed
   in the user's terminal context; plain files (`.mcp.json`, TOML, YAML) are
   edited directly.
3. Parsing gains exactly two dependencies (TOML and YAML); each is justified
   in the PR that adds it.
4. CLIs without a headless status signal (Muse, Antigravity outside a
   session) report "configured" honestly; PiCode never invents live state.

## Decision

Every agent CLI's Connectors pane at `#/clis/<cli>/connectors` manages that
CLI's native MCP servers through a per-CLI driver behind one interface
(layers, list, add, toggle, remove, probe). The driver for Pi remains the
existing `internal/mcp` implementation. Guest-CLI drivers live in
`internal/connectors`, one codec per config shape, each preserving unknown
keys, comments where the format allows them, `${VAR}` placeholders and
`!command` indirections verbatim. Remote-server sign-in always delegates to
the vendor command run in a PiCode terminal; tokens are never parsed, merged,
imported or copied across CLIs, and API responses redact secret-shaped
values as `internal/mcp` does today. The pane, Add dialog, catalog presets,
host imports and status vocabulary stay one surface; only the driver behind
it changes per CLI.

## Consequences

Users get one connector surface across all nine CLIs, and a connector added
for one CLI can be reused in another through the existing definition import
without retyping. Every guest CLI adds a codec with golden-file tests and a
probe strategy; CLIs with no machine-readable status cap the pane at
"configured", which is honest but less informative than Pi's live report.
Writing vendor files makes PiCode a participant in files other programs
write: the re-read-before-write contract and the HOME-swapped test suite
(`internal/server/main_test.go`) are load-bearing, and a vendor format change
can break a codec between releases. The single shared pane keeps UX
consistent but means the slowest driver's capability set (e.g. no per-server
enable flag) bounds the common vocabulary; the pane shows what each driver
declares and nothing more. If a vendor rewrites its config format, that
CLI's pane fails loudly at parse time and refuses to write — corruption over
silence.

## Alternatives considered

- One PiCode-owned MCP registry shared by all CLIs, injected at launch:
  rejected — it contradicts ADR-0069's terminal-only contract for guests,
  duplicates each CLI's own loader, and breaks every launch made outside
  PiCode.
- Vendor-style marketplace with hosted OAuth: rejected — ADR-0075 refuses a
  marketplace and a credential vault; providers are external MCP servers.
- Per-CLI bespoke panes: rejected — eight near-identical surfaces to keep
  visually in sync for no capability gain.
- Read-only parity (list without add/toggle): rejected — the owners of the
  connectors-ux plan made add/toggle/remove the pane's core contract; a
  read-only guest pane would be a fabricated control.
