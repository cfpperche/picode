# Pi settings GUI — plan

> Status: **S0 shipped** (2026-08-25). Knobs start in S1. Slash matrix:
> [`slash-parity.md`](slash-parity.md). Product vs pi split: ADR-0012.

## Goal

Composer `/settings` opens a **read/write GUI of pi settings**, scoped
to the current agent. PiCode-the-product (theme, port) leaves
`#/settings` and lives at `#/preferences`.

## Routes

| Hash | Owns | Opens from |
|---|---|---|
| `#/preferences` | PiCode: appearance, server port | User menu **Preferences** |
| `#/settings` | **pi** JSON, this agent | Composer `/settings`, user menu **Settings** (only with an agent selected) |

`/settings` with no agent selected: toast “Select an agent” — do not
open an empty pi page.

## Three layers (what the page shows)

Always **Global**. **Workspace** only if the agent is bound to a real
folder (not `ws_free`). **Agent** always.

```
┌─ Global (this machine) ──────────────────────────┐
│  ~/.pi/agent/settings.json                       │
├─ Workspace (this folder) ──── only if bound ─────┤
│  <workspace.path>/.pi/settings.json              │
├─ Agent (this pi process) ────────────────────────┤
│  SQLite agents row + live RPC where it exists    │
└──────────────────────────────────────────────────┘
```

Unbound agent (`ws_free`): Global + Agent. Their `work_path` is **not**
a workspace; we do not invent a project file there in v1.

**Agent ≠ session.** One agent is one live `pi` (ADR-0011). It holds
**N session files** (JSONL); only one is current. The Agent card is
config for **every session that process opens** (provider, model,
thinking, Full/Read-only, cwd). It is not “this chat”.

| Layer | Shared by |
|---|---|
| Global | all pi on this machine |
| Workspace | all agents whose cwd is that folder |
| Agent | all sessions of that agent |
| Session (not on this page) | this JSONL only: name, tree, compact *this* context |

`/new` `/resume` `/name` `/compact` `/tree` stay session verbs.
Settings does not grow a fourth card in v1.

Effective value = merge (project overlays global, same as pi). GUI
shows the effective value and **which layer last wrote it**.

## What we expose (v1) vs leave in the TUI

**Write in GUI** (pi docs names):

| Knob | Layer | Persist | Live (if agent running) |
|---|---|---|---|
| Auto-compact | global / workspace | `compaction.enabled` | RPC `set_auto_compaction` |
| Steering mode | global / workspace | `steeringMode` | RPC `set_steering_mode` |
| Follow-up mode | global / workspace | `followUpMode` | RPC `set_follow_up_mode` |
| Default provider / model / thinking | global / workspace | `defaultProvider` `defaultModel` `defaultThinkingLevel` | — (next start) |
| Transport + HTTP idle | global / workspace | `transport` `httpIdleTimeoutMs` | next start |
| Hide thinking blocks | global / workspace | `hideThinkingBlock` | next start |
| Skill commands | global / workspace | `enableSkillCommands` | `/reload` |
| Block / auto-resize images | global / workspace | `blockImages` `imageAutoResize` | next start |
| Default tools | global / workspace | `defaultTools` | next start (`--tools` / Full vs Read-only still agent) |
| Scoped models | global / workspace | `enabledModels` | next start |
| Default project trust | **global only** | `defaultProjectTrust` | — |
| Agent provider / model / thinking / Full\|Read-only | **agent** | SQLite | already PATCH + restart |

**Do not reimplement** (TUI chrome, stay in the dock): theme of the TUI,
editor/output padding, hardware cursor, OSC progress, `tuiMode`,
fullscreen, quiet startup, changelog collapse, double-escape, tree
filter, mermaid-as-unicode.

**Still other routes:** auth → `#/providers` (ADR-0009). Packages →
`#/packages` (ADR-0010). MCP → `#/mcps`. Secrets never on this page.

## Write rules

1. **Read-modify-write** the JSON object. Unknown keys stay. We are not
   a formatter of the user's file.
2. There is **no** `pi settings set` CLI. Persistence is the JSON files
   pi already owns. Live knobs also RPC when the agent is up.
3. Workspace writes go to `<path>/.pi/settings.json` (create `.pi/` if
   needed). Same trust story as `pi install -l`: if the project is not
   trusted, refuse and say so — do not bypass `trust.json`.
4. After a JSON write, if the agent is running, send `/reload` (or the
   RPC equivalent) for knobs that only apply on reload; otherwise the
   page says “applies on next run”.
5. Global write is visible to **every** pi on this machine. The UI
   copy must say that. Workspace write is this folder. Agent write is
   this row.

## API (sketch)

```
GET  /api/pi-settings?agentId=
     → { global, project | null, agent, effective, writable: {global, project, agent} }

PUT  /api/pi-settings
     body: { agentId, layer: "global"|"project"|"agent", patch: { ... } }
```

`layer: agent` is today's `PATCH /api/agents/{id}` (no second store).
`project` returns 400 for unbound agents.

## UI

One page, three stacked cards (Global / Workspace / Agent). Workspace
card omitted for unbound agents. Each control: value, layer badge
(`machine` / `folder` / `agent`), save on change (no giant form).

Composer `/settings` → `go("settings")` with the current agent id in
the hash or selection (selection already is the agent).

User menu: **Preferences** (old Settings), **Settings** (pi, needs
agent).

## Phases

| Phase | Ship |
|---|---|
| **S0** | Rename `#/settings` → `#/preferences`. User menu + palette + `go()`. Empty `#/settings` shell. Composer `/settings` routes here. Matrix row → **ui** (shell). |
| **S1** | GET global JSON. Show + write the v1 knobs (auto-compact, steer/follow-up, defaults). Live RPC when running. |
| **S2** | Project layer for workspace agents. Trust-aware. |
| **S3** | Agent card uses existing PATCH. Scoped models + defaultTools. |
| **S4** | Remaining slash rows that are settings (`/scoped-models` → this page). |

## Out of scope

- Cloning the 29-row TUI list.
- Editing `packages[]` here (gallery).
- Editing `auth.json`.
- Session-only overrides (this run) — still deferred.

## Risks

- Live pi may rewrite `settings.json` under us → read-modify-write +
  tests with a fixture file, never the developer's real auth.
- Writing project JSON in an untrusted folder is a security fail —
  reuse trust checks; if unknown, 409 + copy to run `/trust` in the TUI.
- Users confuse PiCode Preferences (light/dark) with pi Theme (TUI
  colors). Labels: **PiCode appearance** vs **Pi defaults**.
