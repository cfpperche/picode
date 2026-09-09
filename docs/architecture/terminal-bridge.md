# TerminalBridge

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

One tmux session per interactive agent (`internal/tmux`: create/kill/list,
exact-name matching via `=` prefix, PiCode-owned `picode-` name namespace,
ids sanitized — dots/colons are tmux target separators and corrupt
lookups). Project shells use `picode-sh-<id>` (same prefix, different name).
`internal/term` bridges WebSocket ↔ PTY (`tmux attach`):
binary frames = terminal bytes, text frames = `resize` control JSON;
closing the tab ends only the attach — the agent or shell keeps running in tmux.
Terminal output and keepalive ping frames are serialized per connection because
Gorilla WebSocket allows only one concurrent writer.
On the browser side `web/shared/client/termSocket.js` owns the attach
lifecycle: a dropped WebSocket (phone lock suspends the page, network
hand-off) reattaches automatically — exponential backoff 1 s → 10 s,
up to 12 consecutive failures per burst, then one honest
"Session ended. Reopen the terminal." line; regaining visibility or
connectivity (or remounting the pane) kicks an immediate attempt and
restarts an exhausted burst. Reattach reuses the same xterm instance and
resends the pane size, so the reader's place survives; tmux keeps the
copy-mode scroll position across attaches, and the tmux session itself
never depends on the socket — closing the tab ends only the attach.
`RespawnPaneEnv` changes a pane's child without changing its immutable tmux
session id (used by restart-same-mode and the explicit dead-pane recovery).
`PasteText` types text into a pane as a bracketed paste plus Enter — the
ADR-0060 reply fallback. `PaneCommand` and `PaneSessionID` verify transitions. Resize propagates
via `TIOCSWINSZ` on the attach PTY. Requires tmux ≥ 3.5. `HasSession` and `NewSession` retry the client's
"server exited unexpectedly" — the message a client gets when it lost the race to start the first tmux
server (two terminals created together on a machine with none running); the command never ran, so the
retry is safe. Attach and `NewSession` set `extended-keys on` /
`extended-keys-format xterm` (modifyOtherKeys). Probed live: tmux 3.6
answers only DA1 to a pane's Kitty query, so pi falls back to
modifyOtherKeys and expects `ESC [27;2;13~`; tmux re-encodes client keys
per this format, so Shift+Enter reaches the TUI as a newline. The OSS
xterm.js (6.0.0) has neither Kitty nor modifyOtherKeys input encoding,
so `termKeys.js` encodes modified Enter itself (VS Code's terminal gets
the same result from its xterm fork). `/api/system` warns if the running
server is on another format.

Terminal CLI state (ADR-0056) and presence (ADR-0062) are ephemeral:
scoped wrappers inject Claude, Codex, Grok, Hermes Agent (PYTHONPATH
sitecustomize, no `HERMES_HOME` overlay), OpenCode (`OPENCODE_CONFIG`
session plugin, no data-dir overlay and no write to `~/.config/opencode`),
or manual Pi TUI hooks; a wrapper
lease becomes `terminal.runtime`, while lifecycle reports become
`terminal.state` feed events. The lease carries a canonical CLI, run id, PID,
and process start token when available. The server watcher removes a lease
when its pane or process disappears and may recover older sessions from an
exact `pane_current_command` plus `pane_pid` match, never from pixels. Codex
hooks are invocation-only `-c` values whose exact SHA-256
fingerprints are trusted in the same session flags — PiCode never bypasses
trust and never writes `~/.codex`. For agent invocations, the Pi wrapper
prepends one generated `-e <data>/intercept/pi-terminal-state.ts` extension
without replacing user extensions or arguments. Pi's maintenance/auth
subcommands and help/version flags bypass injection so their argv[1] dispatch
stays intact; the wrapper never writes `~/.pi`. Its native-event decision table
is executable in `TestPiTerminalStateExtensionDecisionTable`:

| `PICODE_TERM_ID` | Pi mode | Native event / condition | Lifecycle action |
|---|---|---|---|
| missing | any | any | no report |
| present | RPC, print, or JSON | any | no report |
| present | TUI | `session_start`, `agent_settled`, `session_shutdown` | publish `idle` |
| present | TUI | `agent_start` | publish `working` |
| present | TUI | `ui_prompt_start` | publish `needs-you` |
| present | TUI | `ui_prompt_end`, `ctx.isIdle() == false` | publish `working` |
| present | TUI | `ui_prompt_end`, `ctx.isIdle() == true` | publish `idle` |

The two guards keep managed RPC agents, `pi -p`, JSON streams, and inherited
sub-processes out of terminal state.

Hermes Agent does not get a `HERMES_HOME` overlay: its `state.db` is SQLite
with WAL files beside the home path, so relocating the home would split the
user's sessions. The session wrapper unwraps the official trampoline (which
`unset`s `PYTHONPATH`), sets a session `PYTHONPATH` sitecustomize, and execs
`--accept-hooks`. sitecustomize patches `agent.shell_hooks.register_from_config`
with a deepcopy so `load_config` / `save_config` never see PiCode's entries
and `~/.hermes/config.yaml` is not written. Hermes itself may still record
the hook command in `shell-hooks-allowlist.json` when auto-accepting.
Maintenance subcommands (`setup`, `model`, `auth`, …), including after
`-p`/`--profile`, skip the patch. `pre_tool_call` is not registered (it is a
gate, not a status signal). The map is executable in `TestHookMapPy`:

| Hermes shell-hook event | Lifecycle action |
|---|---|
| `pre_llm_call`, `post_approval_response` | publish `working` |
| `on_session_start`, `on_session_end`, `on_session_reset`, `on_session_finalize`, `post_llm_call`, `subagent_stop` | publish `idle` |
| `pre_approval_request` | publish `needs-you` |
| `pre_tool_call` (not injected) | no report |

OpenCode does not get an `XDG_DATA_HOME` overlay: `opencode.db` is SQLite
with WAL files beside the data path, and auth lives in the same directory.
The session wrapper sets `OPENCODE_CONFIG` to a PiCode-owned json that
adds one local plugin; configs merge, so the user's
`~/.config/opencode/opencode.jsonc` is not written. The TUI path prints
`Starting OpenCode...` before exec — OpenCode draws nothing until the
first frame. Maintenance subcommands (`session`, `auth`, `run`, …) skip
the plugin and the banner. The map is
executable in `TestOpencodeActivityMap`:

| OpenCode plugin event | Lifecycle action |
|---|---|
| `session.status` busy / retry | publish `working` |
| `session.status` idle, `session.idle` | publish `idle` |
| `permission.asked`, `permission.v2.asked`, `question.asked`, `question.v2.asked` | publish `needs-you` |
| `permission.replied`, `permission.v2.replied`, `question.replied`, `question.v2.replied` | publish `working` |
| `question.rejected`, `question.v2.rejected` | publish `idle` |
| `tool.execute.before` (not mapped) | no report |

PTY input closes the gap left by a CLI that omits its interruption callback:

| Input | Current terminal state | Session | Lifecycle action |
|---|---|---|---|
| Ctrl+C or one bare Escape frame | `working` / `needs-you` | matching project shell | publish `idle`, then forward input |
| Ctrl+C or bare Escape | `idle` / no signal | matching project shell | no event; forward input |
| Ctrl+C or bare Escape | any | agent or another shell | no event; forward input |
| Arrow, Alt, or function-key escape sequence | any | any | no event; forward input |
