# Handoff — living project state

> Heartbeat of PiCode. Session that changes state **must** leave this file
> matching **HEAD** (ritual: `/skill:handoff-update`, contract: AGENTS.md).
> Stale *Next up* (listing shipped work) is FAIL. Newest *Recent activity*
> first. Archive to `docs/handoff-archive.md` when this file exceeds ~150 lines.

## Current state (read this first)

**Visual gate:** UI work must `read` a screenshot, run
`window.__picodeOverlayAudit()`, and answer the visual-card. Clip = FAIL.

**Phase:** ADE past M3. Slash TUI **24 ui · 0 missing**. Providers + vault.
Public docs on Pages. llama.cpp manager is **docs + dialog**, not an installer.

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

## In flight

Sidebar chrome: Agents / Pins tabs shipped. Pin editor (Tachyon-inspired rich form) is **next**. llama.cpp depth is backlog.

## Next up

1. Voice V1 dogfood (owner, Chrome Windows mic). V1.1 cloud STT only after that (`docs/design/voice-mode.md` §4).
2. MCP visual manager — **blocked** until adapter format is decided.
3. Packages cycle dogfood (owner; deferred).

## Backlog

- llama.cpp: in-app installer / start router, SSE progress + cancel, delete `.gguf`, Ollama/vLLM (`models.json`).
- Mobile parity (shell exists; not feature-complete).
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.

## Known debts / open questions

- Two concurrent agents share whichever credential is **active** in `auth.json` (pi limitation; vault does not fork that).
- Token auth: ADR-0007 personal-network trust; mandatory only if exposed beyond the tailnet.
- `internal/proclock` leftover `picode.lock` after a Windows crash.
- Vendored xterm.js 5.5.0 — manual upgrade (ADR-0004).
- Branch protection + CODEOWNERS — owner action on GitHub.
- tmux-gated tests skip on windows/macos CI (accepted).

## Recent activity

- **2026-08-26** — Sidebar tabs Agents / Pins. QR → user menu. Pin form next.
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
