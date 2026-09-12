# Study: the Connectors pane — a directory and a roster, not a wall of tiles

- **Date:** 2026-09-12
- **Sources:** owner screenshot of `#/clis/pi/connectors` (2026-09-12 09:36,
  2559×1599 px, ~150 % scale); a live read of the same pane on a
  `qa-scratch.sh` instance (1600×1000, 942 CSS px of pane) with the DOM
  measured; live docs fetched 2026-09-12 —
  [VS Code, "Add and manage MCP servers"](https://code.visualstudio.com/docs/agent-customization/mcp-servers),
  [Cursor, "Model Context Protocol (MCP)"](https://cursor.com/docs/mcp),
  [Claude Code, "Connect Claude Code to tools via MCP"](https://code.claude.com/docs/en/mcp).
  In-house bars: [benchmarks.md](../benchmarks.md) (UI copy rule, one page
  width, density), [plans/integrations.md](../plans/integrations.md)
  (connector decision table), ADR-0075, ADR-0102, ADR-0103, the Packages
  install-bar refinement (`docs/handoff/2026-09-11-pkg-target-bar.md`) and the
  providers-roster study ([2026-09-11](2026-09-11-providers-density.md)).
- **Scope:** the Connectors pane's information architecture and chrome —
  `#/clis/pi/connectors` in both apps. The MCP API, the config files, the
  scope layers, OAuth and `pi-mcp-adapter` stay exactly as they are.
  `#/mcps` and `#/integrations*` already rewrite onto this pane.

## What the reference products do

| Product | Pattern | Receipt | PiCode adaptation |
|---|---|---|---|
| Cursor | MCP is managed from one **Customize** page; the marketplace is one-click install **with OAuth**; `mcp.json` is the escape hatch for everything else | "Install and manage MCP servers from the Customize page or configure them in mcp.json"; "Click *Add to Cursor* on a marketplace entry to install it and authenticate with OAuth" | One **Add connector** dialog: a searchable service list, then two quiet secondary entries (Custom server…, Import a file…). The file picker is never the first control. |
| VS Code | A **gallery** (Extensions view, `@mcp`) with `Install` and *Install in Workspace*; a **trust confirmation before the server starts**; *Configure Tools* toggles individual tools | "You can install an MCP server in your user profile or in your workspace… right-click the MCP server and select Install in Workspace"; "confirm that you trust the server to start it" | The target is **part of the action**, labelled ("Save to"), not a sticky unlabelled pill row; the first add of a service keeps a review step. |
| Claude Code | Named **scopes** (local / project / user) with a documented precedence and an **approval** step for project files; per-server **status**; disable without removing | "Each command writes to local scope unless you add `--scope project` or `--scope user`"; sections *Server status*, *Project server approvals and workspace trust*, *Disable a server without removing it* | Scope label on the configured row (already there) **and** on the control that writes; a status word per row plus a real switch. |
| Zapier (already cited by [plans/integrations.md](../plans/integrations.md)) | Status **plus the next action**; explicit removal consequences | in-repo acceptance table | The row carries its next action (`Sign in`) instead of a bare word; removal names the file. |
| providers roster ([2026-09-11](2026-09-11-providers-density.md)) | One row per item, aligned columns, measured empty runs | in-repo | The **configured** list is the hero; the catalog is a dialog, not the pane. |

## The problem, measured

The pane is `settings-ctx` → intro → **Configured services** → **Add
connector** → catalog grid. On the owner's window and on the scratch read,
the second half dominates and the first half says almost nothing.

| Measured | Value | Why it hurts |
|---|---|---|
| First child of the pane | `.settings-ctx` ("grok · PiCode", 28 px), 0 px below the tab rule | A second heading glued to the tabs; it repeats the workspace the sidebar and the scope pill already name. Same defect the Packages pane shed on 2026-09-11. |
| "Add connector" section | **476 px of 612 px** (78 %) | The catalog is the pane; the configured services are a footnote. |
| Catalog tiles | 7 equal weight, no logo, no publisher, no search | *Custom* (opens a form) looks identical to *Context7* (saves immediately) — two different consequences behind one tile. |
| Import control | `<input type="file">`, **18 px tall, 942 px wide**, native "Choose File / No file chosen" | A browser-default control half the control height, and it is the **first** thing under the heading — before the catalog it is meant to complement. |
| Scope pills | `This machine / QA / This agent`, **28 px**, unlabelled, directly above the **Catalog/Claude Code** pills (28 px, same class) | Two stacked groups styled identically, no labels: reads as one broken radio group. The first silently decides where the next click writes. |
| Configured row | 54 px: scope tag + name + `Idle` word + target (193 px, ellipsis) + `[On] [Remove]` | Status is a word with a tooltip; enable/disable is a text button that looks like the scope pill; the target is the first thing truncated. |
| Status vocabulary | `Idle` / `Live` / `Failed` / `Sign in` | With the agent stopped every row says `Idle` (title: "Not used yet"). Four states, one of which is not a state. |
| Row actions | `On`/`Off` **36 px** vs scope pills **28 px**, same `.pkg-scope-btn` class | One class, two heights, in one pane — the control-rhythm audit cannot see either. |

## What we refuse

- **Tool-level toggles** (VS Code *Configure Tools*): the adapter does not
  report the tool list to PiCode, and ADR-0102 keeps the adapter the owner of
  discovery. A toggle over data we never fetched is a fabricated control.
- **A marketplace with its own install path**: connectors stay native MCP
  definitions plus packages (ADR-0075); the catalog below is a preset list,
  not a second package manager.
- **Credential storage**: OAuth stays the adapter's; the pane only reports
  signed-in / needs sign-in and launches the existing flow.
- **A second width or a new page**: the pane keeps ADR-0103's geometry.

## What follows

The refinement plan (phases, the pane and dialog spec, the decision table and
the verification) lives in [../plans/connectors-ux.md](../plans/connectors-ux.md).
