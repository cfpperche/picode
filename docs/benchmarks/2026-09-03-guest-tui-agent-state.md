# Study: terminal TUI agent state (spinner / "needs you" for coding CLIs)

- **Date:** 2026-09-03
- **Sources:** fetched live 2026-09-03 —
  [agentclientprotocol.com](https://agentclientprotocol.com) (intro,
  get-started/agents, get-started/clients, registry),
  [code.claude.com/docs/en/hooks](https://code.claude.com/docs/en/hooks),
  …/statusline, …/terminal-config,
  [learn.chatgpt.com/docs/config-file/config-reference.md](https://learn.chatgpt.com/docs/config-file/config-reference.md),
  …/app-server.md, …/non-interactive-mode,
  [opencode.ai/docs/server](https://opencode.ai/docs/server/),
  [opencode.ai/docs/acp](https://opencode.ai/docs/acp/),
  [github.com/superagent-ai/grok-cli](https://github.com/superagent-ai/grok-cli),
  [github.com/BloopAI/vibe-kanban](https://github.com/BloopAI/vibe-kanban)
  (+ docs/settings/agent-configurations),
  [github.com/stravu/crystal](https://github.com/stravu/crystal),
  [conductor.build](https://conductor.build), and the installed Pi 0.84.4
  `docs/extensions.md` lifecycle contract. Repo studies reused:
  [t3code / paseo](2026-08-24-adopt-t3code-paseo-cursor.md),
  [herdr](2026-08-27-herdr.md).
- **Scope:** the owner runs Claude Code, Codex, Grok CLI, Antigravity,
  pi and opencode **inside PiCode terminals** and wants the sidebar to
  show *working* (spinner) and *needs you* for them. This study maps
  how the market gets that state and what PiCode should adopt. Output:
  **ADR-0056 (proposed)** — this study does not by itself change
  ADR-0003 or the "not a multi-runtime harness" adaptation rule; that
  re-measure is the owner's call.

## The question, sharpened

"Use the TUIs inside PiCode" needs two things today's terminal gives
only one of: **running** the TUI (yes — tmux surfaces, ADR-0017/0026)
and **observing its lifecycle** (no). A TUI in a PTY is pixels;
spinner/needs-you need a *channel*. The benchmark: no credible
orchestrator reads pixels as its primary state source.

## Four integration levels (who does what)

| Level | State channel | Who uses it | Verdict for PiCode |
|---|---|---|---|
| 1. Terminal only | pixels | PiCode today (guest CLIs) | status quo; the only honest state is "unknown" |
| 2. TUI + side channel | hooks / notify / bell emitted by the CLI while its TUI stays on screen | paseo (PTY + provider hooks, 2026-08-24 study), Conductor, Crystal | **the tier to build** — TUI stays, state arrives as events |
| 3. Screen detection | heuristics over the terminal buffer | herdr (`AgentState {Idle, Working, Blocked}` read from the bottom buffer, 2026-08-27 study) | last-resort fallback only; benchmarks.md demands truthful status |
| 4. Protocol (no TUI) | JSON-RPC with the agent; the UI is entirely ours | t3code (SDKs + ACP), Zed, Casper (web ACP client), Vibe Kanban (headless executors) | highest ceiling; separate future ADR |

## Receipts

| Source | Fact (2026-09-03) |
|---|---|
| agentclientprotocol.com | ACP "standardizes communication between code editors/IDEs and coding agents"; local agents run as editor subprocesses over JSON-RPC on stdio |
| …/get-started/agents | Listed agents: Claude Agent (via Zed's SDK adapter), Codex CLI (via Zed's adapter), Cursor, Factory Droid, Gemini CLI, Kimi CLI, OpenCode, **Pi (via pi-acp adapter)**, Qwen Code |
| …/get-started/registry | "pi ACP — ACP adapter for pi coding agent" |
| …/get-started/clients | Zed, JetBrains, Neovim, Qt Creator, Unity, Toad, Casper — "a web client for kiro-cli that talks to it over ACP … live streaming, session history, per-tool-call rendering" |
| github.com/agentclientprotocol/claude-agent-acp | "ACP adapter for the Claude Agent SDK" |
| github.com/agentclientprotocol/codex-acp | Codex adapter "built on the new Codex App Server"; "The latest version of Zed can already use this adapter out of the box" |
| code.claude.com/docs/en/hooks | Hooks are "user-defined shell commands, **HTTP endpoints**, MCP tool calls, LLM prompts, or subagents" fired at lifecycle points; events include `PermissionRequest`, `Notification`, `Stop`, `SubagentStop`, `TaskCompleted`; `Notification` types include `permission_prompt`, `idle_prompt`, `auth_success`, `agent_needs_input`, `agent_completed` |
| code.claude.com/docs/en/statusline | the statusline script "receives JSON session data on stdin" — a per-keystroke liveness side channel |
| code.claude.com/docs/en/terminal-config | "When Claude finishes a task or pauses for a permission prompt … it fires a notification event"; `preferredNotifChannel: "terminal_bell"`; tmux passthrough documented |
| learn.chatgpt.com config-reference | `notify` = "Command invoked for notifications; receives a JSON payload from Codex"; `tui.notifications` / `tui.notification_method` / `tui.notification_condition` exist but are TUI-scoped |
| learn.chatgpt.com app-server | "the interface Codex uses to power rich clients (for example, the Codex VS Code extension)"; JSON-RPC 2.0 over stdio JSONL or WebSocket; threads, approvals, streamed events; open source at `openai/codex/codex-rs/app-server` |
| learn.chatgpt.com non-interactive-mode | `codex exec` for scripts/CI; progress streams to stderr, final message to stdout; "make output machine-readable" section |
| opencode.ai/docs/server | `opencode serve` = headless HTTP server with an OpenAPI 3.1 spec endpoint (SDK generated from it); basic auth via `OPENCODE_SERVER_PASSWORD`; "the TUI is the client that talks to the server" |
| opencode.ai/docs/acp | `opencode acp` — "use OpenCode in any ACP-compatible editor" (JSON-RPC via stdio) |
| github.com/superagent-ai/grok-cli | `grok -p --format json` emits "a newline-delimited JSON event stream … `step_start`, `text`, `tool_use`, `step_finish`, `error`" |
| github.com/BloopAI/vibe-kanban | "Switch between 10+ coding agents — Claude Code, Codex, Gemini CLI, GitHub Copilot, Amp, Cursor, OpenCode, Droid, CCR, and Qwen Code"; docs: per-agent executor profiles with launch env |
| github.com/stravu/crystal | "(Crystal is now Nimbalyst) Run multiple Codex and Claude Code AI sessions in parallel git worktrees" — desktop status/notifications over the CLIs |
| conductor.build | "Conductor runs the first-party Claude Code, Codex, Cursor, and OpenCode agents under the hood" |
| Pi 0.84.4 extension docs | Native events include `agent_start`, blocking `ui_prompt_start` / `ui_prompt_end`, and `agent_settled` after retry, compaction, and follow-up work; `ctx.mode` distinguishes TUI from RPC/print/JSON |

## Per-TUI channel catalog (what PiCode can read today)

| TUI | working | needs you | deep protocol |
|---|---|---|---|
| Claude Code | `Stop` / `TaskCompleted` hooks; statusline JSON; headless `stream-json` | `Notification` hook (`permission_prompt`, `agent_needs_input`, `idle_prompt`, `agent_completed`…) — **can POST straight to the daemon over HTTP** | ACP adapter (official) |
| Codex | `notify` JSON payload; `codex exec` machine-readable output | same `notify` (turn end / approval) | **app-server** (the official embed protocol); ACP adapter on top of it |
| opencode | server HTTP/SSE events | permission events via the API | `opencode acp` + `opencode serve` |
| Gemini CLI | — | — | ACP native |
| Kimi / Qwen / Droid / Cursor | — | — | ACP listed |
| Grok CLI (superagent-ai) | NDJSON events from `grok -p --format json` | partial (event stream, no permission semantics) | none public |
| Antigravity | — | — | none public (Google IDE) → stays level 1 |
| pi | native extension `agent_start` / `agent_settled` in a manual TUI; RPC for managed agents | native extension `ui_prompt_start` / `ui_prompt_end`; RPC `waiting` for managed agents | **pi-acp adapter listed in the official registry** |

Universal fallback for any TUI inside our PTYs: the **terminal bell**
(Claude Code rings it when finished/prompting while the user is away;
we own the PTY, so BEL is catchable). herdr-style screen detection is
the last resort after that.

## What PiCode should take (→ ADR-0056, proposed)

1. Guest CLIs and manually launched Pi keep their **real TUIs in
   terminals** (ADR-0017/0026 substrate). State comes from **level-2
   sensors**, not scraping: a
   per-tool hook/notify config POSTs lifecycle events to the daemon.
2. The daemon republishes sensor events on the **ADR-0048 feed** using
   the same state vocabulary pi already emits (`working` / `needs you`
   / `idle`, honest `unknown` when no sensor reports). Sidebar spinner
   and "needs you" then work for guests with zero new UI concepts.
3. Sensors are **config-side**: PiCode offers/writes hook snippets for
   the user's installed CLIs. **No agent SDK is embedded** — ADR-0003's
   letter holds; guests stay user-installed.
4. ACP (level 4) is the named future track — observing is tier 2,
   *controlling* (answering permission prompts, driving turns) is what
   ACP/app-server add. That track would also re-measure ADR-0003's
   "Pi-only agents" clause and the benchmarks README's "we are not a
   multi-runtime harness" line. Own ADR, owner's call.

## Refuse (for now)

| Temptation | Why not |
|---|---|
| Screen-scraping as the primary state source | inference from pixels breaks on every TUI release and produces false states; herdr only does it because it has no channel |
| Per-TUI package managers / installers for guests | owner's stated non-goal; guests stay user-installed CLIs |
| Replacing the guest TUI with our chat (level 4 now) | bigger bet than the stated need; the terminal is the product's "door, not cage"; needs the ADR-0003 re-measure first |

## Open gaps (honest)

- Codex `notify` payload event-type names were **not** re-verified on
  2026-09-03 (docs confirm the key + "receives a JSON payload"; the
  classic `agent-turn-complete` type string did not surface in the
  fetched page).
- Vibe Kanban's internal status mechanism (process exit + logs vs
  structured events) not fully mapped — cited as a product benchmark,
  not a protocol source.
- Antigravity: no public integration surface found; stays level 1
  (terminal) until Google publishes one.
- Casper drives kiro-cli, not the CLIs in the owner's rotation — cited
  as proof that a browser client can own sidebar states over ACP.
