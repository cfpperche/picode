# Study: Live browser preview of agent work in chat / side panel

- **Date:** 2026-09-02
- **Sources:** Cursor Browser docs
  ([cursor.com/docs/agent/browser](https://cursor.com/docs/agent/browser), fetched
  2026-09-02), Cursor 2.0 announcement
  ([cursor.com/blog/2-0](https://cursor.com/blog/2-0)), Devin Session Tools and
  Computer Use ([docs.devin.ai/work-with-devin/devin-session-tools.md](https://docs.devin.ai/work-with-devin/devin-session-tools.md),
  [computer-use.md](https://docs.devin.ai/work-with-devin/computer-use.md)), Manus Cloud
  Browser ([manus.im/docs/features/cloud-browser.md](https://manus.im/docs/features/cloud-browser.md)),
  Google Antigravity Browser + Agent Side Panel
  ([antigravity.google/docs/ide/browser](https://antigravity.google/docs/ide/browser),
  [agent-side-panel](https://antigravity.google/docs/ide/agent-side-panel)), Browserbase
  Session Live View
  ([docs.browserbase.com → observability/session-live-view](https://docs.browserbase.com/platform/browser/observability/session-live-view)),
  browser-use Live Preview
  ([docs.browseruse.com/cloud/browser/live-preview](https://docs.browser-use.com/cloud/browser/live-preview)),
  OpenHands SDK browser guides
  ([agent-browser-use.md](https://docs.openhands.dev/sdk/guides/agent-browser-use.md),
  [browser-session-recording.md](https://docs.openhands.dev/sdk/guides/browser-session-recording.md)),
  OpenAI Operator (openai.com blocked the automated fetch; facts from
  [Wikipedia](https://en.wikipedia.org/wiki/OpenAI_Operator) citing WSJ/Wired/NYT —
  *inference* on exact UI mechanics), and the installed local stack:
  `agent-browser` 0.34.0 (`stream enable [--port]`, artifacts + `artifactVerification`,
  `record start/stop`) wrapped by `pi-agent-browser-native` 0.5.0 (a **pi package**,
  tool `agent_browser`). All pages fetched live with `agent_browser`; closed-source
  UI behavior is marked *inference*.
- **Scope:** what shape a PiCode **live browser preview** should take — the agent is
  using `agent_browser` and the GUI should show what it is doing, in the conversation
  and/or as a side panel — plus the packaging question: does this belong in PiCode
  core, or can a pi package extend the conversation?

## The convergence (and it is total)

Every product that lets an agent drive a browser shows the human **two layers**, and
names them almost identically:

1. **The activity layer in the chat** — screenshots and the action taken, inline where
   the tool call happened.
2. **A live surface** — the browser itself (or a video/DOM stream of it), docked
   beside the conversation, with **take-over** for human-in-the-loop.

| Product | Activity in chat | Live surface | Take-over / HITL | Auth & state | Gate |
|---|---|---|---|---|---|
| **Cursor 2.0+** | "screenshots and actions in the chat" (docs) | "the browser window itself either in a separate window or an **inline pane**" | Tool **approval modes**: manual review of each browser action (default), configurable in Agent Settings; enterprise MCP allow/denylist | Session persists **per workspace** (cookies, localStorage, IndexedDB), isolated per project | Docs quote, shipped UI (*inference* on pane internals) |
| **Devin** | **Progress tab**: "all shell commands, code edits, and browser activity … in one unified view" | **Browser tab** in the session (alongside Shell, IDE); Computer Use streams a 1024×768 desktop | "take over Devin's work" — Interactive Browser for logins, MFA, CAPTCHA; **hand back** to Devin; side chats stay read-only | "Save the browser profile" → zipped into the org **blueprint**; every future session starts authenticated | Public docs |
| **Manus** | Task timeline shows steps; screenshots surface in the transcript | **Cloud Browser** runs in the task view — "You see everything Manus is doing in real-time" | **"Take Over"** prompt on SMS/CAPTCHA/MFA; user completes, then "hand control back to Manus" | User logs into personal accounts inside Cloud Browser; sessions persist | Public docs |
| **OpenAI Operator** (2025, deprecated → ChatGPT agent) | Screenshots + narrated steps in the thread | Dedicated browser rendered **in a side panel** next to the chat | Watch mode → **"Take control"** toggle; agent pauses on sensitive-action guardrails (*inference*; page blocked to automation) | User-typed credentials stay in-session; watchlists for sensitive sites | Press/Wikipedia; UI *inference* |
| **Google Antigravity** | Browser Subagent "capturing screenshots and saving **action videos as interactive artifacts**" (artifact = first-class reviewable object) | Browser tooling over **local Chrome**; `/browser` slash command; agent side panel tracks "file changes, running terminal processes, and artifacts" | **Allowlist / Denylist** two-layer URL gate; isolated Chrome profile; Browser Tools kill switch in settings | Separate Chrome profile for the agent; recordings as artifacts | Public docs |
| **Browserbase** | — (platform, not chat) | **Session Live View**: "watch, click, type, and scroll in real-time"; embeddable `<iframe>`; **read-only** variant via `pointer-events:none`; per-tab live-view URLs | Human-in-the-loop is a listed use ("instantly take control"); `browserbase-disconnected` message event for session end | Live-view URL is a credential; sessions are cloud-side | Public docs |
| **browser-use (cloud)** | Run events stream (V4 API) | `browser.ready` → `live_view_url`, embeddable iframe (`live.browser-use.com`); **"Treat the URL as a credential: anyone with it can interact"**; MP4 recording after stop | Interaction through the same iframe | Profiles per end user; ZDR projects never record | Public docs |
| **OpenHands SDK** | Tool results of `BrowserToolSet` (built on the `browser-use` library) | — (no live pane; headless) | — | — | **rrweb recording**: `browser_start_recording` captures DOM mutations/mouse/scroll → JSON, replayed in rrweb-player | Public docs |
| **agent-browser** (the engine PiCode already ships) | Tool result text: refs, `pageChangeSummary`, `artifacts` + `artifactVerification` | **`stream enable [--port]`**: runtime **WebSocket** streaming for the session; `record start/stop` video; `dashboard start/stop` | — (headless/isolated Chromium; Electron side) | Profile/restore keys per git-checkout generation (managed-session restore) | Local CLI docs, v0.34.0 |

Read across the table: **the chat card and the live pane are the same feature seen
from two distances.** The chat answers "what did it just do"; the pane answers "what
is it looking at right now". Nobody ships only one of the two.

## Patterns worth stealing

| Pattern | Who | PiCode adaptation |
|---|---|---|
| Screenshots + action inline where the tool ran | Cursor, Antigravity, Devin Progress | Render the `agent_browser` tool pill with its **latest screenshot** and a one-line action label instead of raw JSON; history stays in the pill (expand) |
| Live window inline or docked | Cursor ("separate window or inline pane"), Manus/Operator (side panel) | A **Preview side panel** on `#/agent/<id>` (Chat \| Terminal \| **Browser**), fed by frames; opens when a browser tool starts, closes when the turn ends without one |
| Live-view URL is a credential | Browserbase, browser-use | If PiCode proxies frames, the proxy route rides the existing `internal/auth` gate — no unauthenticated stream port reaches the browser |
| Take-over (MFA/CAPTCHA hand-off) | Manus, Devin, Operator | v2: a "Take over" affordance = open the URL in the user's own browser / or headful `--headed` session later; not v1 |
| URL allowlist/denylist + isolated profile | Antigravity | Already true in PiCode: `agent-browser` runs an isolated Chromium; policy lives in the package, not the GUI |
| Recording as artifact, not stream | Antigravity (action videos), OpenHands (rrweb), browser-use (MP4) | `record start/stop` + artifacts serve the **post-hoc** need (audit, "what did it do"); the live pane only needs to exist **during** the run |
| Log-to-file, read-on-demand | Cursor ("browser logs written to files that Agent can grep") | Token discipline is the package's job — `agent-browser` already spills large outputs to files with `artifactVerification`; keep verbose payloads out of `details` |

## The two render channels, technically

Live browser-to-web preview has exactly three implementations in the wild:

1. **CDP screencast frames** (`Page.startScreencast`) pushed over a WebSocket — what
   embedded live views (Browserbase/browser-use) serve behind their iframe; what
   `agent-browser stream enable` exposes locally. Cost: continuous frames, but
   framerate is driven by page damage (idle page = idle stream).
2. **DOM-level recording** (rrweb) — OpenHands' choice: smaller payloads, perfect for
   replay, wrong for "see it now" on pages with canvas/video.
3. **Screenshot polling** — the degenerate but robust fallback: last screenshot as a
   poster frame, refreshed per tool step. This is also exactly what Cursor's chat
   layer shows.

PiCode's pipeline today: the pi process streams `tool_execution_start / update / end`
as JSONL over RPC → `internal/rpc` fans them out → `GET /ws/agent` → the reducer in
`web/src/lib/agentEvents.js`. **`tool_execution_update` is not consumed by the web
reducer**, and the generic `Tool` pill renders `detail` as a `JSON.stringify` string —
no image rendering anywhere in tool results. That is the only real gap; everything
upstream of the GUI already flows.

## What PiCode adapts (proposal shape, pre-ADR)

| Layer | Change | Why it belongs there |
|---|---|---|
| **pi package** (`pi-agent-browser-native`, or a thin companion) | On browser-tool activity, emit the preview contract: `details.preview = { image (data URI), url, title, step }` in `partialResult` on each meaningful step (screenshots it already takes; opt-in cadence) | The package owns browser semantics; PiCode core must never learn the string "agent_browser" (ADR-0003 discipline: we orchestrate the ecosystem, we don't fork it) |
| **PiCode core — conversation** | Generic, tool-agnostic rendering rule: a tool pill whose result/updates carry `details.preview` renders the image + title instead of raw JSON; reducer consumes `tool_execution_update` | This is a *rendering contract*, the same move as ADR-0036's app primitives: the host renders shapes, the ecosystem supplies content. Any future tool (API testers, dashboards) inherits it for free |
| **PiCode core — side panel** | `#/agent/<id>` gains a **Browser** surface (Chat \| Terminal \| Browser tabs exist as a pattern); panel shows the newest preview frame + URL + step log; opens automatically when a preview-active tool starts, collapses on settle | One control height, empty state = "No browser activity in this turn"; follows the motion rule (panel exists the moment work starts) |
| **v2 — true live stream** | The Go server proxies the package's `stream enable` WebSocket behind `internal/auth` (`GET /api/agents/{id}/preview?tool=…`), so frame-rate video reaches the panel without exposing a local port | Auth gate is server-side and already exists; the proxy is generic ("tool streams"), not browser-specific |

**Answer to the packaging question:** no new PiCode package system is needed — pi's
already exists and PiCode already manages it (`#/packages`, ADR-0010: machine /
workspace / this-agent scope, `-e` ephemeral). What is missing is the **last mile of
the existing pipe**: a tool→GUI preview contract that *any* pi package can emit and
PiCode's conversation renders. Build the contract, not a browser feature. If
`pi-agent-browser-native` upstream won't carry the `details.preview` emission, ship it
as a small companion pi package (`pi-agent-browser-view`) that wraps/observes the same
tool events — still zero browser code in PiCode core.

## What PiCode refuses

| Temptation | Why not |
|---|---|
| `agent_browser` special-cased in PiCode core UI | Tool-name coupling in the host is the fork we swore off (ADR-0003, MCP section of architecture.md); the contract must stay tool-agnostic |
| VNC / noVNC into the isolated Chromium | Second transport to secure, worse quality than CDP screencast, no read-only mode; Browserbase/browser-use both standardized on the embeddable frame stream instead |
| Writing preview frames into the `events` feed (ADR-0048) | The feed is durable audit; frames are ephemeral bulk. Ephemeral agent state (streaming, waiting, `agent.tui`) already rides id-0 events — a binary frame does not belong there; the `/ws/agent` pipe or a dedicated WS proxy is the honest channel |
| Auto-taking screenshots the agent didn't ask for | Token/cost discipline is upstream's: extra `screenshot` calls burn context and time. v1 renders screenshots the flow already produces; forced cadence is a setting on the package, not a GUI demand |
| Take-over controls inside the panel (v1) | Input injection from the GUI into the tool's session duplicates what `--headed` + the user's own browser already solve; Devin/Manus needed it for cloud browsers — ours runs on the user's machine |

## Open questions for the ADR

1. **Contract placement** — does `details.preview` get defined PiCode-side
   (documented convention) or proposed upstream to pi (so every pi host benefits)?
   Tilt: propose upstream; the RPC protocol is pi's, and `setWidget` shows precedent
   for extension→host UI data.
2. **Frame budget in V1** — screenshots per tool step through `partialResult` are
   ~100–500 KB each as PNG data URIs; throttle to one live frame per tool call plus
   on-navigation, and keep full resolution in the expandable history only on demand.
3. **Panel placement on mobile** — `#/agent/<id>` on the phone is single-surface
   (Chat \| Terminal); a third segment or a collapsed strip? Termux-style: poster
   strip + tap to open full-screen.
4. **`stream enable` lifecycle** — who calls disable (the package, on settle), and
   what the panel shows when the stream port dies (poster frame + "stream ended").
