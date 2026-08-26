# Handoff archive

Moved off `docs/handoff.md` when it exceeded ~150 lines. Newest living
state is always `docs/handoff.md`. Do not treat this file as current.

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
