# Connectors pane — UX refinement plan

Status: proposed (not implemented). Benchmark study with the receipts and the
measurements: [../benchmarks/2026-09-12-connectors-ux.md](../benchmarks/2026-09-12-connectors-ux.md).
Extends the connector decision table in [integrations.md](integrations.md) —
it does not replace it. No ADR: this crosses no protocol, persistence,
security-model or process boundary (AGENTS.md, ADR rule).

## Goal

One screen that answers three questions in order — *what is connected*, *is it
working*, *how do I add another* — instead of a catalog grid that answers
only the third and makes the first two scroll.

Nothing under the API changes: `GET/POST/DELETE /api/mcp`, the config layers
(`~/.pi/agent/mcp.json`, `.mcp.json`, host imports), the `scope` semantics,
OAuth through the adapter, the definition-file import and the connection
decision table all stay as they are.

## Target shape (desktop pane, `#/clis/pi/connectors`)

```
Connect services your agents can use.                [+ Add connector]
                                                     ^ one primary action

CONFIGURED SERVICES · 3
┌───────────────────────────────────────────────────────────────────────┐
│ ● Live   context7   MACHINE   https://mcp.context7.com/mcp   [◉] [⋯]  │
│ ● Failed notion     MACHINE   https://mcp.notion.com/mcp     [◉] [⋯]  │
│ ○ Off    local-files MACHINE  npx -y @modelcontextprotocol/… [○] [⋯]  │
└───────────────────────────────────────────────────────────────────────┘
   (empty) No connectors yet.            [+ Add connector]

CONNECTOR PACKAGES · 1        ← only when a package ships pi.mcp (unchanged)
```

### Pane

- **`[+ Add connector]`** — the single primary action, in the header row
  beside the intro line. Keyboard: it is the first focusable control.
- **Empty** — "No connectors yet." + the same button. One line, one action.
- **Stopped agent** — one line above the list: *Live status comes from a
  running agent.* Rows then carry **no** status chip. Today every row says
  `Idle`, a state that means nothing when nothing runs.
- **Blocked (adapter absent)** — unchanged and already correct: "Install the
  MCP adapter to connect services." + *Open packages*; no catalog, no pills.
- **Error** — "Couldn't load connectors." + *Retry* (unchanged).
- Search over the configured rows appears at **> 6 rows** (same threshold
  language as the terminals list), never as chrome on a short list.
- The catalog grid, the raw file input and the two unlabelled pill rows leave
  the pane (they move into the dialog below).

### Row anatomy

`● status` · `name` · `scope tag` · `target (mono, truncated, title=full)` ·
`[switch]` · `[⋯]`

| Piece | Spec |
|---|---|
| Status | Dot + word: `Live` (ok), `Failed` (danger), `Sign in` (**button**, the next action), `Off` (muted, row dimmed). No status chip at all when the agent is stopped. Titles keep today's wording (*Connected*, *Last connect failed*, *Needs sign-in*). |
| Switch | Radix `Switch.Root` (`.rx-switch`, already a dependency) with `aria-label="Enable <name>"`, at `--ctl-h`. Replaces the `On`/`Off` text button that is styled as a scope pill today. |
| `[⋯]` | Radix `DropdownMenu` (`.um-popover`): *Sign in* / *Sign out* when applicable; *Remove* (danger, keeps today's confirm naming the config file). Nothing invented: no retry we cannot perform, no tool list we never fetched. |
| Scope tag | Unchanged (`.pkg-scope-tag`), now the only scope claim in the pane. |
| Target | Unchanged text, `title` with the full url/command vector. |

### Add connector dialog

Desktop: `ResponsiveDialog.jsx` (centred ≥ 720 px, bottom sheet below);
mobile: `MobileSheet.jsx` (ADR-0046/0072). One dialog, three entries:

```
Add connector
[ Search services…                          ]  Save to: (This machine)(QA)(This agent)
┌───────────────────────────────────────────────────────────────────────┐
│ Context7    Look up current library documentation and examples.  [Add] │
│ DeepWiki    Ask questions about public GitHub repositories.     [Added]│
│ …                                                                      │
└───────────────────────────────────────────────────────────────────────┘
  Custom server…        Import a file…          ← quiet secondary actions
```

- **Service rows** (presets + host imports from `connectorTabs(found)`), one
  per line: name, summary, action. `Added` (disabled) when the service exists
  in the **selected** scope — the same fact the scope tag shows on the row.
- **Search** filters name + summary, client-side.
- **Source tabs** (Catalog / Claude Code / …) stay, but as the repo's
  `pkg-tabs` underline tabs with a visible label *Source*, not a second pill
  row that looks like the scope row.
- **Save to** reuses the Packages install-bar pattern (2026-09-11): labelled
  control beside the action, `--ctl-h`, `data-align-row`. The selected scope
  is stated at the moment of writing, not implied by a sticky pill.
- **Custom server…** opens today's `mcpAddSchema` form dialog unchanged
  (progressive disclosure — the transport form stops competing with the
  directory).
- **Import a file…** opens the file picker *inside* the dialog and keeps the
  existing review-then-add confirmation (`readConnectorDefinition`, the trust
  copy, the 64 KB bound, the refusal paths). The native input is styled behind
  a `.btn` label; the browser's "No file chosen" string never appears.

### Cross-cutting

- Remove `settings-ctx` from `Mcps` in both apps (the same fix the Packages
  pane shipped); the pane owns the 12 px gap under the tab bar.
- Delete the unreachable non-embedded branch (`title "MCPs"`, `.mcp-presets`,
  `mcps-view`): `#/mcps`, `#/integrations*` and both `ConnectorsPane`s render
  `embedded` only, so every later edit is single-path. The route redirect
  stays.
- One control height (`--ctl-h`) for every control in the pane — switches,
  pills, tabs, buttons, the file-label. Today the same `.pkg-scope-btn` class
  is 28 px in one row and 36 px in another.
- Both apps, one branch: `web/desktop/src/components/Mcps.jsx` +
  `web/mobile/src/components/Mcps.jsx`, and their own
  `styles/integrations.css` / `components/agent-clis.css` (ADR-0072: no shared
  stylesheet).

## Phases

| Phase | Deliverable | Files |
|---|---|---|
| **P0 — chrome and rhythm** | ctx line gone, pane gap, one control height, real switches, labelled control groups, empty/blocked states with one action, dead branch deleted | both `Mcps.jsx`, both `integrations.css`, both `agent-clis.css` |
| **P1 — Add dialog** | catalog + import + custom + target into one `ResponsiveDialog`/`MobileSheet` with search, service rows and the "Save to" control | both `Mcps.jsx`, new `AddConnectorDialog.jsx` per app, `web/shared/domain/integrations.js` (filter helper + vitest) |
| **P2 — row information** | status dot + word, inline *Sign in*, ⋯ menu, stopped-agent line, row count | both `Mcps.jsx`, `integrations.css` |
| **P3 — harness and docs** | `scripts/qa-cli-connectors.mjs`, `docs/architecture/mcp.md` + `integrations.md`, `docs-site/guide/mcp.md`, changelog fragment | `scripts/`, `docs/` |

P0 alone fixes the owner's screenshot complaint; P1 is the structural change;
P2 is where the pane becomes a control surface rather than a report.

## Decision table

| Conditions | Action | Verified by |
|---|---|---|
| Adapter absent | One line + *Open packages*; no catalog, no scope control, no add button in the pane body | browser: blocked state |
| Adapter present, 0 servers | "No connectors yet." + *Add connector* (primary) | browser: empty state |
| Adapter present, N servers | Rows + count in the heading; *Add connector* stays the header action | browser: configured state |
| Server disabled | Row dimmed, switch off, no status chip | browser: off row |
| Server needs OAuth sign-in | Inline *Sign in* action in the row (accent), not a word | browser: signin row |
| Server failed last connect | `Failed` chip; no retry action we cannot perform | browser: failed row |
| Agent stopped | One line above the list; no per-row status | browser: stopped agent |
| Catalog entry already in **the selected** scope | Action reads *Added*, disabled | browser: dialog duplicate |
| Add clicked with scope = project/agent | Request body carries that scope; the row appears tagged with it; POST asserted | browser + existing Go API tests |
| First add of an OAuth service | Existing sign-in flow starts after the write; failure is reported, never silent | browser: signin flow |
| Add/remove/toggle in flight | Existing job overlay; controls disabled meanwhile | browser: job overlay |
| Remove | Confirm dialog naming the config file (unchanged), then the row disappears | browser: remove |
| Imported file malformed / > 64 KB / multiple definitions / credential command | Existing refusals, unchanged | `integrations.test.js` (existing) |
| Host configs found (Claude Code, …) | Source tab inside the dialog; entries add through the existing import path | browser: source tab |
| Pane < 720 px / mobile | Dialog becomes a bottom sheet; row keeps switch + menu, target truncates | screenshots 1024/mobile 390 |
| Live status arrives from the feed | Rows update in place (existing `mcp.updated`/`mcp.config` subscription) — no new timer | browser: feed update |

## Verification

1. `make ci-scoped`, then `make close` on the branch; `make ci` on `main`.
2. New `scripts/qa-cli-connectors.mjs` on a `qa-scratch.sh` instance, in the
   shape of `qa-cli-packages.mjs`: blocked (fixture HOME without
   `pi-mcp-adapter` in `~/.pi/agent/settings.json`), empty (`mcp.json` `{}`),
   configured (url + stdio + disabled + oauth rows), the Add dialog with a
   workspace target (assert the POST body `scope`), duplicate → *Added*,
   remove, and one narrow pass at 1024 px plus mobile 390 px.
3. Screenshots of empty, blocked, configured, dialog open and the job overlay,
   read with the `read` tool (`var/screenshots/connectors-ux/`), plus
   `window.__picodeOverlayAudit()` `ok: true` in each.
4. Measured in the DOM: every control in the pane at `--ctl-h` (36 px), row
   heights equal, no clipped tile or truncated control (the 18 px file input
   and the 28 px pills are the current violations).
5. No Go change expected: `internal/mcp/*_test.go` and
   `internal/server/mcp_test.go` must stay green untouched. A filter helper in
   `web/shared/domain/integrations.js` gets a vitest.

## Out of scope

- Tool-level toggles and any per-tool UI (the adapter owns discovery and
  namespacing; [architecture/mcp.md](../architecture/mcp.md)).
- The marketplace, package installation and `pi.mcp` package rows beyond the
  existing *Connector packages* section.
- Webhooks (`#/integrations`), provider accounts, terminal surfaces.
- Any change to scopes, precedence, file formats, OAuth or the MCP API.

## Open questions (owner)

1. **Add flow**: one dialog with a searchable list (recommended — it is what
   makes "Save to" explicit and removes the 78 %-of-pane grid), or keep the
   inline catalog and only fix the chrome (P0/P2 without P1)?
2. **Stopped agent**: one line above the list (recommended) or keep a per-row
   word for live status when nothing is running?
3. **Mobile**: ship both apps in one branch (recommended, ADR-0072 keeps them
   consistent), or desktop first and mobile as a follow-up?
