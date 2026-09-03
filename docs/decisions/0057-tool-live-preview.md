# ADR-0057: Tool live preview in the conversation

- **Status**: accepted
- **Date**: 2026-09-02
- **Study**: [docs/benchmarks/2026-09-02-live-browser-preview.md](../benchmarks/2026-09-02-live-browser-preview.md)

## Context

When an agent drives a browser through `agent_browser` (the
`pi-agent-browser-native` pi package), the user sees a generic tool pill with
raw JSON — nothing shows what the agent is looking at. The benchmark study
found total convergence in peer products (Cursor, Devin, Manus, Operator,
Antigravity): an **activity layer in the chat** (screenshot + action, where
the tool ran) plus a **live surface** beside it, with human take-over.

On the technical side, PiCode's pipe already carries everything needed:
`tool_execution_update` events flow pi → RPC → `GET /ws/agent`, and the
`details` object on tool results and partial results is free-form JSON that
reaches the web reducer untouched — including on replay, because
`internal/session/transcript.go` keeps `details` in the persisted toolResult
message. What is missing is entirely on the rendering side: the web reducer
never consumes `tool_execution_update`, and no component renders images from
tool details.

The packaging question resolves the same way: pi's package system (managed in
PiCode by ADR-0010) already exists, so no second package system is needed.
The missing piece is a **rendering contract** any tool package can emit and
the conversation renders — the same host-renders-shapes move as the app
primitives (ADR-0036). PiCode core must never learn the string
`agent_browser` (ADR-0003: we orchestrate the ecosystem, we do not fork it).

## Decision

**PiCode defines a tool-agnostic live-preview contract and renders it in the
conversation.** A tool may surface what it is looking at by emitting, inside
its result or partial-result `details`:

```json
{ "preview": { "image": "<data-uri or https URL>", "url": "https://…", "title": "…", "ts": 0 } }
```

`image` is required; `url`, `title`, `ts` are optional. The contract rides
existing channels only:

1. **Live, during execution** — `partialResult.details.preview` on
   `tool_execution_update`. Frames replace each other (latest wins, no
   accumulation). One frame per meaningful step, not a video.
2. **Persisted** — the final tool result carries the last frame in
   `details.preview`, so reload/replay (which reads tool results, not
   partials) still renders it.
3. **Renderer** — the tool pill shows the frame inline (always visible,
   click → existing lightbox) with the page title and URL as caption. The
   reducer consumes `tool_execution_update` in both event paths (desktop
   `handleEvent` and the shared `agentEvents.js` reducer) and replay
   constructs the same item shape.

V1 ships the contract + pill rendering. A dedicated **Browser** panel
surface and a v2 **frame stream** (proxying the engine's `stream enable`
WebSocket behind `internal/auth`) are follow-ups, not part of this decision's
scope.

## Consequences

**Easier:** any pi package gets a live surface for free (API testers,
dashboards, image tools) with zero PiCode-core knowledge of any tool name.
The chat gains the activity layer every peer product has. Frames-per-step is
honest about what "live" means here: the RPC pipe only flows while a tool
executes — and that is exactly when the page changes. Between calls the last
frame is the truth, not a stale image.

**Harder / accepted costs:**

- Frame budget rides on the package: large data URIs through the JSONL RPC
  pipe cost memory. Emission should downscale (JPEG, ~≤200 KB) and throttle
  to one frame per step plus navigations. The renderer replaces, never
  accumulates, so a chatty emitter cannot grow the thread.
- "Live" is step-granular, not video. Users expecting a streaming pane must
  wait for the v2 proxy — recorded in the study, not hidden.
- The desktop `handleEvent` and `agentEvents.js` must stay in lockstep (they
  already must; this adds one more case to both).

**Who breaks if we are wrong:** if a tool emits huge or hostile previews,
the thread slows — the renderer's replace-semantics and the emitter-side
downscale are the guards; a renderer-side size cap (refuse `image` above a
bound) can be added without breaking the contract. If upstream
`pi-agent-browser-native` ships its own richer convention, we adopt it and
deprecate ours — the contract is one optional object in free-form details.

## Alternatives considered

- **Propose the contract upstream to pi first.** Lost for V1: pi needs no
  change (details already flows), so waiting on upstream review adds delay
  for zero technical need. An upstream issue socializing the shape runs in
  parallel; if pi standardizes later, we conform.
- **Special-case `agent_browser` in the conversation UI.** Refused: tool-name
  coupling in the host is the fork ADR-0003 forbids, and every other tool
  would stay blind.
- **VNC / noVNC into the isolated Chromium.** Refused: a second transport to
  secure, worse fidelity than frame-per-step, no read-only mode; Browserbase
  and browser-use both standardized on embeddable frame streams instead.
- **Frames on the ADR-0048 events feed.** Refused: the feed is durable audit;
  frames are ephemeral bulk. The agent WebSocket is the honest channel.
- **VNC-style forced screenshot cadence from the GUI.** Refused: extra
  `screenshot` calls burn the agent's context and time; the emitter owns
  cadence.
