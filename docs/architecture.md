# Architecture

> Status: v0.1 — evolves with the project. Last reviewed: 2026-08-27.
> Changing anything described here requires updating this file (see [AGENTS.md](/AGENTS.md)).

## The one-paragraph version

PiCode is a **single Go binary that serves a browser UI** and manages **real
`pi` processes** on the machine where it runs. Each agent gets a dual channel:
a tmux-backed PTY that renders the genuine Pi TUI inside a browser terminal,
and an RPC bridge (`pi --mode rpc`, JSONL over stdio) that feeds the rich UI
with structured events. A broker routes messages between agents through a Pi
extension, so agents talk to each other using Pi's own tool-calling protocol.

## Application routes

The SPA has **two shells** in one Vite app (`web/src/desktop`, `web/src/mobile`),
picked at boot by viewport (`max-width: 767px`) or `?desktop=1` / `?mobile=1`.
Shared code lives in `web/src/lib`. Rotate does not remount (would drop WS).
The mobile shell is a PWA (`manifest.webmanifest`, `sw.js`, Apple
`apple-mobile-web-app-capable`) so Add to Home Screen opens full screen.

Hash routes (ADR-0012). **Preferences** is PiCode-the-product.
**Settings** is pi JSON for the selected agent. Auth, MCP, packages
stay on their own routes.

| Hash | Surface | Owns |
|---|---|---|
| `#/` | Agent workspace | tabs, chat, terminal, per-agent provider/model/thinking |
| `#/settings` | pi config | global + workspace + agent (composer `/settings`) |
| `#/preferences` | PiCode chrome | theme, notifications, server port, **backup** (ADR-0014); tabs `#/preferences/<section>` |
| `#/system` | Machine facts | host, network, deps, version (read-only) |
| `#/providers` | Pi providers | catalog + signed-in state; Sign in → TUI `/login` |
| `#/mcps` | Pi MCP | adapter manager: list / add / toggle / remove / **Use from…** (mirror host configs; Off hides a server). |
| `#/packages` | Pi packages | machine / workspace (`pi install`) / this agent (`-e` on start) (ADR-0010). Same agent context as MCP. |
| `#/devices` | Connected browsers | host vs LAN/tailnet phones (presence ping) |

Composer `@` lists files in the agent cwd (`GET /api/agents/{id}/files`).
Paste/drop images send `POST /api/agents/{id}/prompt` (live RPC, not the task table).
`!cmd` runs in the agent cwd via `POST /api/agents/{id}/bash` (`abort_bash` cancels); output renders in the chat and joins the next prompt.
MCP manager: `GET/POST/PATCH/DELETE /api/mcp` reads and writes the adapter files
(`~/.pi/agent/mcp.json`, `<cwd>/.mcp.json`, `<agent cwd>/.pi/mcp.json`). Add accepts
optional `env`, `headers`, `auth` (`oauth`|`bearer`) and `bearerToken`.
Live status (`idle`/`live`/`failed`/`signin`) comes from the adapter snapshot when
the GUI agent is running (`-e` silent bridge). OAuth rows with tokens in the OS
keyring show **Sign out** (clears the keyring entry). Hide Sign in while signed in. **Sign in** always uses a short
`pi --mode rpc --no-session -e` (not a second agent — no session file, ADR-0006)
running headless adapter `authenticate()` (callback only, no paste). Pi does not
open the browser (WSL would spawn a second tab). Status returns the authorize URL;
the GUI `window.open`s it once so the callback can `window.close()` like Claude/Codex.
If the authorize URL's `redirect_uri` is not localhost, Sign in fails immediately (Linear's hosted callback would never reach PiCode). Authenticate registers `http://127.0.0.1:<port>/callback`.
Success HTML is PiCode's (logo + return to `#/mcps`). Add or On on an OAuth server
starts Sign in immediately. Tokens live in the OS keyring, keyed by server name on
this machine — not per agent. No native MCP.


Sessions are **pi JSONL files** (`~/.pi/agent/sessions/`). PiCode lists,
switches (`--session`), and **replays** them into the chat surface. History
is not copied into SQLite (ADR-0005).

Entry: user menu (Settings, Preferences, Providers, MCPs) and `Ctrl+K`.
QR in the sidebar brand opens a phone-share drawer (`GET /api/share`):
HTTPS + bind + reachable IP + cert SAN + mkcert CA. Missing checks
list the action; a QR is only drawn when every check passes.

## Component diagram

```
┌────────────────────────────────────────────────────────────┐
│ Browser                                                     │
│  ├─ Rich UI (React + Vite + Tailwind — ADR-0008)           │
│  │   agents panel · tasks · diffs · sessions tree · auth   │
│  └─ xterm.js terminals (the real Pi TUI, 1:1)              │
└───────────────┬────────────────────────────────────────────┘
                │ HTTP /api/*  +  WebSocket /ws/*
┌───────────────▼────────────────────────────────────────────┐
│ picode (single Go binary, UI embedded via go:embed)        │
│                                                            │
│  AgentManager ─── spawn/stop/restart ──► pi processes      │
│  TerminalBridge ─ tmux sessions ───────► pi (interactive)  │
│  RPCBridge ────── JSONL stdio ─────────► pi --mode rpc     │
│  TaskQueue ────── steer / follow_up ───► RPCBridge         │
│  Broker ───────── inbox routing ───────► PiCode extension  │
│  SessionReader ── parse ~/.pi/agent/sessions/*.jsonl       │
└────────────────────────────────────────────────────────────┘
```

## Why dual channel per agent

Revised by ADR-0006: **one live pi process per agent, two exclusive run
modes** — interactive (tmux TUI) or managed (rpc + panel). The original
simultaneous design risked concurrent writers on pi's session files.

| Channel | Carries | Fails to | Cost |
|---|---|---|---|
| tmux + PTY (`/ws/term/:agent`) | Full Pi TUI: interactive login, any command, escape hatch | provide structured data | ~zero — tmux does the work |
| RPC (`pi --mode rpc`) | Structured events: streaming, tool calls, state | interactive dialogs (needs UI mapping) | protocol discipline |

Both channels read/write **the same session files** (`~/.pi/agent/sessions/`),
so the user can alternate between the rich view and the terminal without
losing state. Closing the browser tab does not kill anything: the interactive
agent lives inside tmux; the RPC agent is detached from the browser entirely.

## Key subsystems

### Data & persistence (ADR-0005)

SQLite (pure-Go driver) at `~/.picode/picode.db` — **orchestration overlay
only**. Pi's own files remain the source of truth for sessions, credentials,
MCP and skills; PiCode never duplicates them. Schema v1: `workspaces`,
`agents` (many per workspace, own model/config; free agents in `ws_free` — ADR-0011),
`tasks` (prompt/steer/follow_up queue with a delivery state machine),
`messages` (reserved M4 broker inbox), `events` (orchestration audit),
`settings`. Embedded sequential migrations; the M1 JSON registry is imported
once and retired (`workspaces.json.migrated`).

### AgentManager (M1 core shipped)
Owns agent lifecycle via the SQLite store (`internal/store`): workspaces
each with a default agent; tmux session names derive from the agent id.
An agent runs as `pi` (ADR-0003, user-installed) in a named tmux session.
Per-agent provider/model/thinking is stored on `agents` and passed as
`pi --provider/--model/--thinking` on start (ADR-0009). Auth stays in
`~/.pi/agent/auth.json`; PiCode never collects keys.

HTTP API (Go 1.22 method patterns):
- `GET/POST /api/workspaces` — list (with live `running` flag) / add
- `DELETE /api/workspaces/{id}` — remove (stops **all** agents first).
  Optional `?sessions=1` deletes the pi session dir when this workspace is
  the last occupant of that cwd. Project folders are never deleted.
- `GET /api/workspaces/{id}/cleanup` / `GET /api/agents/{id}/cleanup` —
  preview for the delete dialog (session count, last occupant, owned work folder).
- `DELETE /api/agents/{id}` — unregister. Optional `?sessions=1&work=1`
  (work only if cwd is under `~/.picode/work/` and nobody else uses it).
- `POST /api/workspaces/{id}/open|close` — start/stop the pi agent (idempotent)
- `GET /api/system` — pi/tmux detection + setup warnings (ADR-0003 UX)
- `GET /ws/term?session=<name>` — xterm.js bridge

### TerminalBridge ✅ (M1)
One tmux session per interactive agent (`internal/tmux`: create/kill/list,
exact-name matching via `=` prefix, PiCode-owned `picode-` name namespace,
ids sanitized — dots/colons are tmux target separators and corrupt
lookups). `internal/term` bridges WebSocket ↔ PTY (`tmux attach`):
binary frames = terminal bytes, text frames = `resize` control JSON;
closing the tab ends only the attach — the agent keeps running in tmux.
Resize propagates via `TIOCSWINSZ` on the attach PTY. Requires tmux ≥ 3.5
with `extended-keys on` / `extended-keys-format csi-u` (Pi's own
recommendation) so modifiers like `Shift+Enter` survive the hop;
`/api/system` detects and warns.

### RPCBridge ✅ (M2 core)
`internal/rpc`: JSONL client for `pi --mode rpc` (strict `\n` framing via
bufio.Scanner, command/response correlation by id, event fan-out, exit
propagation) plus the **managed runtime** (ADR-0006): task delivery engine
claiming from the store (`prompt`/`steer`/`follow_up` → rpc commands,
gated on `agent_settled`, finished delivered/failed with audit), per-agent
event hub feeding `GET /ws/agent?agent=<id>` (events + `enqueue` input).
Extension dialogs (`select`/`confirm`/`input`/`editor`) surface as RPC
`extension_ui_request`. Managed snapshot includes `waiting` + `dialog`.
The GUI shows a conversation card; **Yes / No / Cancel / type** POST
`/api/agents/{id}/ui` (`ReplyUI`). Notify is a toast, not waiting.
Timeout on the request dismisses the card (pi auto-resolves).
Auto-approve policy stays undecided. Track C:
[conversation-control-roadmap.md](design/conversation-control-roadmap.md).

### Broker (M4)
A Pi extension (`picode-extension`, TypeScript, installed per workspace)
registers tools `send_message` / `read_inbox` that call PiCode's local HTTP
API. The broker delivers messages as `follow_up` prompts to the target agent.
Agents communicate through their native protocol — no internals hacked.

### SessionReader
Parses Pi session JSONL files (version 3, tree-structured via `id`/`parentId`)
to render session history, branching and diffs in the UI. Read-only.

### MCP (Model Context Protocol) support

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

### Security model (ADR-0007 — supersedes the original localhost-only clause)
- **HTTPS always** (bind 0.0.0.0): mkcert-issued cert via
  `scripts/setup-cert.sh` (SANs: localhost + LAN + tailscale; CA exported to
  the Windows trust store on WSL) or a generated self-signed cert as the
  zero-config bootstrap. `PICODE_INSECURE=1` disables TLS (dev only).
- **Port**: default range `8445-8455`, first free port wins; **editable in
  the Settings UI at runtime** (graceful rebind: bind-new-first, revert on
  failure — see ADR-0007). Precedence: UI/DB > `PICODE_PORT` env > default.
  Discovery: `~/.picode/server.json`.
- **Trust boundary**: personal machine / tailnet (same stance as agentdeck);
  no app-level auth — token auth is a recorded debt if ever exposed beyond
  the tailnet. PiCode executes with the user's permissions, like Pi itself.

## Explicit non-goals

- Re-implementing the Pi TUI (we embed it instead).
- Sandboxing agents (that's Pi's trust model, not ours).
- Hiding Pi: every GUI action maps to something inspectable in the terminal.
