# Handoff archive

Moved off `docs/handoff.md` when it exceeded ~150 lines. Newest living
state is always `docs/handoff.md`. Do not treat this file as current.

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
