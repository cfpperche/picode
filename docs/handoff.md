# Handoff — living project state

> Heartbeat of PiCode. Session that changes state **must** leave this file
> matching **HEAD** (ritual: `/skill:handoff-update`, contract: AGENTS.md).
> Stale *Next up* (listing shipped work) is FAIL. Newest *Recent activity*
> first. Archive to `docs/handoff-archive.md` when this file exceeds ~150 lines.

## Current state (read this first)

**Visual gate:** `read` uiux-review **before** JSX. Empty/blocked/error
screenshots must be `read`. overlayAudit + visual-card. Clip = FAIL.
Skip = quality-gate FAIL.

**Phase:** ADE past M3. Slash TUI **24 ui · 0 missing**. Providers + vault.
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
- Voice **V1 shipped** (dictation + Grok composer + browser TTS). Owner dogfood pending (Chrome Windows mic).
- Public docs: VitePress `www/` → GitHub Pages. GUI chrome carries **state**, not docs. ADRs 0001–0013.
- `#/mcps` missing adapter: one line + Open packages (`www/guide/mcp.md`). No npm/architecture in the view.
- MCP / Packages / Settings name the selected agent (icon + name) as the first line in the card. Scope pills say **This agent**. Sidebar agent click from a pane goes to `#/`.
- MCP **Use from…** mirrors other apps. User picks; Off hides a server. Empty host files are not offered.
- MCP Add **More**: env on command servers; headers + Sign in / Token on URL servers.
- MCP list live state: Idle / Live / Failed / Sign in when the GUI agent is running. File Off has no word.

## In flight

Backup V1 shipped (local directory). Remote S3/Drive is later. llama.cpp depth is backlog.

## Next up

1. Voice V1 dogfood (owner, Chrome Windows mic). Auth / llama / Radius after.

## Backlog

- llama.cpp: in-app installer / start router, SSE progress + cancel, delete `.gguf`, Ollama/vLLM (`models.json`).
- Mobile parity (shell exists; not feature-complete).
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.

## Known debts / open questions

- Dogfood MCP entries `picode-dogfood-*` in `~/.claude/mcp.json`, `~/.codex/config.toml`, `~/.grok/config.toml` — remove after the Use-from check.
- MCP GET still returns env/header values from the file (plaintext). `bearerToken` is write-only and not listed.
- MCP **installed + zero servers** Add form is live (`docs/screenshots/mcp-named.png`). Sidebar-on-pane → `#/` checked in browser, not unit-tested.
- Two concurrent agents share whichever credential is **active** in `auth.json` (pi limitation; vault does not fork that).
- Token auth: ADR-0007 personal-network trust; mandatory only if exposed beyond the tailnet.
- `internal/proclock` leftover `picode.lock` after a Windows crash.
- Vendored xterm.js 5.5.0 — manual upgrade (ADR-0004).
- Branch protection + CODEOWNERS — owner action on GitHub.
- tmux-gated tests skip on windows/macos CI (accepted).

## Recent activity

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

Older blocks: [handoff-archive.md](handoff-archive.md).
