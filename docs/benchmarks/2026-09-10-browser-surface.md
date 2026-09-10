# Study: Embedded browser surface — preview to design mode

- **Date:** 2026-09-10
- **Sources:** Lovable docs
  ([docs.lovable.dev/features/preview-toolbar.md](https://docs.lovable.dev/features/preview-toolbar.md),
  fetched live 2026-09-10), Replit docs
  ([docs.replit.com/features/editor/preview.md](https://docs.replit.com/features/editor/preview.md),
  Mintlify markdown), bolt.diy source (stackblitz-labs/bolt.diy @ main:
  `app/components/workbench/Inspector.tsx`, `public/inspector-script.js`,
  `Preview.tsx` — read from raw.githubusercontent.com), Orca docs
  (github.com/stablyai/orca @ main: `docs/site/content/docs/browser/overview.mdx`,
  `design-mode.mdx`, plus onorca.dev landing), vercel-labs/agent-browser
  streaming docs (`docs/src/app/streaming/page.mdx` @ main), and the prior
  in-repo study `docs/benchmarks/2026-09-02-live-browser-preview.md`
  (Cursor/Devin/Manus/Operator/Antigravity/Browserbase/browser-use). All pages
  fetched live during this study; closed-source UI internals are marked
  *inference*.
- **Scope:** the owner's question — how would a browser running *inside the
  PiCode URL* work, evolving into a Lovable/Replit/Orca-style design mode —
  and which surface shape the first phase should take.

## The convergence: two render channels, one interaction grammar

| Product | Preview channel | Design-mode grammar |
|---|---|---|
| **Lovable** | iframe of the hosted app | Preview toolbar with **modes**: Select elements (multi-select → chat attachment), Edit text inline, Draw annotation, pinned Comments; `S`/`T`/`D`/`C` shortcuts |
| **Replit** | iframe of the dev server on `*.replit.dev`, viewport presets, devtools | No click-to-context yet; "Replit Design" carries design systems + Figma import as chat context |
| **bolt.diy** (open source) | iframe (WebContainer-served app) | Injected `inspector-script.js`: hover → highlight; click → `postMessage` `ElementInfo {tagName, id, classes, relevant computed styles, rect}` → chat attachment |
| **Orca** | **Real Chromium per worktree** (Electron-embedded), CDP viewport emulation | **Design Mode**: click → captures outerHTML + small neighborhood, computed CSS, cropped screenshot, file/line via dev source maps → one chat attachment; loop = agent edits, hot-reload, click again |
| **Cursor/Devin/Browserbase/browser-use** | CDP screencast over WebSocket (see 2026-09-02 study) | Take-over / live view; element context via the agent's own tooling |

Read across: the **iframe channel** (Lovable, Replit, bolt) wins latency and
works only for the project's own dev server; the **screencast channel**
(Orca-remote, Devin, Browserbase) wins universality — any URL, one browser for
human and agent. The interaction grammar is identical either way: *pick an
element → HTML + computed CSS + cropped screenshot become one chat attachment
→ the agent edits → hot-reload → verify by clicking again.*

Orca is the closest analog to PiCode (an ADE) and the design-mode reference:
its automation CLI drives **the same browser** the human watches
(`orca snapshot/click/fill`). PiCode already has that property — the agent's
`agent_browser` session — and has never surfaced it.

## What the engine already gives us

`agent-browser` 0.35+ (installed: 0.37.0) starts, **per session, a loopback
WebSocket** that (streaming doc, upstream main):

- streams `{"type":"frame","seq",data(base64 JPEG),metadata:{deviceWidth,
  deviceHeight,pageScaleFactor,scrollOffset…}}`, latest-wins, no backlog;
- sends `url`/`status`/`console`/`tabs` messages on an ordered channel;
- accepts per-client `config {maxFps, pacing}` and cumulative `ack {seq}`;
- **accepts input** (`input_mouse`, keyboard, touch) — input resets the idle
  timer and dispatches immediately, independently of frame delivery;
- rejects non-localhost browser origins (403 on upgrade) — by design; a
  remote viewer needs a proxy.

The pi-browser-capture sidecar (ADR-0082) already proved the Go-side
discovery route: `get_state` → `sessionId`+`sessionFile` → implicit session
name → socket-root `<name>.stream` rendezvous with uid/0700/realpath/
O_NOFOLLOW/liveness checks.

## What PiCode adapts

| Benchmark pattern | PiCode adaptation |
|---|---|
| One browser for human and agent (Orca) | Surface attaches to the agent's existing `agent_browser` session — never a second browser (ADR-0003) |
| Frames behind an authenticated proxy (Browserbase "live-view URL is a credential") | `GET /ws/browser?agent=<id>` inside the daemon's one auth gate; loopback-only engine stays loopback-only (ADR-0114) |
| Toolbar modes (Lovable), pointer-to-code (Orca) | Phase 3: canvas click → `elementFromPoint` eval via the engine → HTML/CSS/cropped screenshot → one chat attachment; source-file/line via dev source maps later |
| Element info shape (bolt.diy `ElementInfo`) | Same fields, produced engine-side instead of iframe-side |
| iframe + injected inspector (Lovable/bolt) | Possible later *complement* for the dev-server case (zero-latency); not the base — blind to external URLs and mobile |

**Answer to "is the panel an app?"** No — an ADR-0036 app is manifest +
primitives and cannot touch the DOM; a live canvas is shell material. The
surface is a native view of the agent tab (the Chat | TUI pattern gains a
third state), the same way Terminal is not an app. ADR-0114 records the
decision; the capacity stays in pi packages (ADR-0003/0082).

## Refusals

| Temptation | Why not |
|---|---|
| chromiumedp/headless browser owned by PiCode | Forks the ecosystem engine; breaks "same browser" (ADR-0003) |
| Frames on the events feed | Durable audit vs ephemeral bulk (2026-09-02 refusal stands) |
| Input injection in phase 1 | Interactive control of a browser that may hold the user's sessions deserves its own consent decision (phase 2) |
| VNC/noVNC | Second transport, worse fidelity (2026-09-02 refusal stands) |

## Open questions for implementation

1. **Phase-2 consent** — input injection gate: reuse the `/browser-captures`
   consent pattern or a new one?
2. **Multi-viewer pacing** — one proxy fan-out or per-client engine
   connections? (Engine is multi-client with per-connection `maxFps`; proxy
   fan-out of one connection is simpler but couples clients' pacing.)
3. **Mobile** — stream view on the phone (tap → `input_mouse`/touch works
   engine-side) vs capture pill only.
4. **Design-mode attachment transport** — workspace file + prefill, or a
   dedicated attachment envelope on the enqueue path.
