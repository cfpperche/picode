# Handoff archive

Moved off `docs/handoff.md` when it exceeded ~150 lines. Newest living
state is always `docs/handoff.md`. Do not treat this file as current.

## Recent activity (archived 2026-08-30)

- **2026-08-30** — Compaction residue fixed at the root: the transcript window now cuts at the **last compaction boundary** (what pi replays), so reload no longer resurrects pre-compaction history. The summary renders as a collapsible **Session compacted** card (39 K chars confined to 253 px + scroll) instead of a giant assistant message. `compacted` in the API response separates "needs /compact" from "already compacted, file stays large" (cold boots stay slow — pi#8843).
- **2026-08-30** — /compact progress moved out of the conversation into the **composer statusbar**: "⠋ Compacting 1:23" segment with spinner + 1 s ticker, persisted per agent in localStorage (survives the TUI→managed panel rebuild and page reloads), cleared by the HTTP answer or `compaction_end`. Verified live on the 140 MB adopted session (78 events post-boundary vs 9 673 raw).
- **2026-08-30** — Sidebar spinner covers TUI work too: GET /api/tui-working polls tmux capture-pane for pi's Working state (3 s).
- **2026-08-30** — /compact is visible end to end: "Compacting session…" line in the thread, closes on HTTP answer or pi's compaction_end, "too small" maps to "Nothing left to compact."
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

## Recent activity (archived 2026-08-29)

- **2026-08-29** — File V1.1: `.svg` / `.mmd` open Preview \| Raw \| Save (one chip group). Empty: “Nothing to preview.” Bad mermaid: “Can't draw this diagram.” visual-review: PASS (file-svg-preview.png, file-svg-raw.png, file-mmd-preview.png, file-mmd-empty.png, file-mmd-error.png).
- **2026-08-29** — ADR-0019: Ctrl/Cmd+click a path in the terminal opens `#/file/…` on the tab strip (text only). http(s) → browser. visual-review: PASS (file-tab.png, file-tab-gone.png, file-tab-outside.png). Chat FilePane unchanged.
- **2026-08-29** — Reconnect covers fast restarts: health `bootId` + WS-close kick → tab reloads itself (proved: restart → page age reset). Shift+Enter trusted-key E2E: bytes `[27;2;13~` only, multiline composer confirmed; earlier "submit" reading was a bad pane-tail interpretation. keypress guard added for browsers that fire it after a canceled keydown.
- **2026-08-29** — Shift+Enter three-layer fix: ground truth via session JSONL proved the user's Windows Chrome submits (a stray `\r` after the canceled keydown). `termDataFilter` on `term.onData` swaps/drops that `\r` (120ms window). Keydown tracker lives on the xterm textarea (capture).
- **2026-08-29** — Terminal Ctrl+C copies if text is selected (Warp / Windows Terminal); else interrupt. Keys list in Preferences → Terminal; `/hotkeys` shows them.
- **2026-08-29** — Settings **Keys**: search / Add (press a key) / Reset writes `~/.pi/agent/keybindings.json`. visual-review: PASS (settings-keys.png, settings-keys-empty.png, settings-keys-listen.png).
- **2026-08-29** — Terminal scroll: tmux `mouse on` so xterm.js enables mouse tracking (#426); screen `margin-right` so the scrollbar is clickable (#1751); stop capturing wheel (that blocked SGR). visual-review: PASS (term-scrollbar.png, `enable-mouse-events` true).
- **2026-08-29** — Tabs flush (no left pad); drag to reorder. Terminal: debounce resize 150ms; wheel sends SGR to TUI transcript; Shift+Enter newline (Preferences → Terminal). visual-review: PASS (tabs-reorder.png, first tab gap 0).
- **2026-08-29** — Terminal: ResizeObserver fits the pane as soon as the sidebar/split changes (not only window.resize). Wheel capture: scroll xterm, else PageUp/PageDown to the TUI. visual-review: PASS (term-resize.png; screen 1000→1168px when sidebar 244→80).
- **2026-08-29** — Terminal wheel on a TUI (Pi) pages/scrolls the view instead of the composer cursor. Stopped stretching `.xterm-screen` to 100% (broke hit-testing + scrollbar). visual-review: PASS (term-wheel-scrolled.png).
- **2026-08-29** — Composer chips joined into button groups (session+New; provider/model/thinking/mode/kind). visual-review: PASS (composer-chip-groups.png, composer-chip-groups-run.png).
- **2026-08-29** — Machine menu: Theme/Layout are one left-aligned capsule (radio semantics); entries have icons. Fixed segment svg shrinking to 0 (`flex:none` + `flex:auto`). visual-review: PASS (user-menu.png).
- **2026-08-29** — Composer Expand + More menu; sketch Cancel/Insert 28px. visual-review: PASS (composer-pages.png, composer-more.png, composer-sketch.png).
- **2026-08-29** — Preferences → Terminal is two columns (controls + live xterm preview). visual-review: PASS (pref-terminal.png).
- **2026-08-29** — `make deploy` / `picode deploy` = repo → systemd. `picode update` = GitHub release for a normal user.
- **2026-08-29** — `picode install` / `uninstall` (systemd --user, ADR-0018). No Windows task.
- **2026-08-28** — Rename terminals (pencil / double-click). visual-review: PASS (term-rename.png, term-renamed.png). Terminal Light/Dark in Preferences (pref-term-theme.png), independent of the app theme.
- **2026-08-28** — File pane header compact (24px); Save, Close, Expand last. visual-review: PASS (file-header.png, file-expanded.png, overlayAudit ok).
- **2026-08-28** — File pane left edge resizes (persists). visual-review: PASS (file-resize.png).
- **2026-08-28** — File pane syntax colors follow GUI light/dark. visual-review: PASS (file-theme-light.png, file-theme-dark.png, overlayAudit ok).
- **2026-08-28** — Track E4 turn file names. visual-review: PASS (turn-files.png, turn-files-open.png, overlayAudit ok).
- **2026-08-28** — Track E3 Keep/Undo on edit diffs. visual-review: PASS (hunk-keep.png, hunk-kept.png, hunk-undo.png, overlayAudit ok).
- **2026-08-28** — Track E2 CodeMirror Save. visual-review: PASS (file-edit.png, file-discard.png, overlayAudit ok).
- **2026-08-27** — Track E1 open file beside chat. visual-review: PASS (file-open.png, file-gone.png, file-binary.png, overlayAudit ok).
- **2026-08-27** — ADR-0015 + Track E plan (file pane, Save, hunk Keep/Undo). Herdr study: runtime peer, not the editor bar.
- **2026-08-27** — Voice V1 owner dogfood: Chrome Windows mic works. No V1.1 unless a quality gap shows up.
- **2026-08-27** — Track D5 package updates (badge + Update). npm only. visual-review: PASS (pkg-update.png, pkg-update-menu.png, overlayAudit ok).
- **2026-08-27** — Track D4 Prompts timeline (Now + continue in a new session). visual-review: PASS (prompts-tree.png, prompts-continue.png, overlayAudit ok).
- **2026-08-27** — Track D3 `@` mentions (agent / skill / file). visual-review: PASS (at-mention.png, at-mention-empty.png, overlayAudit ok).
- **2026-08-27** — Track D2 cost on the session chip. visual-review: PASS (session-cost.png, overlayAudit ok).
- **2026-08-27** — D1 loop: hash apply no longer depends on selectedId (tabs vs URL freeze). visual-review: PASS (agent-url-switch.png, overlayAudit ok).
- **2026-08-27** — Track D1 `#/agent/<id>`. visual-review: PASS (agent-url.png, agent-url-gone.png, overlayAudit ok).
- **2026-08-27** — Plan: `@agent` / `@skill` are mentions (context in this prompt). Agents talking to each other is the broker item, later.
- **2026-08-27** — Track C2 draft persistence (text + kind per agent). visual-review: PASS (chat-draft-reload.png, overlayAudit ok).
- **2026-08-27** — Follow-up queue: Edit / Remove on the bubble (held until idle). visual-review: PASS (chat-queue-edit.png, overlayAudit ok).
- **2026-08-27** — Track C3 visible queue: Send while busy/waiting; prompt→follow-up; abort drops Steer. visual-review: PASS (chat-queued.png, overlayAudit ok).
- **2026-08-27** — Track C1 waiting: extension dialogs in the conversation (`POST /api/agents/{id}/ui`). Notify is a toast. visual-review: PASS (chat-waiting.png, chat-ask-cancel.png, overlayAudit ok).
- **2026-08-27** — Track C roadmap: waiting → queue → draft (`docs/design/conversation-control-roadmap.md`). Next-roadmap gaps recorded there (rewind, cost, `#/agent/<id>`, extra `@`, hunk accept, broker, ACP, IDE). Backlog unchanged.
- **2026-08-27** — Cleared MCP dogfood: `picode-dogfood-*` out of Claude/Codex/Grok; Notion/Linear tokens out of the keyring; `~/.picode/mcp-auth` job files gone. Claude keeps `context7`.
- **2026-08-27** — MCP layout: Servers (list or “No servers yet.”) then Add with **Use from…** as first chip. Dropped “Using Claude Code”. visual-review: PASS (mcp-named.png, mcp-use-from.png, overlayAudit ok).
- **2026-08-27** — Signed-in rows do not also say “Signed in”; **Sign out** is the state. visual-review: PASS (mcp-signout.png).
- **2026-08-27** — MCP **Sign out** forgets the keyring login (Off does not). Linux secret-tool. visual-review: PASS (mcp-signout.png, mcp-signout-confirm.png, overlayAudit ok).
- **2026-08-27** — After Sign in the toast fired but the row still said Sign in: overlay ends on callback before keyring write, and Idle list does not poll without a running agent. Optimistic **Signed in** + retry load. visual-review: PASS (mcp-linear-signed.png).
- **2026-08-27** — Linear Sign in was sending `redirect_uri=https://mcp.linear.app/callback` (Linear homepage, overlay hung). Now we register a localhost callback. Authorize URL verified `127.0.0.1` + refuse non-loopback.
- **2026-08-27** — Add GitHub still saves the row; Copilot DCR failure is a toast, not a failed Add. Dogfood Linear in Claude (`picode-dogfood-linear`). visual-review: PASS (mcp-linear-list.png).
- **2026-08-27** — MCP overlay ends on callback (not after ~40s keyring write). Signed-in OAuth rows say **Signed in** and hide Sign in. visual-review: PASS (mcp-signed-in.png). Signed-in detection is Linux secret-tool (WSL); macOS/Windows later.
- **2026-08-27** — MCP Sign in: GUI opens the authorize tab (Pi/PowerShell does not) so the PiCode callback can `window.close()` like Claude/Codex. visual-review: PASS (mcp-signin-auto.png, overlayAudit ok).
- **2026-08-27** — MCP callback page is the same PiCode success HTML as providers (close + return to `#/mcps`), not the adapter's "return to pi".
- **2026-08-27** — MCP Sign in no longer uses `/mcp-auth` paste UI. Headless `authenticate()` (callback only) writes a result file; overlay ends when that file is ok. visual-review: PASS (mcp-signin-auto.png).
- **2026-08-27** — MCP Sign in opened two Notion tabs (GUI window.open + Pi open on WSL). GUI no longer opens a tab. visual-review: PASS (mcp-signin-auto.png, overlay unchanged).

## Recent activity (archived 2026-08-27)

- **2026-08-26** — MCP Sign in is automatic (no paste): GUI opens the tab, adapter callback auto-closes it, overlay ends on success notify. visual-review: PASS (mcp-signin-auto.png).
- **2026-08-26** — MCP Sign in overlay stayed up after Notion Authorization Successful (callback did not unblock `/mcp-auth` UI). Now notify success finishes the wait; Paste is always there.
- **2026-08-26** — MCP Sign in opened two Notion tabs (GUI window.open + Pi open()). GUI no longer opens a second. visual-review: PASS (mcp-signin-wait.png, overlay unchanged).
- **2026-08-26** — MCP Sign in waits for the browser callback (no paste by default). Paste address is fallback. visual-review: PASS (mcp-signin-wait.png).
- **2026-08-26** — MCP Sign in uses a short pi when no agent is running. Add/On on OAuth starts Sign in. visual-review: PASS (mcp-signin-short.png). Dogfood: Notion login opened with no agent.
- **2026-08-26** — Sign in is a button next to On (not a SIGN-IN tag). Off has no Sign in. visual-review: PASS (mcp-signin-btn.png).
- **2026-08-26** — MCP Sign in starts `/mcp-auth` (RPC + paste dialog). visual-review: PASS (mcp-signin.png).
- **2026-08-26** — MCP GET redacts env/header values. Dogfood in Codex/Grok left for later.
- **2026-08-26** — Diff cards in conversation use JetBrains Mono + Fira Code (same as source fences). visual-review: PASS (chat-diff-font.png).
- **2026-08-26** — Remove on a Use-from overlay no longer unmasks the import (stays Off). Dogfood Claude servers deleted. List is A–Z. visual-review: PASS (mcp-list-az.png).
- **2026-08-26** — MCP list is A–Z by name (live poll no longer reshuffles).
- **2026-08-26** — Conversation source uses JetBrains Mono + Fira Code ligatures. visual-review: PASS (chat-code-font.png).
- **2026-08-26** — MCP live status (Idle / Live / Failed / Sign in). visual-review: PASS (mcp-live-idle.png).
- **2026-08-26** — MCP Add More: env / headers / Sign in / Token. visual-review: PASS (mcp-add-more-url.png, mcp-add-more-env.png, mcp-add-more-error.png).
- **2026-08-26** — MCP card: agent icon + name at top; scope pill is This agent again. visual-review: PASS (mcp-this-agent.png).
- **2026-08-26** — Use from is a tree (app → servers). Pick per server; Off the rest. visual-review: PASS (mcp-use-from-tree.png).
- **2026-08-26** — Dogfood MCP in Claude/Codex/Grok globals (`picode-dogfood-*`). Use from lists counts. visual-review: PASS (mcp-use-from-dogfood.png).
- **2026-08-26** — Import renamed **Use from…** (mirror, not copy). Empty hosts hidden. visual-review: PASS (mcp-use-from.png).
- **2026-08-26** — MCP Import is a picker, not import-all. visual-review: PASS (mcp-import-pick.png).
- **2026-08-26** — MCP B3 Import (adapter `imports` only). visual-review: PASS (mcp-import.png).
- **2026-08-26** — Agent context is the first line in the MCP/Packages card, not under the title. visual-review: PASS (mcp-card-ctx.png).
- **2026-08-26** — MCP/Packages name the agent (title + pills). Sidebar click from a pane opens that agent. visual-review: PASS (mcp-named.png).
- **2026-08-26** — MCP empty redesigned (one line + Open packages). UI skills now load-before-JSX; visual skip = quality-gate FAIL. visual-review: PASS (mcp-blocked.png).
- **2026-08-26** — MCP manager: list/add/toggle/remove on adapter files (machine / folder / this agent). B3 import next.
- **2026-08-26** — Composer `!cmd`: RPC bash in the agent folder, inline block + Stop. Track A done.
- **2026-08-26** — Toolbar clip attaches workspace files (image → chip, else `@path`); reads stay inside the folder.
- **2026-08-26** — Click a composer/chat thumbnail to preview the image.
- **2026-08-26** — Composer image chip 64px; `@` list has a filter and hides dotfiles until typed.
- **2026-08-26** — Composer paste/drop images (RPC `images[]`). Next: `!`.
- **2026-08-26** — Composer `@` file picker (agent cwd). Next: images, then `!`.
- **2026-08-26** — Roadmap: composer files then MCP (`docs/design/composer-mcp-roadmap.md`). Auth/llama parked.
- **2026-08-26** — Restore walks the same job overlay (stop agents → db → pins → sessions) and asks to reload.
- **2026-08-26** — Reveal uses host Explorer on WSL. Backup job steps animate. Motion + optimistic UI is a gate.
- **2026-08-26** — Backup schedule is explicit (off until Schedule). Preferences split into tabs.
- **2026-08-26** — Folder picker on WSL: Home / C: / E: chips; accepts `C:\\` paths.
- **2026-08-26** — Backup V1: Preferences folder + interval/retention. `VACUUM INTO` + hardlink snapshots. Restore refuses newer schema.
- **2026-08-26** — Decision table is a quality gate when conditions change the outcome (AGENTS.md).
- **2026-08-26** — Delete agent/workspace: confirm may offer session + work-folder purge (last occupant only). All workspace agents stopped first.
- **2026-08-26** — Pin V3 sketches (Excalidraw, lazy). Blank or annotate image.
- **2026-08-26** — Pin V2.1 TipTap editor (markdown on disk).
- **2026-08-26** — Pin attachments V2 (image + file). Sketch/Excalidraw is V3.
- **2026-08-26** — Pin studio is a route (`#/pins/new` / `#/pins/:id`). List stays in the sidebar.
- **2026-08-26** — `npm:pi-agent-browser-native` + skill shrunk. Fix: IconPin crash (blank app).
- **2026-08-26** — Pins V1: title, tags, markdown body. Flat list, `+` on title bar.
- **2026-08-26** — Sidebar tabs Agents / Pins. QR → user menu.
- **2026-08-26** — Conversation polish: blockquotes + ```diff``` hunks. Images + Mermaid + KaTeX + tables.
- **2026-08-26** — Source **Run** (bash/python/js/go) in the agent cwd. Not a browser sandbox.
- **2026-08-26** — Conversation source renderer (fenced code: lang + copy + highlight).
- **2026-08-26** — Codex DID reply; chat ignored `message_end` (no `text_delta`). Free-agent Sessions listed the wrong folder (0).
- **2026-08-25** — Chose `npm:pi-web-search` (this machine). Chat search cards from tool sources. Full packages-cycle dogfood deferred.
- **2026-08-25** — Packages **This agent** + optional isolate (skip machine/folder). `pi -e` every start / every session.
- **2026-08-25** — llama GUI installer reverted. Setup stays in `www/guide/llama.md`; dialog is URL + link. Continuity → backlog.
- **2026-08-25** — `/llama` dialog on the agent (not Providers redirect). HF download, wait-for-load, default `127.0.0.1:8080`.
- **2026-08-25** — Slash TUI 24 all **ui**. Skills/templates picker. `/export` `/import` `/share` (gist) `/hotkeys` `/changelog`.
- **2026-08-25** — ADR-0013 multi-account vault. OAuth re-login updates same account; click name to rename.
- **2026-08-25** — Device-code OAuth: Copilot / Kimi / xAI. Claude + Codex stay loopback.
- **2026-08-25** — SW never caches `index.html`. Sidebar tree: 12px indent, shared chevron|icon|label grid.
- **2026-08-25** — Providers GUI: no docs copy (guide is public). Voice V1 shipped; owner dogfooding.
- **2026-08-25** — Relicensed PolyForm Noncommercial. Public docs VitePress on Pages (no in-app iframe).

## Recent activity (archived 2026-08-25)

- **2026-08-25** — Public docs: VitePress Markdown, new-tab slash hints
  (`/commands#{id}`). No in-app docs/iframe.
- **2026-08-25** — Public docs (`www/` → GitHub Pages). `#/docs/{cmd}`
  iframes them (later removed). `/tree` click remains fork (pi#8645).
- **2026-08-25** — ADR-0011: sidebar **Free** vs **Workspaces**, many agents
  per folder (own model). Selected entity is the agent id.
- **2026-08-25** — Packages: This machine vs This workspace (`-l`).
  Session/`pi -e` still deferred (This run). ADR-0010 amended.
- **2026-08-25** — Optimistic UI is a bar (`docs/philosophy.md` §7).
  Packages gallery uses layout skeletons on first load; refetch keeps
  last hits. Blank wells while fetching are FAIL.
- **2026-08-25** — Voice V1: dictation + Grok-style voice composer
  (`docs/design/voice-mode.md`). Web Speech API, no Realtime fork.
- **2026-08-24** — Desktop/mobile shells in one Vite app (`web/src/desktop`,
  `web/src/mobile`). Boot picker by viewport or `?desktop=1`/`?mobile=1`.
- **2026-08-24** — Phone QR: prefer current LAN IP. Drawer lists lan/tailnet
  targets; QR only for addresses on the cert.
- **2026-08-24** — Adopted AgentDeck's product-benchmark set: Cursor +
  t3code + paseo. Studies in `docs/benchmarks/`.
- **2026-08-24** — Route split: Settings = PiCode system; `#/providers`
  and `#/mcps` are first pi-facing routes.
- **2026-08-24** — Agent provider/model/thinking moved onto the agent tab
  bar (auto-save).
- **2026-08-24** — **ADR-0009 + M3 v1**: catalog from pi, auth via `/login`
  in the TUI, MCP status-only, agent config flags on start, exclusive lock.
- **2026-08-24** — M2 closed: inline diffs and Ctrl+K palette. Accept/reject
  hunks deferred.
- **2026-08-24** — **ADR-0008**: UI React + Vite + Tailwind. Source in `web/`.
- **2026-08-24** — Dock: `[hidden]{display:none !important}`; single pane
  owned by the active agent tab (no inner tab strip).
- **2026-08-24** — IDE-style agent tabs; dock opens only by explicit action.
- **2026-08-24** — Exploratory QA (agent-browser): real-pi prompt stream,
  port rebind 8445→8446→8445, theme sweep.
- **2026-08-24** — `agent-browser` skill added (agentdeck port).
- **2026-08-24** — User-menu popover SyntaxError; Cache-Control no-cache +
  `cmd/uicheck`. JS-syntax gate mandatory after app.js edits.
- **2026-08-24** — ADR-0007 shipped: HTTPS default, port rebind, server.json.
- **2026-08-24** — UI redesign after owner feedback (conversation-hero,
  tool pills, rounded composer, terminal dock).
- **2026-08-24** — M2 core shipped (ADR-0006): rpc + runtime + delivery,
  mode-switch, /ws/agent, agent panel. Verified against real pi.
- **2026-08-24** — ADR-0005 shipped: SQLite store, schema v1, migrations,
  legacy JSON import.
- **2026-08-23** — UI copy de-documentarized, Vercel-style user menu,
  settings route, live statusbar. M1 visually validated by owner (PASS).
- **2026-08-23** — CI `-race` term-bridge shutdown race; single-owner pty.
- **2026-08-23** — M1 complete: screenshot tooling, tmux, WS↔PTY, terminal
  grid, ADR-0004.
- **2026-08-23** — Language policy: English is the repository language.
