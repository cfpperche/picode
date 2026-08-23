# Architecture

> Status: v0.1 — evolves with the project. Last reviewed: 2026-08-23.
> Changing anything described here requires updating this file (see [AGENTS.md](/AGENTS.md)).

## The one-paragraph version

PiCode is a **single Go binary that serves a browser UI** and manages **real
`pi` processes** on the machine where it runs. Each agent gets a dual channel:
a tmux-backed PTY that renders the genuine Pi TUI inside a browser terminal,
and an RPC bridge (`pi --mode rpc`, JSONL over stdio) that feeds the rich UI
with structured events. A broker routes messages between agents through a Pi
extension, so agents talk to each other using Pi's own tool-calling protocol.

## Component diagram

```
┌────────────────────────────────────────────────────────────┐
│ Browser                                                     │
│  ├─ Rich UI (React or Svelte — see ADR-0004 when decided)  │
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

| Channel | Carries | Fails to | Cost |
|---|---|---|---|
| tmux + PTY (`/ws/term/:agent`) | Full Pi TUI: interactive login, any command, escape hatch | provide structured data | ~zero — tmux does the work |
| RPC (`pi --mode rpc`) | Structured events: streaming, tool calls, state | interactive dialogs (needs UI mapping) | protocol discipline |

Both channels read/write **the same session files** (`~/.pi/agent/sessions/`),
so the user can alternate between the rich view and the terminal without
losing state. Closing the browser tab does not kill anything: the interactive
agent lives inside tmux; the RPC agent is detached from the browser entirely.

## Key subsystems

### AgentManager
Owns agent lifecycle. An agent is defined by: workspace (cwd), model,
thinking level, config set (extensions/skills/templates). Persists a registry
in a local data dir (see ADR-0005 when persistence is designed).

### TerminalBridge
One tmux session per interactive agent. WebSocket attach/detach; tmux holds
scrollback and survives disconnects. Requires tmux ≥ 3.5 with
`extended-keys on` / `extended-keys-format csi-u` (Pi's own recommendation)
so modifiers like `Shift+Enter` survive the hop.

### RPCBridge
Speaks Pi's RPC JSONL protocol. Protocol notes that bit us already:
- **Strict framing: `\n` only.** Never split records on U+2028/U+2029.
  (Node's `readline` is non-compliant; Go's `bufio.Scanner` is fine.)
- Extension dialogs (`select`, `confirm`, `input`) surface as requests the
  client must answer — the UI maps them to modals, or auto-approves
  per user policy.

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

### Security model (binding contract until an ADR supersedes it)
- Binds `127.0.0.1` by default; listening on other interfaces requires an
  explicit flag **and** an auth token.
- Token auth for every API/WS route when bound beyond localhost.
- PiCode executes with the user's permissions — same trust boundary as Pi
  itself. Never adds remote execution surface silently.

## Explicit non-goals

- Re-implementing the Pi TUI (we embed it instead).
- Sandboxing agents (that's Pi's trust model, not ours).
- Hiding Pi: every GUI action maps to something inspectable in the terminal.
