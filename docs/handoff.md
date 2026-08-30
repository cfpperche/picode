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

**ADR-0022 — git graph.** Decision accepted, no code yet. One graph per
repository (`git rev-parse --git-common-dir`), so every worktree of a repo
shares one tab and each worktree's HEAD is marked with the agents living
there. Read-only: a commit opens its diff, nothing writes. Route carries the
owner (`#/git/<t|a>/<id>`), the tab id carries the repo (`g:<key>`) — that
split is what keeps `relUnderCwd` confinement intact. Layout runs in the
browser; Go returns data on one endpoint.

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

- **2026-08-30** — Terminal/chat view persists per agent across reload (localStorage `picode-term-view`).
- **2026-08-30** — Transcript API paginated (`?tail=&skip=`): opening the 129.5 MB session loads a 200-event slice (~1 s server, small payload); Load earlier fetches older turns from the server with scroll anchoring.
- **2026-08-30** — Huge sessions render a window (~60 turns) with Load earlier on scroll; composer height tracks local text. Proven on a 122 MB session (226 turns → 60 mounted).
- **2026-08-29** — ADR-0022: the git graph belongs to a repository, not to an agent. Studied VS Code's Git Graph first (it is mhutchie's extension, not VS Code): one runtime dependency, `iconv-lite`, no framework and no charting library — 913 lines of DOM + SVG, of which 73 place the columns. Measured that worktrees share refs and differ only in HEAD, which is why one graph with N marked heads beats N near-identical graphs. Read-only in this ADR: no write path exists that could hit a worktree while an agent is mid-turn.
- **2026-08-29** — ADR-0021 adopt Pi session by copy. GET/POST `/api/pi-sessions`. New agent → From a Pi session.
- **2026-08-29** — Agents work in an isolated git worktree (AGENTS.md #5). After merge, remove the tree and the branch.
- **2026-08-29** — Keepalive debt paid: the tray's `wsl.exe` child now lives in a job object with `KILL_ON_JOB_CLOSE`, so the kernel tears it down however the tray dies — `taskkill /F` and crashes included, where no deferred Go code runs. Established first that killing the Windows-side `wsl.exe` does end the Linux process (launched a `sleep 99999`, killed its wsl.exe, watched it go), because the whole fix rests on that. Then proved it end to end on the machine: before, a force-killed tray left `sleep infinity` behind; after, `taskkill /F` leaves nothing. `startDetached` became `startSupervised`, and a failure to supervise is non-fatal — a keepalive that can be stranded beats no keepalive.
- **2026-08-29** — Tray icon is now the browser favicon (`web/public/favicon.svg`): the blocky Pi in white on `#09090b`, transcribed as rectangles in `mkicon.go` and drawn at 4x then boxed down so 16px stays legible. `icon_test.go` reads the SVG's fills and asserts the committed ICO carries both — verified it actually fails by regenerating with the old indigo. Restarted the tray on the owner's machine with the new mark (PID 29532); doctor still reports everything ok on both halves.
- **2026-08-29** — **PiCode Desktop is live on the owner's machine.** Ran `picode-desktop.exe` through WSL interop: `doctor` showed the mkcert CA already trusted from an earlier `setup-cert.sh`, leaving only the logon task; `install` elevated (one UAC) and registered `PiCodeDesktop` — `At logon time`, `Run As User: cfpp`, `"…\picode-desktop.exe" --tray`. The tray runs (PID 30320) with its `sleep infinity` keepalive holding the distro open. Every step now reports ok on both halves. Fixed a fourth bug the run exposed: `doctor` summarised only the distro, so a fully set-up machine was still told to run install; `printWindowsSteps` now reports whether its half is finished and the summary weighs both.
- **2026-08-29** — Ran the real provision on the owner's machine (Linux half). One change: `Linger=no` → `yes`. `/etc/wsl.conf` came back with the **same md5** and no `.picode.bak`; PID 447165 and all three tmux sessions untouched, because the unit was already enabled and running so nothing restarted. The root pass then exposed a third bug: `systemctl --user` always answers for the *calling* account, so from root it reported goat's running service as "present but not enabled" — confident and wrong. `Env.Acting` now records who the process actually is, `OnBehalf()` compares it to the target, and the service check returns **blocked** instead of guessing. The summary line also stopped counting a skip as "would change". Windows half (logon task, CA trust, keepalive) still not run — it needs the `.exe` on Windows plus one UAC click.
- **2026-08-29** — First `picode-desktop doctor` run found two real bugs no test had. (1) The root pass invoked a bare `picode`, which is on **no** PATH but the owner's — ADR-0018 installs to `~/.local/bin`, reachable only through that account's profile. `PicodePath` now resolves it once via the owner's **login** shell (`sh -lc`) and hands both passes the absolute path. (2) Far worse: `cmd/picode` fell through to `serve()` for any unrecognised argument, so the *old* installed picode, asked to `provision`, silently started a second server **as root** in `/root/.picode` on port 8446. Killed it and removed the directory; the owner's picode on 8445 was untouched. `dispatch` now owns the command list and a bare word it does not know exits 2.
- **2026-08-29** — Desktop **M5**: release plumbing (ADR-0020, plan complete). Tag-triggered `release.yml` builds picode for 5 platforms plus `picode-desktop-windows-amd64.exe` into one release; `version.Version` became a var so `-X` can stamp it (verified: a stamped build reports 9.9.9). `install.LatestReleaseFor` extracted so both binaries share one update check. A test reads the workflow and fails if the asset names drift from what `update` looks for — the kind of break that is silent forever. Tray icon: a pi generated by `mkicon.go` (`//go:build ignore`, `go:generate`) at six sizes, BMP payloads; the first draft merged its legs into a "T" at 16px, so the geometry was fixed against a rendered preview. Elevation is runtime `ShellExecuteW`/`runas`, **not** a manifest — the same binary is the tray, and `requireAdministrator` would elevate that too.
- **2026-08-29** — Desktop **M4**: the clean-machine path (ADR-0020). A stage machine derived from observed state, not saved progress, so an interrupted install resumes and a finished one is a no-op: install WSL `--no-distribution` → restart → install the distro `--no-launch` → create the account → provision. Both flags dodge the interactive account setup plain `wsl --install` opens. Exit **3010** is "succeeded, restart pending", not a failure; `RunOnce` resumes at the next logon and deletes itself. The Linux account is named after the Windows one (accent-folded, sanitised) with the password left **locked** — PiCode reaches root via `wsl -u root` and never needs sudo. Writing the 3010 test found that POSIX shells truncate exit status to 8 bits (3010 arrives as 194) while Windows does not, so the rule is tested through an interface rather than a real process. Live-WSL test confirms this machine is classified as ready and skips every stage.
- **2026-08-29** — Desktop **M3**: `cmd/picode-desktop` + `internal/desktop` (ADR-0020). Drives the distro with two `picode provision --json` passes (root, then owner) and merges them so whichever pass resolved a step wins — the rule that keeps "skipped for lack of privilege" from masking "fixed". Windows side: `onlogon` task at `/rl limited` (an elevated tray cannot reach the notification area), mkcert CA import gated on a count so logon does not re-import, `sleep infinity` keepalive against the idle timeout, `CREATE_NO_WINDOW` on every child. `wsl.exe` output is UTF-16LE **with no BOM** — decoding is decided by inspecting bytes, and the real 136-byte output is a base64 fixture. `make desktop` cross-compiles from WSL: 6.5 MB PE32+ GUI, `CGO_ENABLED=0`, no C compiler. Live-WSL tests (skip in CI) confirm it picks `Ubuntu` (WSL 2) and reads the account `goat`. New dep: `fyne.io/systray` (pure Go on Windows). **Not executed against the machine** — no `install` run.
- **2026-08-29** — Desktop **M2**: `picode provision` (ADR-0020) converges six steps — wsl.conf, systemd, linger, cert, unit, health — with `--dry-run` and `--json`, and root vs user scopes so the Windows side can drive it in two calls. `EnsureKey` merges `/etc/wsl.conf` by line: the owner's real file (comment, key order, `generateResolvConf = false` spacing) is a test fixture asserted byte-identical, and the fix writes no backup when nothing changed. Writing the `Run` decision table caught a real bug: blocked steps were reported as "planned" in a dry run, promising a fix no run could deliver. Extracted `tlsutil.LocalNames` so the self-signed and mkcert paths issue for the same hosts. Dry run on the owner's machine: 4 ok, 2 to fix (linger, unit) — matching the plan, with `/etc/wsl.conf` verified unchanged (same md5, no `.picode.bak`).
- **2026-08-29** — Track 3 live cwd: Ctrl+click asks tmux `#{pane_current_path}`. File-preview roadmap closed. Tests: PaneCwd + GET `/api/terminals/{id}/cwd` after `cd`.
- **2026-08-29** — Chat file cards sit under the turn file names (one per click). visual-review: PASS (file-chat-cards-inline.png).
- **2026-08-29** — Terminal: Shift+drag select, Ctrl+C copy if selected, Ctrl+V paste. visual-review: PASS (pref-term-copy.png).
- **2026-08-29** — Sidebar Terminals list flush like Pins. visual-review: PASS (terms-flush.png).
- **2026-08-29** — **ADR-0020** accepted: PiCode Desktop provisions the distro from Windows (tray `.exe` at the WSL boundary, `picode provision` inside). Supersedes ADR-0018, whose "Alternatives considered" had rejected both the logon task and linger; the linger objection is answered by enabling it on install and never disabling it. Carries a preservation contract (`~/.picode` untouched, tmux survives, `wsl.conf` line-merged after backup, cert reissued only near expiry). Docs only — no code, no user-visible change, so no CHANGELOG entry.
- **2026-08-29** — Chat file card: click a path → closable card in the thread; Open in tab → same `#/file/a/…` as the terminal. Split FilePane removed. visual-review: PASS (file-chat-card.png, file-chat-tab.png).
- **2026-08-29** — File preview track 1: png, pdf, md, audio, video, glb/gltf (model-viewer). visual-review: PASS (file-png-preview.png, file-md-preview.png, file-pdf-preview.png, file-audio-preview.png, file-video-preview.png, file-glb-preview.png, file-bin-raw.png, file-bin-gone.png).
- **2026-08-29** — Ctrl+click on a terminal path eats mousedown/mouseup so tmux SGR (`<16;NaN;NaNm`) does not land in the Pi composer.
- **2026-08-29** — Terminal Ctrl+click also matches bare `App.jsx` / `foo.js` (not only paths with `/`).
Older blocks: [handoff-archive.md](handoff-archive.md).
