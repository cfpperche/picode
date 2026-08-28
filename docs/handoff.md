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

## In flight

Track E (E1–E4) shipped. Diff/editor roadmap complete.
Backup V1 shipped (local directory). Remote S3/Drive is later.

## Next up

No coding track in flight. Backlog: llama installer, mobile parity, `/tree` in-place (pi#8645), worktrees.

## Backlog

- llama.cpp: in-app installer / start router, SSE progress + cancel, delete `.gguf`, Ollama/vLLM (`models.json`).
- Mobile parity (shell exists; not feature-complete).
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.
- Worktrees / parallel isolated agents (Orca + Herdr) — after Track E.

## Known debts / open questions

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

## Recent activity

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

Older blocks: [handoff-archive.md](handoff-archive.md).
