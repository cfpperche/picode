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
via `TIOCSWINSZ` on the attach PTY.

A project shell's child is `$SHELL` when it names an executable file, else
`/bin/bash`, else `/bin/sh` — the daemon often has no `SHELL` at all
(systemd, a container, a test binary), and `/bin/sh` is dash on
Debian-family systems, where the intercept rcfile is not a valid argument:
`ensureShell` passes `--rcfile` only when the shell resolves to bash
(measured 2026-09-20 — dash answered `Illegal option --`, the pane died
before its first prompt, a server with nothing else on it followed under
`exit-empty`, and `new-session` still answered 0 while the API reported a
live terminal). Creation now verifies the session is alive before it
answers: a shell that exits at once is a 500 naming the shell, not a
terminal that never lived. Requires tmux ≥ 3.5; **3.7 or newer is
required to overlay anything on the panes** (floating panes; 3.8 adds modal
ones), and what matters is the *server's* version — `tmux -V` reports the
client binary, and after an upgrade the old server keeps interpreting the
panes until it exits (`PROTOCOL_VERSION` is stable across 3.6-3.8), so
`Manager.VersionInUse` reads `#{version}` and `/api/system` reports that
(ADR-0164). `HasSession` and `NewSession` retry the client's
"server exited unexpectedly" — the message a client gets when it lost the race to start the first tmux
server (two terminals created together on a machine with none running); the command never ran, so the
retry is safe. Attach and `NewSession` set `extended-keys on` /
`extended-keys-format xterm` (modifyOtherKeys). Probed live on 3.6 and
3.7c: both answer only DA1 to a pane's Kitty query, so pi falls back to
modifyOtherKeys and expects `ESC [27;2;13~`; tmux re-encodes client keys
per this format, so Shift+Enter reaches the TUI as a newline. The OSS
xterm.js (6.0.0) has neither Kitty nor modifyOtherKeys input encoding,
so `termKeys.js` encodes modified Enter itself (VS Code's terminal gets
the same result from its xterm fork). `/api/system` warns if the running
server is on another format.

Terminal CLI state (ADR-0056) and presence (ADR-0062) are live projections
recovered from validated native observations (ADR-0112):
scoped wrappers inject Claude and Codex hooks, native Grok hooks and Hermes
plugins (ADR-0107), OpenCode (`OPENCODE_CONFIG`
session plugin, no data-dir overlay and no write to `~/.config/opencode`),
or manual Pi TUI hooks; a wrapper
lease becomes `terminal.runtime`, while lifecycle reports become
`terminal.state` feed events. The lease carries a canonical CLI, run id, PID,
and process start token when available. The server watcher removes a lease
when its pane or process disappears and may recover older sessions from an
exact `pane_current_command` plus `pane_pid` match, never from pixels. Codex
hooks are invocation-only `-c` values whose exact SHA-256
fingerprints are trusted in the same session flags — PiCode never bypasses
trust and never writes `~/.codex`. On `resume`/`fork`, overrides follow the
subcommand arguments because the native subcommand parser discards root overrides. For agent invocations, the Pi wrapper
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

Grok and Hermes preserve their native homes and executable entrypoints. Grok
loads PiCode's receipted `hooks/picode-native.json`; Hermes loads a native
`picode-native` plugin, enabled through its own CLI without tool overrides. Its
selected/sticky profile is resolved by `hermes config path`. Unrelated files are
preserved and edited integration files refuse replacement. Integration code is
inert outside a PiCode terminal. Maintenance commands bypass installation.

| Hermes native plugin event | Lifecycle action |
|---|---|
| `pre_llm_call`, `post_approval_response` | publish `working` |
| `on_session_start`, `on_session_end`, `on_session_reset` | publish `idle` |
| `pre_approval_request` | publish `needs-you` |
| Delegated child ID or non-CLI platform | no report |

Grok publishes `working` at `UserPromptSubmit` and at every `PostToolUse`
(`PostToolUseFailure` when a tool failed to dispatch), and `idle` at
`SessionStart` and at the final `idle_prompt` notification. Its `Stop` hook can
request continuation; queued turn-end reports can arrive after another prompt
with a later dispatch timestamp. They are therefore ignored, along with child
events and `SessionEnd`; wrapper exit removes presence. This conservative
policy can delay idle by about a minute.

Grok runs a turn's tool calls as a parallel batch, so a question card can
wait while sibling tools finish. Its `ask_user_question` card notifies
`elicitation_dialog` and nothing else fires until the card is answered or
dismissed, so the report is a *held* attention: `needs-you` with
`attention: question`. Only the question tool's own completion (reported as
`attention: answered`) or a settled lifecycle report releases the hold — a
sibling's `PostToolUse` never does. The hold is stamped when the card
appeared, which can be older than a sibling completion in the same batch, so
it is exempt from the native ordering fence and cannot be silently dropped.

A permission prompt is `needs-you` everywhere, and none of the hook-driven
CLIs emits a "permission resolved" event. The approved tool's completion is
the resume signal that returns the terminal to `working`, so every CLI that
can wait on a permission registers its tool lifecycle. A permission hold has
no release identity, so in a parallel batch a sibling's `PostToolUse` resumes
the row while the prompt still waits; only a question hold names its own
release:

| CLI | `needs-you` | resume to `working` |
|---|---|---|
| Grok | `Notification` `permission_prompt`; `Notification` `elicitation_dialog` (held) | `PostToolUse`, `PostToolUseFailure`; a held question ends only at `ask_user_question`'s own completion |
| Claude Code | `Notification` `permission_prompt` / `agent_needs_input` | `PostToolUse`, `PostToolUseFailure` |
| Codex | `PermissionRequest` | `PostToolUse` |
| OpenCode | `permission.asked`, `question.asked` | `permission.replied`, `question.replied` |
| Hermes | `pre_approval_request` | `post_approval_response` |
| Pi TUI | `ui_prompt_start` | `ui_prompt_end` |
| Omp | `tool_approval_requested`; `tool_execution_start` with `toolName: "ask"` | `tool_approval_resolved`; `tool_result` |

The map also accepts `PreToolUse` but no CLI registers it: it fires before the
permission gate, so it reports work that may still be waiting for the human. A
long approved tool therefore stays `needs-you` until it finishes — no hook sees
the answer itself. Grok's other notifications (`task_complete`, …) carry no
attention meaning and do not change state. Claude's `TaskCompleted` and
`SubagentStop` also cannot mark the parent idle.

Native reports carry session identity, sequence and wrapper PID. The server
publishes identity and activity together and refuses stale incarnations. A
matching current pane/process can recover a wrapper after server restart;
another wrapper cannot be displaced. Communication never infers a new address
from the most recent session file (see [messages](direct-session-communication.md)).

OpenCode does not get an `XDG_DATA_HOME` overlay: `opencode.db` is SQLite
with WAL files beside the data path, and auth lives in the same directory.
The session wrapper sets `OPENCODE_CONFIG` to a PiCode-owned json that
adds one local plugin; configs merge, so the user's
`~/.config/opencode/opencode.jsonc` is not written. The TUI path prints
`Starting OpenCode...` before exec — OpenCode draws nothing until the
first frame. Maintenance subcommands (`session`, `auth`, `run`, …) skip
the plugin and the banner. The map is
executable in `TestOpencodeActivityMap`. Native `chat.message` selects a verified
root session (`client.session.get` without `parentID`); an exact connected resume
can seed that selection. Other roots and child events cannot change its identity
or activity. Reports are serialized before launching hook processes:


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

Codex modern hook reports outrank legacy notify within the same wrapper run,
including older wrappers that injected both paths. New launchers use one path;
legacy mode clears inherited modern-hook flags. SessionStart adds native message
command context (ADR-0111). OpenCode's resume metadata/status lookup runs after
plugin initialization, without blocking the instance bootstrap; native activity
invalidates a pending startup snapshot.

### tmux guard (ADR-0138)

The same session PATH (ADR-0056) carries a `tmux` wrapper that is a policy
gate rather than a launcher. It exists because three measured incidents had
one shape: an agent inside a managed terminal ran a tmux command whose blast
radius was the whole server (a prefix sweep killed 29 sessions on
2026-09-06; `kill-server` took every session, twice, on 2026-09-14/15 — the
second time from a "scratch" server the agent believed TMUX_TMPDIR had
isolated). **`$TMUX` outranks `TMUX_TMPDIR`**: a client started inside a
tmux session talks to the server named in `$TMUX` no matter what
`TMUX_TMPDIR` says, so "isolated" scratch work runs against production. The
guard is on by default (`enabled.json` key `tmux-guard`; only an explicit
opt-out disables it). The switch lives on **Agent CLIs**, above the CLI
catalog, on desktop and mobile — not on Terminal defaults. That page is
font, color and tmux options a terminal inherits; the guard is a PATH
wrapper for every terminal opened from then on, and it is not a setting of
the CLI selected under it. The wrapper resolves the real binary with pure
shell (`${0%/*}`) — a guard that shells out to `dirname` under a minimal
PATH found itself in its own bin dir and exec'd in an endless loop, caught
by its own test suite on 2026-09-15.

| Command | Condition | Action |
|---|---|---|
| `kill-server`, `kill-window`, `kill-pane` | any | refuse, actionable copy |
| `kill-session` | `-a` anywhere in a flag cluster | refuse |
| `kill-session -t X` | X is a pattern | refuse |
| `kill-session -t X` | X's session environment carries this terminal's `PICODE_TERM_ID` | allow |
| `kill-session -t X` | another terminal's marker, or no marker (the user's own) | refuse |
| `send-keys` | payload contains `kill-server`/`pkill`/`killall` | refuse |
| `new-session` (simple form) | no `PICODE_TERM_ID=` argument | add `-e PICODE_TERM_ID=<this terminal>` |
| anything else | — | passthrough, unmodified |

Outside a managed terminal the wrapper is not on PATH; invoked directly it
passes through. Each refusal prints the reason on the pane and appends one
line to `<dataDir>/tmux-guard.log`. The ownership probe reads the target's
session environment (`show-environment -t X PICODE_TERM_ID`) — the marker is
the receipt, the same rule the server-wide read model enforces. The guard is
a guardrail, not a security boundary: `/usr/bin/tmux kill-server` typed
verbatim still bypasses it. Probes carry no `-L`/`-S` on purpose: inside a
pane they inherit `$TMUX`, which names that pane's own server — the
dedicated socket since ADR-0139 — and from outside a pane they ask the
default server, where an exact-name kill for a session on another socket
finds no marker and is refused (fail-closed).

### Browser hand-off (ADR-0180)

The same session PATH also carries a browser hand-off: a `picode-open`
wrapper, `xdg-open` and `wslview` shadows, and `BROWSER=<dataDir>/bin/picode-open`
in the session environment. A CLI's "open in the browser" moment (OAuth
`/login` above all) used to resolve inside WSL — an installed chromium
opened in a window nobody watches. The wrapper forwards exactly its first
`http(s)` argument to `POST /api/terminals/{id}/open-url` (bearer token
from `<dataDir>/token`, the hook reporters' auth); anything else — other
arguments, missing curl, a failed POST — falls through to the real opener
it shadows. Outside managed terminals the wrappers are not on PATH.

The endpoint routes by audience:

| Condition | Action |
|---|---|
| feed has a subscriber | ephemeral `terminal.open_url` → the visible client opens it |
| no subscriber | `osopen.OpenURL` — the Windows default browser via PowerShell `Start-Process` (the URL rides the `PICODE_OPEN_URL` environment variable, never a command line) |
| refused URL (non-http(s), control characters, > 2048 bytes) | 400, nothing opened |

A visible client answers the event through the clicked-link path, so the
link-destination preference governs: the desktop app opens its integrated
tab, a plain browser opens a tab in itself. The event is ephemeral — a feed
reconnect never replays a login page. The wiring row (`open-url`, Agent
CLIs page) defaults on like the guard; an explicit opt-out removes the
wrappers and the `BROWSER` entry.

### Dedicated socket and the drain (ADR-0139)

Every instance's sessions live on their own server: the daemon runs tmux
with `-S <dataDir>/tmux.sock` (production: `~/.picode/tmux.sock`; scratch
and fixture instances get theirs under their data dir, so two instances can
never share a server — the trap AGENTS.md records about scratch sessions
landing in the owner's tmux). `$TMUX` inside a pane names that server, so
in-pane tmux commands, the guard included, work unchanged.

tmux has no live migration, so pre-move sessions are **drained**: the
daemon's Manager holds a second Manager on the default socket, every
session-scoped operation falls back to it when the primary does not have
the session, and the server-wide reads (`ServerSessions`, `ListSessions`,
`ListOwned`, `ServerInfo`) merge both sides so the fleet stays one list.
The bridge asks `SocketFor(session)` which socket to attach with. The
drain disappears with the last legacy session; nothing is restarted, and
the rollback is the previous binary.

### Restart recovery (ADR-0112)

The common native hook writes a private, ordered observation before HTTP, so
reports survive daemon downtime. The existing presence watcher validates its OS
boot ID, process start token, wrapper run, CLI and exact pane before rebuilding
live identity/activity. The original Working age is retained. Missing or invalid
records stay unknown; last-session pins never prove an active conversation.
The private lock retains the ordering/source fence before checkpoint publication.
Both must agree; a failed newer write cannot expose an older Idle. A failure to
retain the fence blocks that incarnation until a fresh wrapper start.
Attention checks that the disk observation still agrees before the existing
native input guards. Pi receiver presence renews within five seconds, independently
of activity. This recovery currently requires Linux/WSL process metadata.

Messages shows activity and connection separately. Pending identification uses
Syncing / Reconnecting, native approvals use Needs your input, and only preparation
errors use Connection failed. Historical Test passed is separate from readiness.
Enabled unavailable owners remain visible in the test selectors; selecting them
never redirects a test to another participant.

Hermes activity follows an explicit root CLI/TUI session/turn, including approval
and completion correlation. Unscoped starts and background helper turns do not
prove Idle. Its explicit CLI reset boundary changes identity; another root event
selects the next tool context. Sequences are assigned before subprocess I/O so
a slow completion cannot overwrite a newer turn.
