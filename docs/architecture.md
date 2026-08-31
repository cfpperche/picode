# Architecture

> Status: v0.1 — evolves with the project. Last reviewed: 2026-08-31.
> Changing anything described here requires updating this file (see [AGENTS.md](/AGENTS.md)).

## The one-paragraph version

PiCode is a **single Go binary that serves a browser UI** and manages **real
`pi` processes** on the machine where it runs. Each agent gets a dual channel:
a tmux-backed PTY that renders the genuine Pi TUI inside a browser terminal,
and an RPC bridge (`pi --mode rpc`, JSONL over stdio) that feeds the rich UI
with structured events. A broker routes messages between agents through a Pi
extension, so agents talk to each other using Pi's own tool-calling protocol.

`picode install` (ADR-0018) enables a systemd **user** unit so it starts with
this Linux session (WSL included). `picode deploy` / `make deploy` copies a
repo build and restarts that unit. `picode update` checks GitHub for a newer
release.

`picode provision` (ADR-0020) converges a machine on all of that at once:
`[boot] systemd=true` in `/etc/wsl.conf`, lingering so the unit starts
without a login, a valid certificate, the unit itself, and `/api/health` as
proof. Every step is check → fix → verify, and a step that finds nothing to
do changes nothing at all — `/etc/wsl.conf` is merged line by line after a
backup, so a file that already satisfies the check comes back byte for byte
identical. `--dry-run` reports the plan; `--json` feeds it to a caller. Root
work (`wsl.conf`, linger) and user work (unit, cert) are separate scopes, so
the two can be applied by different runs. On WSL that caller is PiCode
Desktop, the Windows tray binary that owns the logon task.

`picode-desktop` (`cmd/picode-desktop`, `internal/desktop`) is that binary: a
single `.exe` cross-compiled from WSL with `CGO_ENABLED=0` (`make desktop`).
`doctor` reports, `install` applies and registers the logon task, and the
default mode sits in the notification area. It drives the distro through
**two** `picode provision --json` calls — `-u root` for `wsl.conf` and
lingering, then `-u <owner>` for the unit, certificate and data dir, because
installing those as root would put PiCode in `/root`. The merged view keeps
whichever pass resolved each step. Windows-side it owns only what cannot live
in the distro: trusting the mkcert CA (already elevated at install, so no UAC
dance), the `onlogon` scheduled task (`/rl limited` — an elevated tray cannot
reach Explorer's notification area), and a `sleep infinity` child that stops
WSL's idle timeout from reclaiming the VM. It learns the address from
`server.json` once, then polls `/api/health` over HTTP rather than spawning
`wsl.exe` on a timer. `wsl.exe` answers in UTF-16LE **without a BOM**, so its
output is decoded by inspecting the bytes.

On a machine without WSL, `install` first walks a stage machine derived from
what it observes — never from saved progress, so an interrupted run resumes
and a finished one is a no-op: install WSL (`--no-distribution`), restart,
install the distro (`--no-launch`), create the account, then provision. Both
flags exist to dodge the interactive account setup that plain `wsl --install`
opens. Exit code **3010** means the step succeeded and Windows wants a
restart, not that it failed; a `RunOnce` key resumes setup at the next logon
and deletes itself. The Linux account is named after the Windows one and its
password is left **locked** — provisioning reaches root through `wsl -u root`
and never needs sudo, so setting a password or granting passwordless sudo
would be a security decision the installer has no standing to make.

Both binaries ship in one GitHub release, tag-triggered
(`.github/workflows/release.yml`), with the version stamped through
`-X internal/version.Version` — `Version` is a var for exactly that. Asset
names are a contract: `internal/install.assetName()` looks for
`picode-<goos>-<goarch>`, and `picode-desktop` for
`picode-desktop-windows-amd64.exe`; a test reads the workflow to keep the two
from drifting. `picode update` and `picode-desktop update` both use
`install.LatestReleaseFor`. Elevation is decided at **runtime** (`ShellExecuteW`
with `runas`), never by a manifest: the same executable is also the tray, and a
`requireAdministrator` manifest would elevate that too.

## Application routes

The SPA has **two shells** in one Vite app (`web/src/desktop`, `web/src/mobile`),
picked at boot by viewport (`max-width: 767px`) or `?desktop=1` / `?mobile=1`.
Shared code lives in `web/src/lib`. Rotate does not remount (would drop WS).
The mobile shell is a PWA (`manifest.webmanifest`, `sw.js`, Apple
`apple-mobile-web-app-capable`) so Add to Home Screen opens full screen.
If `GET /api/health` fails, the shell shows a Reconnecting overlay and
reloads when the server is back. Fast restarts (shorter than the 2.5s
poll) are caught by comparing the health `bootId` — a new id reloads the
tab; unexpected WebSocket closes kick an immediate check.

Hash routes (ADR-0012). **Preferences** is PiCode-the-product.
**Settings** is pi JSON for the selected agent. Auth, MCP, packages
stay on their own routes.

| Hash | Surface | Owns |
|---|---|---|
| `#/` | Agent workspace | tabs, chat, terminal. Replaced by `#/agent/<id>` when an agent is open. |
| `#/agent/<id>` | Agent workspace | same shell; URL is the open agent (wins over saved tabs on load) |
| `#/file/t/<id>/<path>` | File tab | text editor for a path under that terminal's cwd (Ctrl+click in xterm). `#/file/a/<id>/<path>` is the same for the Pi TUI dock; `#/file/w/<id>/<path>` reads through a workspace (ADR-0030). Preview \| Raw for svg, mermaid, md, png, pdf, audio, video, glb/gltf (`GET …/blob`). |
| `#/tree/<w\|t\|a>/<id>` | File tree tab | read-only tree of the owner's folder (ADR-0030): lazy per-level browse, a **Changes** section from `…/gitstatus` on top, changed files and their folders dotted. Tab identity is the canonical root (`d:<root>`), so owners of one folder share a tab; a click opens the normal file tab. |
| `#/settings` | pi config | global + workspace + agent (composer `/settings`) + **Keys** (`keybindings.json`) |
| `#/preferences` | PiCode chrome | appearance, **terminal** (xterm look), notifications, server port, **backup** (ADR-0014); tabs `#/preferences/<section>` |
| `#/system` | Machine facts | host, network, deps, version (read-only) |
| `#/providers` | Pi providers | catalog + signed-in state; Sign in; **Usage** per vault account (ADR-0031) |
| `#/mcps` | Pi MCP | adapter manager: list / add / toggle / remove / **Use from…** (mirror host configs; Off hides a server). |
| `#/packages` | Pi packages | machine / workspace (`pi install`) / this agent (`-e` on start) (ADR-0010). Same agent context as MCP. A behind npm row shows **Update**; the user menu badges when any are. |
| `#/devices` | Connected browsers | host vs LAN/tailnet phones (presence ping) |

Composer `@` lists files in the agent cwd (`GET /api/agents/{id}/files`), plus other agents and skills (mentions in this prompt, not a message to that agent).
Composer `/` also lists **extension commands** from the running managed agent
(`GET /api/agents/{id}/slash` → RPC `get_commands`, ADR-0029). Picking one
sends `/name` as a prompt. Stopped agents omit that list. Names that collide
with a PiCode command are dropped.
Click a path on an `edit`/`write` card (or the turn's file names) opens a closable card in the thread. **Open in tab** is the same `#/file/a/<id>/<path>` as the terminal. Save writes the file in the tab. A stale mtime is 409 (open again). Keep/Undo on the diff card: Undo rewrites the old lines (or Open if the file moved).
The sidebar has five flat tabs, one kind each (ADR-0026, fifth added by ADR-0036), in order: **Workspaces** (the landing tab — one collapsible card per workspace holding its agents and its terminals; no section-level collapse), **Agents** (free agents, name-sorted, no hierarchy — agent and terminal cards share one flat row shape), **Terminals** (free terminals only), **Apps** (a grid of app tiles drawn from `GET /api/apps` manifests — numeric badge for actionable counts, dot for activity, aggregated onto the tab icon; a tile opens the app as a main tab `x:<id>` / `#/app/<id>`) and **Pins**. Nothing appears in two tabs. Terminals are first-class shells (ADR-0017): **+** on the Terminals tab creates a free one (`POST /api/terminals` → tmux `picode-sh-<id>` in `$HOME`); the terminal button on a workspace card creates one owned by it, born in the workspace folder (`workspaceId` in the POST body). Either opens on the main tab strip (`#/term/<id>`). Closing the tab detaches; Remove kills tmux; removing a workspace kills its terminals with it (the cleanup dialog warns with the count from the preview). Not tied to an agent. The agent's Pi TUI view renders through the **same TermSurface/ShellTerm component** as terminals (same xterm.js options, wheel, keys, links, envelope) — one engine, one look; managed mode shows a one-line hint with an Open TUI action instead. Ctrl/Cmd+click a path under the **live** pane cwd (`tmux #{pane_current_path}`, `GET /api/terminals/{id}/cwd`) opens `#/file/…` on the same strip (`GET/PUT /api/terminals/{id}/text`). `cd` then a relative path opens the file in the new folder. http(s) opens in the browser. Paths outside that live cwd are not links. Keys (Preferences → Terminal): Shift+drag select, Ctrl+C copy if selected, Ctrl+V paste. A gear after **+** opens the defaults every terminal inherits; a gear on a row opens that terminal's overrides (ADR-0024).
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
is not copied into SQLite (ADR-0005). The transcript endpoint serves a
window (`?tail=&skip=`) — the browser holds only the newest slice and
`Load earlier` pages older turns from the server. The window is cut at the
last compaction boundary (what pi itself replays): pre-compaction history
lives only inside the collapsible summary card, and the response reports
`compacted` so the UI can tell "needs /compact" from "already compacted,
file just stays large".
fetches older turns on demand. **From a Pi session** copies a JSONL
and creates a stopped agent (ADR-0021). The original TUI is not touched.

Entry: user menu (Settings, Preferences, Providers, MCPs) and `Ctrl+K`.
QR in the sidebar brand opens a phone-share drawer (`GET /api/share`):
HTTPS + bind + reachable IP + cert SAN + mkcert CA. Missing checks
list the action; a QR is only drawn when every check passes.

## Component diagram

```
┌────────────────────────────────────────────────────────────┐
│ Browser                                                     │
│  ├─ Rich UI (React + Vite + Tailwind — ADR-0008)           │
│  │   sidebar tabs (agents·workspaces·terminals·pins)      │
│  │   tasks · diffs · sessions tree · auth                 │
│  └─ xterm.js terminals (the real Pi TUI, 1:1)              │
└───────────────┬────────────────────────────────────────────┘
                │ HTTP /api/*  +  WebSocket /ws/*
┌───────────────▼────────────────────────────────────────────┐
│ picode (single Go binary, UI embedded — `-tags embedui`)   │
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
`terminals` (each owned by a workspace, `ws_free` for free ones — ADR-0026; no FK, cascade is app-driven),
`tasks` (prompt/steer/follow_up queue with a delivery state machine),
`messages` (reserved M4 broker inbox), `events` (orchestration audit),
`settings`. Embedded sequential migrations; the M1 JSON registry is imported
once and retired (`workspaces.json.migrated`).

### AgentManager (M1 core shipped)
Owns agent lifecycle via the SQLite store (`internal/store`): workspaces
workspaces start empty (ADR-0027) and own zero or more agents; tmux
session names derive from the agent id.
An agent runs as `pi` (ADR-0003, user-installed) in a named tmux session.
Per-agent provider/model/thinking is stored on `agents` and passed as
`pi --provider/--model/--thinking` on start (ADR-0009). Auth stays in
`~/.pi/agent/auth.json`; PiCode never collects keys.
`GET /api/providers/{id}/usage` (ADR-0031) reads the active slot.
`GET /api/providers/{id}/accounts/{aid}/usage` reads that vault row
without swapping `auth.json` (refresh writes the row; `auth.json` only
if it is active). Catalog `quotaKind` on each account tells `#/providers`
when to show Usage (`oauth` or `api_key`). Banked resets (Codex, Grok)
ride `resets[]`; `POST …/usage/reset` redeems one after the UI confirms.
Grok resets also try `~/.grok/auth.json` then `GROK_COOKIE`.

HTTP API (Go 1.22 method patterns):
- `GET/POST /api/workspaces` — list (with live `running` flag) / add.
  Add registers the folder only (ADR-0027): the 201 carries `agents: []`
  and no `agent` key; an idempotent re-add answers with the real agents
  and never resurrects a deleted one. `agent` is omitted whenever a
  workspace is empty.
- `POST /api/workspaces/clone` — `{url, name, path}` clones a remote
  repository into a fresh/empty destination and registers it as a
  workspace (ADR-0034). Blocking, 10-minute cap; host credentials, all
  interactive prompts disabled. A destination already cloned from the
  same origin is adopted (`200 {adopted:true}`); occupied by anything
  else → 409. The one git write reachable from the GUI.
- `GET /api/apps` — apps host (ADR-0036): manifests `{id, name, icon,
  apiVersion}` plus a live `badge` (`count` = actionable, `dot` =
  activity) per app; the poll target for the Apps tab. A badge failure
  degrades to no badge, never a failed list. First-party apps only,
  assembled in `cmd/picode` (`PICODE_DEMO_APP=1` adds a hidden QA app).
- `GET /api/apps/{id}/view?path=…` — one screen of an app as a tree of
  UI primitives (list / detail-markdown / form / actions) the SPA
  renders with host components; `apiVersion` gates rendering on both
  sides. `POST /api/apps/{id}/action` — `{action, path, args}` →
  `{toast?, view?, path?}`.
- `GET /api/workspaces/{id}/favicon` — the project's favicon (root, then
  public/static/app/src/app/www/docs; svg > png > ico), read-only and
  confined to the folder; the workspace card wears it.
- `DELETE /api/workspaces/{id}` — remove (stops **all** agents first, then
  kills the workspace's terminals — sessions best-effort, records and
  settings overrides in one transaction; ADR-0026). With
  `?files=1&confirm=<folder name>` it also deletes the project folder
  from disk (ADR-0035): the confirm must match the folder's basename,
  root/home are refused, and validation runs before anything is removed.
  Optional `?sessions=1` deletes the pi session dir when this workspace is
  the last occupant of that cwd. Project folders are never deleted. The
  cleanup preview (`GET /api/workspaces/{id}/cleanup`) counts the terminals
  so the dialog can warn.
- `GET /api/workspaces/{id}/cleanup` / `GET /api/agents/{id}/cleanup` —
  preview for the delete dialog (session count, last occupant, owned work folder).
- `DELETE /api/agents/{id}` — unregister. Optional `?sessions=1&work=1`
  (work only if cwd is under `~/.picode/work/` and nobody else uses it).
- `GET /api/workspaces/{id}/sessions/manage` — every Pi session under the
  folder (`session.ListDir`), each with size/age/messages/cost and
  `inUseBy` (the agent whose current session it is); `cleanupDays` and
  `totalBytes` ride along. `DELETE` on the same path removes one orphan
  (in-use → 409). Powers the `#/sessions/<id>` view (sidebar folder icon):
  Open with… reuses the resume endpoint, Compact reuses the agent compact.
- `GET/PUT /api/session-cleanup` — orphan auto-clean preference in days
  (0 = off, default). Sweep runs at boot, daily, and after each change;
  it deletes only sessions no agent is bound to.
- `GET/DELETE /api/sessions/all` — machine-wide view (`session.ListAll`):
  every Pi session on the machine, each tagged with the workspace owning
  its folder; delete validates against the sessions root. Powers the
  `#/sessions` All-folders view.
- `POST /api/workspaces/{id}/open|close` — start/stop the pi agent
  (idempotent); 409 on a workspace with no agents, like every
  workspace-scoped call that needs one (sessions, status)
- `GET /api/system` — pi/tmux detection + setup warnings (ADR-0003 UX).
  `pi.latest`/`pi.updateAvailable` ride along (registry check, 6 h cache);
  `POST /api/system/pi-update` runs `pi update --self`.
- `GET /ws/term?session=<name>` — xterm.js bridge (Pi TUI or project shell).
  The bridge sets `status off`, `allow-passthrough on` and extended keys on the
  session, then applies the session's **resolved** tmux options (ADR-0024) — on
  every attach, so a setting changed while the terminal was closed takes hold
  when it is reopened and an older session heals itself.
  `web/src/lib/termClipboard.js` handles OSC 52 (write only; the read form is
  refused) so a copy made in the pane reaches the system clipboard — which is
  what keeps copying possible now that `mouse on` gives the drag to tmux.
  Study: `docs/benchmarks/2026-08-30-web-terminal-clipboard.md`.
- `GET/POST /api/terminals` · `POST /api/terminals/{id}/open` · `DELETE /api/terminals/{id}` · `GET /api/terminals/{id}/cwd` — first-class shells (ADR-0017). Each row carries `workspaceId` (`ws_free` = free; ADR-0026) — the wire stays flat and the sidebar groups client-side. `POST` accepts `workspaceId`; a workspace terminal with no cwd starts in the workspace folder. The list's `cwd` and its git facts both come from the live tmux pane path (record as fallback) — the sidebar says where the terminal is, not where it was born. Workspace agent views carry per-agent git from the agent's effective directory (workPath, else the workspace path).
- `GET/PATCH /api/terminals/settings` · `GET/PATCH /api/terminals/{id}/settings`
  — terminal **behaviour** (ADR-0024). One global row plus a row per terminal
  holding only the fields that differ; `internal/termopts` is the registry of
  offered tmux options and the layering rule (defaults ← global ← overrides).
  A PATCH stores and applies live: a terminal patch touches that session, a
  global patch re-resolves **every** owned session individually, so an override
  is never overwritten by the pass that updates everyone else. "Owned" comes
  from the store — the terminals and agents this instance has records for —
  never from `tmux list-sessions`, which answers for the whole machine and
  would let one instance write into another's sessions. `null` clears a
  field — storing the inherited value instead would pin it. The curated
  registry (`mouse`, `status`, `allow-passthrough`, `extended-keys`,
  `extended-keys-format`) carries help and warnings and doubles as the
  defaults layer — the options PiCode once forced in code live here now, so a
  user override wins with no hardcoded exception. Beyond it,
  `GET /api/terminals/settings/catalog` serves the ENTIRE option space of the
  running tmux (ADR-0025), read live from `show-options -sg/-g/-wg`; any of
  it can be stored and applied, validated by tmux itself (scratch session for
  session/window values; server values apply for real) with its refusal
  surfaced in its own words. Server-scoped keys are refused on a per-terminal
  PATCH — tmux keeps them per machine and the UI labels them so. Array
  options travel as ONE string, entries joined by newlines: line *n* is
  `name[n]`, and applying rewrites the list per index, unsetting whatever the
  layer held past the new length (tmux leaves stale indexes in place
  otherwise). An empty block is refused — tmux keeps no empty array layer, so
  it could only behave as inherit while claiming to be a pin. The page is
  `#/termset` (global) and `#/termset/<id>` (one terminal).
  Appearance (font, colours, cursor) stays in `localStorage`, per browser.
- `GET /api/agents/{id}/cwd` — Pi TUI pane path (fallback: agent work dir)
- `GET /api/agents/{id}/git` · `GET /api/terminals/{id}/git` — the commit DAG,
  refs and worktrees of whatever repository that owner's cwd belongs to, plus
  the agents living in each worktree (`?limit=`, default 250). One graph per
  repository: the identity is `git rev-parse --git-common-dir`, so every
  worktree answers with the same key and collapses onto one tab. The route
  carries the *owner* because the owner is what authorises the read — the
  server never resolves a repository from a path in the URL (ADR-0022).
- `GET /api/agents/{id}/git/commit?hash=` · `GET /api/terminals/{id}/git/commit?hash=`
  — one commit with its message body and its patch, already split per file.
  `hash` must be a full object name (40/64 hex): it is the only user-supplied
  part of the git command line, so a ref or a leading dash is refused rather
  than passed through. The patch is read with `-m --first-parent`, which is
  what keeps a merge from arriving as a combined diff (`diff --cc`, `@@@`)
  that a unified-diff reader misreads without ever failing.
- `GET /api/{agents|terminals|workspaces}/{id}/browse?dir=` — one directory
  level under the owner's folder (terminals read the live pane cwd; a
  workspace reads its registered folder, `ws_free` refused — ADR-0030). The
  answer carries `root`, the canonical folder, which is the tree tab's
  identity. Workspaces also mirror `text` (GET/PUT), `blob` and `file`, so
  an empty workspace (ADR-0027) can open files with nobody in it.
- `GET /api/{agents|terminals|workspaces}/{id}/gitstatus` — the working-tree
  changes of the owner's repository, `git status --porcelain -z -uall`
  re-anchored from the repo toplevel to the owner's cwd (what falls outside
  is dropped). No repository is a state, not an error: `200 {"git": false}`.
- `GET /api/{agents|terminals|workspaces}/{id}/gitdiff?path=` — one file's
  working-tree-vs-HEAD patch (ADR-0032), confined by the same cwd rules;
  untracked files arrive as whole-file additions, binary and truncation
  flagged like the commit route. 404 when there is no difference.
- `POST /api/{agents|terminals|workspaces}/{id}/reveal` — opens the owner's
  folder (optional confined `{"path"}` body) in the host file manager via
  `internal/osopen` (WSL → explorer.exe, darwin → open, else xdg-open).
  Host-local by design: a remote browser opens it on the server's desktop.

### TerminalBridge ✅ (M1)
One tmux session per interactive agent (`internal/tmux`: create/kill/list,
exact-name matching via `=` prefix, PiCode-owned `picode-` name namespace,
ids sanitized — dots/colons are tmux target separators and corrupt
lookups). Project shells use `picode-sh-<id>` (same prefix, different name).
`internal/term` bridges WebSocket ↔ PTY (`tmux attach`):
binary frames = terminal bytes, text frames = `resize` control JSON;
closing the tab ends only the attach — the agent or shell keeps running in tmux.
Resize propagates via `TIOCSWINSZ` on the attach PTY. Requires tmux ≥ 3.5. Attach and `NewSession` set `extended-keys on` /
`extended-keys-format xterm` (modifyOtherKeys). Probed live: tmux 3.6
answers only DA1 to a pane's Kitty query, so pi falls back to
modifyOtherKeys and expects `ESC [27;2;13~`; tmux re-encodes client keys
per this format, so Shift+Enter reaches the TUI as a newline. The OSS
xterm.js (6.0.0) has neither Kitty nor modifyOtherKeys input encoding,
so `termKeys.js` encodes modified Enter itself (VS Code's terminal gets
the same result from its xterm fork). `/api/system` warns if the running
server is on another format.

### RPCBridge ✅ (M2 core)
`internal/rpc`: JSONL client for `pi --mode rpc` (strict `\n` framing via
bufio.Scanner, command/response correlation by id, event fan-out, exit
propagation) plus the **managed runtime** (ADR-0006): task delivery engine
claiming from the store (`prompt` waits for `agent_settled`; `steer` /
`follow_up` send while the turn is running), finished delivered/failed with audit), per-agent
event hub feeding `GET /ws/agent?agent=<id>` (events + `enqueue` input).
Extension dialogs (`select`/`confirm`/`input`/`editor`) surface as RPC
`extension_ui_request`. Managed snapshot includes `waiting` + `dialog`.
The GUI shows one compact stepper per turn: labeled pills for answers (click
to go back), a filterable dropdown (cmdk) for the current field. After the
last answer the thread keeps one definition line (`vision — xai/grok-4.5 ·
medium`), persisted for reload. **Yes / No / Cancel / type** POST
`/api/agents/{id}/ui` (`ReplyUI`). Notify is a toast, not waiting, unless a
form just finished.
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

### Model roles (ADR-0028, ADR-0033)

Opt-in pi package at `packages/pi-roles/` (MIT; the rest of this tree is
PolyForm Noncommercial). Users install it with `pi install -l` / `#/packages`;
a missing `<cwd>/.pi/roles.json` leaves it dormant. Three builtin behaviours
(`default`, `vision`, `plan`) plus named custom presets. `default` is the
switch-back target, not a startup override — per-agent `--model` stays with
ADR-0009. `/roles edit|add|remove` writes the workspace file, or
`<cwd>/.pi/roles/<id>.json` when `PI_ROLES_AGENT` is set (PiCode puts the
agent id in that env on RPC and TUI start). Overlay slots win; the rest is
inherited. Composer `/` lists those commands only while the agent is running
(ADR-0029).

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
