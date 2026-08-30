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

**Multi-runtime TUIs in terminals** (owner direction, research in flight in
another session): the ADE will host TUIs from several agent runtimes — in
**terminal surfaces only**; agents stay **Pi-only for now**. The unified
TermSurface/ShellTerm path is the substrate (any tmux session renders the
same). Known landmine for that research: tmux's `extended-keys-format` is
server-wide — we force `xterm` (modifyOtherKeys) so Shift+Enter survives,
but some TUIs prefer csi-u/Kitty; per-session key format may be needed.

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


- ADR-0022's occupant scan has a cliff, now measured. With every agent's cwd
  **at** a worktree root it is free — 2.7ms for 500 agents, no subprocesses.
  With cwds **below** a root each agent costs a `gitgraph.Key` call, ~23ms:
  10 agents → 274ms *per graph load*, 200 → 4.6s. Distinct directories and one
  shared directory cost the same (4.7s vs 4.6s at 200), so it is the call
  count, not the paths. Not fixed: the shape the product produces today is
  agents at worktree roots, and no one has hit this. If it is worth insuring,
  memoising `Key` per cwd inside one request collapses the shared-directory
  case to a single call and costs about ten lines.

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

- **2026-08-30** — ADR-0022's two unmeasured costs, measured. **Commit ceiling: there isn't one** within what the product allows — layout is 14ms for 10,000 commits, the server answers `?limit=2000` in 0.12s (408KB), and the browser holds 2,000 rows / 17k DOM nodes with a row click at 0.4ms and scrolling at 0.1ms. The 250 default is conservative, not a limit. **Occupant scan has a cliff**: free when agents sit at worktree roots, ~23ms per agent whose cwd is below one — see debts. Also worth recording: mid-measurement I nearly filed 'Load earlier is broken' as a bug. Instrumenting the button showed the clicks were never reaching it — `agent-browser click "text=…"` does not hit it, while dispatching on the element does. The feature works.
- **2026-08-30** — `picode install` / `deploy` survive a non-login shell. `systemctl --user` needs `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`, which a script, cron job or agent shell does not have; both commands copied the binary *before* calling it, so the failure left the new binary on disk with the old one running — hit exactly that during today's deploy, and only a hash comparison showed it. `install.Run` now fills the two variables from `/run/user/<own uid>` when the socket is there, and `EnsureUserSession` refuses before copying when it is not, naming `loginctl enable-linger`. Verified the injected values turn `systemctl --user is-system-running` from a bus error into `running`, and that the guard is what prevents the half-update: without it the test finds the installed binary replaced anyway.
- **2026-08-30** — Frontend tests run in CI. 197 of them were passing where nothing watched. The blocker was ordering — `npm test` needed `node_modules`, which only `make build` installed — so installing moved into its own target gated on `web/package-lock.json`, and `web` and the new `test-js` both depend on it. That also removes a second full `npm ci` per `make ci`. Verified the gate can actually fail: a deliberately broken test takes `make ci` to exit 2, and the guard skips the install on a second run (10s → 2.7s) but reruns it when the lockfile is touched.
- **2026-08-30** — `POST /api/workspaces/{id}/agents` accepts `workPath`; it was hardcoded empty, so ADR-0022's centrepiece — two agents in sibling worktrees of one repo — could only be built from free agents. Reuses `resolveAgentWorkDir`, the same resolver free agents use, so the two creation paths cannot drift; blank stays blank and keeps the agent on the workspace folder. Verified the test has teeth: with the old hardcode both agents pile onto `main`. **API only — no UI was added**, since nothing asked for one.
- **2026-08-30** — Clipboard validated in a browser, closing the ADR-0023-era debt. Text emitted as OSC 52 from inside a tmux pane came back out of the *system* clipboard via a real Ctrl+V — the whole chain, not an inference. Chrome refuses the write without a recent user gesture and accepts it right after a click; the handler's toast covers the refusal. That refusal path needed a synthetic probe to reach at all, so it is not being designed around. Firefox is still unchecked: the automation here only drives Chrome.
- **2026-08-30** — `make fmt` and `make fmt-check` stop reaching into `.worktrees/`. Both walked the tree with `.`, so a sibling agent's uncommitted code failed this gate — and `fmt` was worse than reported: `gofmt -w .` would have *rewritten* their files. Both now walk the directories `go list ./...` reports, the same module boundary `vet` and `test` always respected. The `git ls-files` fix I had written in this file was wrong and is dropped: tested, it misses a new file not yet `git add`ed, so it would pass locally and fail in CI. `go list` catches it, and covers 204 of 204 `.go` files in the module including `//go:build ignore` ones. CI's inline gofmt step now calls the same target instead of repeating the command.
- **2026-08-30** — ADR-0023 implemented. `internal/web` splits into a disk loader (default) and an embedded one (`-tags embedui`), which is what `make build`, `ci.yml` and `release.yml` now use; `internal/web/public/` is untracked and gitignored. The ADR's gating question is answered: over https the service worker registers, activates and fills `picode-assets-v1` with the hashed assets in disk mode, and both modes serve byte-identical asset URLs with identical Cache-Control (`immutable` for `/assets/`, `no-cache` elsewhere). The earlier `swRegs: 0` was a red herring — `main.jsx:16` only registers over https, so it was 0 in both modes. A disk build with no UI answers 503 with `run make web` instead of 404s, and starts serving the moment the files appear.
- **2026-08-30** — ADR-0023: the built UI stops being committed. It had been tracked since the bootstrap commit with no decision behind it — 330 files, 33 MB, 77% of the repo's commits, 133 files rewritten per UI change, and 335/334 `rename/rename` conflicts in two merges that day, every one resolved by rebuilding. Checked six peers (Grafana, Prometheus, Vault, Coder, Gitea, Syncthing): none commits it. Prometheus also answers the constraint that kept us — `//go:build !builtinassets` serves from disk and the default build does not embed at all. Decision recorded; the code change is not made yet.
- **2026-08-30** — User menu gained a **Sessions** item (between System and Providers) that opens the machine-wide `#/sessions` view; `go("sessions")` short-circuits the `/sessions/:id` template. Live click-through verified: menu → 36 folders · 99 sessions · audit ok.
- **2026-08-30** — Terminals stopped wearing tmux's skin. PiCode forced tmux's own `mouse on` since before `termWheel.js` grew its SGR fallback; its only remaining effect was tmux copy-mode on every drag, which is why text could not be selected. Owner tested `mouse off` on the real machine — the wheel still scrolls — so both call sites drop it. Paired with `allow-passthrough on` and a write-only OSC 52 handler so a copy made inside the pane reaches the system clipboard. Proved A/B in the browser: with passthrough on the handler fires, with it off the sequence never arrives. The read form (`52;c;?`) is refused on purpose — `@xterm/addon-clipboard` implements it, which would let any agent in a pane read the user's clipboard. Benchmark: `docs/benchmarks/2026-08-30-web-terminal-clipboard.md`.
- **2026-08-30** — **All-folders Sessions view** (`#/sessions`): every Pi session on the machine grouped by folder (workspaces first; "not a workspace" badges; 36 folders / 99 sessions / ~554 MB on the QA machine). Same actions per row — Open with… only where a workspace owns the folder (disabled + reason otherwise), Delete validated against the sessions root, Compact for in-use. New endpoints `GET/DELETE /api/sessions/all`; `ListAll` summaries now carry size. QA live: fixture in a non-workspace folder deleted end-to-end, scope link both ways, audit ok after height fix (19→36 px on the scope link).
- **2026-08-30** — **Sessions view shipped (A+B)**: `#/sessions/<id>` lists every Pi session under the folder (size/age/msgs/cost/provider, `inUseBy`) with Open with… (resume, no copy), Compact (in-use), Delete (orphans, confirm; in-use 409 with reason) — plus **auto-clean orphans** (Off/30/60/90 d; boot+daily+on-change sweep, default Off). New: `session.ListDir` (+Size in Summary, missing dir = empty), manage endpoints, `StartSessionSweep`. QA live on the 629 MB workspace: fixture deleted end-to-end, in-use blocked with tooltip, dialog lists workspace agents, empty state after the missing-dir fix, audit ok. Known debt: list rescans all JSONLs (same cost as the adopt picker) — needs a stat-only mode if it ever feels slow.
