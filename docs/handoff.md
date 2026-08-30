# Handoff — living project state

> Heartbeat of PiCode. Session that changes state **must** leave this file
> matching **HEAD** (ritual: `/skill:handoff-update`, contract: AGENTS.md).
> Stale *Next up* (listing shipped work) is FAIL. Newest *Recent activity*
> first. Archive to `docs/handoff-archive.md` when this file exceeds ~150 lines.

## Current state (read this first)

**Visual gate:** `read` uiux-review **before** JSX. Empty/blocked/error
screenshots must be `read`. overlayAudit + visual-card. Clip = FAIL.
Skip = quality-gate FAIL.

**Phase:** ADE past M3. Composer **A** + MCP **B** + Track **C** (waiting, queue, draft) shipped. Slash TUI **24 ui · 0 missing**. Providers + vault.
Public docs on Pages. llama.cpp manager is **docs + dialog**, not an installer.
Local backup V1 in Preferences (ADR-0014).

What exists:
- Public `cfpperche/picode`, PolyForm Noncommercial + commercial, CI linux/macos/windows.
- HTTPS default `:8445` (ADR-0007). SW caches **only hashed `/assets/`**; HTML/API network.
- Workspaces + many agents per folder (ADR-0011). Settings vs Preferences (ADR-0012). Packages: machine / workspace / **this agent**; optional isolate skips inherit (ADR-0010).
- Providers: API key + OAuth. Claude/Codex loopback `53692`/`1455`. Copilot/Kimi/xAI **device-code**. Radius stays TUI (gateway URL).
- **ADR-0013** vault `~/.picode/accounts.json`; `auth.json` is the one slot pi reads. Add account / Use / Sign out; OAuth re-login updates the same account.
- Composer `/` opens **PiCode UI** (never the dock). Skills/templates = picker insert; pi RPC expands. `/share` = secret gist (`gh`).
- `/llama` dialog on the **current view** (URL, Save, Retry, load/unload, HF download). Link **Set up llama.cpp** → `www/guide/llama.md`. **No GUI installer** (reverted).
- Voice **V1 shipped** (dictation + Grok composer + browser TTS). Owner dogfood done (Chrome Windows mic).
- Public docs: VitePress `www/` → GitHub Pages. GUI chrome carries **state**, not docs. ADRs 0001–0013.
- `#/mcps` missing adapter: one line + Open packages (`www/guide/mcp.md`). No npm/architecture in the view.
- MCP / Packages / Settings name the selected agent (icon + name) as the first line in the card. Scope pills say **This agent**. Sidebar agent click from a pane goes to `#/`.
- MCP **Use from…** mirrors other apps. User picks; Off hides a server. Empty host files are not offered.
- MCP Add **More**: env on command servers; headers + Sign in / Token on URL servers.
- MCP list live state: Idle / Live / Failed / Sign in when the GUI agent is running. File Off has no word.
- MCP **Sign in** is automatic like providers: short `pi -e` runs headless `authenticate()` (no paste). Pi does not open the browser. GUI `window.open`s the URL once. Overlay ends when the callback HTML is served (keyring write continues). OAuth rows with tokens show **Sign out** (forgets the keyring login on this machine). No extra “Signed in” label.
- Composer `@` / images / `!cmd` shipped. MCP list/add/toggle/remove/Use from/live/auth shipped.
- Track **C1 waiting**: confirm/select/input/editor is a chat card; `POST /api/agents/{id}/ui`; sidebar says Waiting. Notify is a toast. Mobile has no waiting card.
- Track **C3** queue: Send while busy/waiting is follow-up (or Steer from the kind chip). Follow-up is held until idle — Edit / Remove. Abort drops Steer.
- Track **C2** draft: composer text + kind persist per agent (`picode-drafts`). Images do not.
- Track **D1** `#/agent/<id>`: URL is the open agent. Missing id: “That agent is gone.”
- Track **D2** session chip shows `$0.05` when the session has spent money.
- Track **D3** `@` lists files, other agents, and skills (`@agent:…` / `@skill:…`). Mentions only.
- Track **D4** `/tree` is Prompts. Now on the current card. Other cards confirm then start a new session (this one stays). In-place jump still pi#8645.
- Track **D5** behind npm packages show **Update**. User menu dots Packages. Nothing updates until you click. Git / path / pinned skipped.
- **ADR-0015** + Track **E**: E1–E4 shipped (open, Save, Keep/Undo, turn file names).
- **ADR-0017** first-class terminals (sidebar + `#/term/<id>`). ADR-0016 editor-tab UI superseded. Pi TUI dock unchanged.
- **ADR-0020** PiCode Desktop: a Windows tray binary provisions the distro; `picode provision` does the Linux half. ADR-0018 superseded (it had ruled out both a logon task and linger). **M2–M5 shipped** (the ADR-0020 plan is complete): `picode provision` (6 steps, `--dry-run` / `--json`) and `picode-desktop.exe` (tray, logon task, keepalive, CA trust, clean-machine bootstrap). Neither has been **run for real** — dry-run only, by the owner's decision.
- Preferences → **Terminal** (colors, font, size, line height, spacing, cursor, blink, scrollback, padding, **Keys**: newline + copy-if-selected). Ligatures omitted: xterm canvas in the browser cannot join glyphs (`@xterm/addon-ligatures` needs Node font-finder).

## In flight

**ADR-0022 — git graph. G1 (data), G2 (the graph) and G3 (commit diff) are
built and visually verified. The ADR is delivered.**

The repository icon beside an agent or terminal in the sidebar opens
`#/git/<t|a>/<id>`; the tab is `g:<git-common-dir>`, so two owners in two
worktrees of one repo land on the same tab. Rows carry ref chips and, on a
branch, the agents living in the worktree checked out on it. Clicking a commit
splits the surface and shows its message and patch, one card per file.

Verified on the real repo at 250 commits, dark and light: `overlayAudit ok`,
no horizontal scroll, 26px rows, the selected row stays in view when the diff
pane opens, and a 59-file commit reports `truncated` rather than trying to
render megabytes.

## Next up

**PiCode Desktop is installed and running on the owner's machine. What is
left is validation — the list below is what has *not* been proved.**

The one that matters: **`wsl --shutdown`, then a Windows sign-out and back
in.** Everything else is inference until this runs. It is the only test that
proves linger and the logon task work together with nobody signed in — which
is the entire problem PiCode Desktop was built for. Cost: it ends the running
tmux sessions, so it is the owner's call. Expected result: sign in, the tray
appears on its own, `https://localhost:8445` answers without anyone opening a
terminal.

Never exercised, because this machine was already past them:

| Path | Why it never ran | How to reach it |
|---|---|---|
| `/etc/wsl.conf` write (backup + line merge) | `systemd=true` was already set, so the fix branch never fired here | a distro without it, or a scratch `ConfPath` |
| mkcert CA import into Windows | the CA was already trusted from an earlier `setup-cert.sh` | a machine that never ran that script |
| mkcert certificate issuance | the cert is valid until 2028-11-25 | `renewWithin` is 30 days, or delete `~/.picode/cert.pem` |
| Clean-machine bootstrap (M4) | no spare Windows VM: `wsl --install`, the 3010 reboot, `RunOnce` resume, account creation | a bare Windows VM |
| Release workflow | it only fires on a tag, and none has been cut | tag `v0.1.x` |
| `picode-desktop update` | needs a published release to find | after the first tag |
| `picode-desktop uninstall` | never run | it only removes the logon task; safe to try |
| Tray menu actions | the process and its keepalive were verified, but Open / Restart / View logs were never clicked | click them |
| `install`'s own output | the elevated copy writes to a console this session could not see; only its *effects* were checked (task registered, CA trusted) | run it from a Windows terminal |

## Backlog

- llama.cpp: in-app installer / start router, SSE progress + cancel, delete `.gguf`, Ollama/vLLM (`models.json`).
- Mobile parity (shell exists; not feature-complete).
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.
- Cold start parses the whole session JSONL (10 s on a 129 MB session) — filed upstream: [pi#8843](https://github.com/earendil-works/pi/issues/8843) (lazy resume / load checkpoint).
- Worktrees / parallel isolated agents (Orca + Herdr) — after Track E.

## Known debts / open questions

- `handleAddWorkspaceAgent` (`internal/server/agents.go:504`) hardcodes
  `workPath` to `""`, so a **workspace** agent cannot be pointed at a worktree
  through the API — only a free agent takes a path (`POST /api/agents?free=1`).
  ADR-0022's centrepiece is agents in sibling worktrees, and today the only way
  to get one is a free agent or a second workspace. Found while seeding the
  visual check; not fixed here because it changes the agent-creation contract.

- JS tests still do not run in `make ci`. `npm test` was widened from 2 files
  to all of `src/lib/*.test.js` (159 → 175 tests, all passing), but wiring it
  into `make test` would run it before `make build` installs `node_modules`,
  and breaking CI is worse than the gap. The ordering has to change first.

- ADR-0022 leaves two costs unmeasured. **Occupant scan**: marking heads
  means `ListAllAgents`, resolving each cwd, and a git call per agent to get
  its common dir — `occupantsOf` (`internal/server/cleanup.go:88`) is the
  shape to generalise, but nobody has timed it with many agents. **Commit
  ceiling**: the SVG weight past which the graph stops being usable is
  unknown; 250 with a load-more is a guess borrowed from Git Graph's 300.

- Machine state left by the ADR-0020 session: `~/.local/bin/picode` was
  replaced with a build of `main` (the old one is not kept in the repo — it
  predates `provision`), `picode-desktop.exe` sits in
  `%LOCALAPPDATA%\PiCode\`, and the `PiCodeDesktop` logon task is
  registered. `picode.service` is enabled and running as PID 447165, which
  still runs the *old* binary — the new one takes effect on the next restart.

- MCP GET redacts env/header values (keys only). `bearerToken` stays write-only.
- MCP **installed + zero servers** Add form is live (`docs/screenshots/mcp-named.png`). Sidebar-on-pane → `#/` checked in browser, not unit-tested.
- MCP Sign in auto-start after Add/On is UI-only (no unit test). Reuse of a running agent shares `beginAuthOn` with the short-pi tests.
- Two concurrent agents share whichever credential is **active** in `auth.json` (pi limitation; vault does not fork that).
- Token auth: ADR-0007 personal-network trust; mandatory only if exposed beyond the tailnet.
- `internal/proclock` leftover `picode.lock` after a Windows crash.
- Vendored xterm.js 5.5.0 — manual upgrade (ADR-0004).
- Branch protection + CODEOWNERS — owner action on GitHub.
- tmux-gated tests skip on windows/macos CI (accepted).
- Mobile shell has no waiting card (C1 is desktop).
- Terminal **ligatures** not offered: `@xterm/addon-ligatures` needs Node `font-finder`; browser xterm is canvas-only.
- `picode provision`'s cert step issues for loopback + local IPv4 (`tlsutil.LocalNames`) and falls back to self-signed without mkcert. It does **not** yet cover what `scripts/setup-cert.sh` still owns: installing mkcert, `mkcert -install` into the Linux trust store, and the Windows CA import (that one moves to `picode-desktop.exe`, which is already elevated).
- `install_windows.go` is a stub returning an error. ADR-0020 gives Windows a real path, but through `picode-desktop.exe`, not through that file.

## Recent activity

- **2026-08-30** — Git graph G3 (ADR-0022): clicking a commit opens its diff. `git show -m --first-parent` is a correctness fix, not a preference — without it a merge arrives as a combined diff (`diff --cc`, `@@@`) that the unified-diff reader misreads silently; proved by removing the flags and watching the test go red. The hash is the only user-controlled part of the git command line, so anything but 40/64 hex is refused before it reaches git. `DiffLine` moved out of Conversation.jsx so the chat and the graph render diffs the same way. Screenshot caught the selected row scrolling out of sight when the pane opened.
- **2026-08-30** — Git graph G1+G2 (ADR-0022). `internal/gitgraph` reads the DAG, refs and worktrees; the column allocator is ported from mhutchie's Git Graph (MIT, attributed in the file header) minus its uncommitted-changes row. Two parser bugs caught by tests with teeth: git hands back a literal 0x1f typed into a message, which a plain Split turned into a *dropped commit*, and a 0x1e split a record into a phantom commit whose hash was someone's subject. Verified on the real repo: 250 rows, `overlayAudit ok`, no h-scroll, 26px rows, dark and light both read, and the occupant chips show `default` on main beside `graph-impl` on its worktree.
- **2026-08-30** — Agent TUI and PiCode terminals share **one surface**: the desktop agent terminal view now renders through TermSurface/ShellTerm (same xterm options, wheel/keys/links wiring, padding 0, custom scrollbar, screen margin) instead of the TerminalDock wrapper — computed-style diff between the two is now identical. Managed mode shows a one-line hint + Open TUI action. `closeTerm` moved to `lib/terms.js` (mobile still uses the dock component). Also recovered `web/node_modules` in the primary checkout after a parallel session replaced it with a self-referential symlink.
- **2026-08-30** — Compaction residue fixed at the root: the transcript window now cuts at the **last compaction boundary** (what pi replays), so reload no longer resurrects pre-compaction history. The summary renders as a collapsible **Session compacted** card (39 K chars confined to 253 px + scroll) instead of a giant assistant message. `compacted` in the API response separates "needs /compact" from "already compacted, file stays large" (cold boots stay slow — pi#8843).
- **2026-08-30** — /compact progress moved out of the conversation into the **composer statusbar**: "⠋ Compacting 1:23" segment with spinner + 1 s ticker, persisted per agent in localStorage (survives the TUI→managed panel rebuild and page reloads), cleared by the HTTP answer or `compaction_end`. Verified live on the 140 MB adopted session (78 events post-boundary vs 9 673 raw).
- **2026-08-30** — Sidebar spinner covers TUI work too: GET /api/tui-working polls tmux capture-pane for pi's Working state (3 s).
- **2026-08-30** — /compact is visible end to end: "Compacting session…" line in the thread, closes on HTTP answer or pi's compaction_end, "too small" maps to "Nothing left to compact."
- **2026-08-30** — Terminal/chat view persists per agent across reload (localStorage `picode-term-view`).
- **2026-08-30** — Transcript API paginated (`?tail=&skip=`): opening the 129.5 MB session loads a 200-event slice (~1 s server, small payload); Load earlier fetches older turns from the server with scroll anchoring.
- **2026-08-30** — Huge sessions render a window (~60 turns) with Load earlier on scroll; composer height tracks local text. Proven on a 122 MB session (226 turns → 60 mounted).
- **2026-08-29** — ADR-0022: the git graph belongs to a repository, not to an agent. Studied VS Code's Git Graph first (it is mhutchie's extension, not VS Code): one runtime dependency, `iconv-lite`, no framework and no charting library — 913 lines of DOM + SVG, of which 73 place the columns. Measured that worktrees share refs and differ only in HEAD, which is why one graph with N marked heads beats N near-identical graphs. Read-only in this ADR: no write path exists that could hit a worktree while an agent is mid-turn.
- **2026-08-29** — ADR-0021 adopt Pi session by copy. GET/POST `/api/pi-sessions`. New agent → From a Pi session.
