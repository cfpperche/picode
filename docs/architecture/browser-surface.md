# Browser surface (ADR-0114, ADR-0115)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

The agent tab's third view (Chat | TUI | **Browser**, opened from the agent
row menu): a live view of the browser the agent drives through
`agent_browser`. One browser for human and agent — PiCode never launches or
forks its own (ADR-0003).

## The pipe

| Hop | What |
|---|---|
| Engine | Every `agent-browser` session starts a loopback WebSocket: JPEG frames (latest-wins, per-client `maxFps`/ack pacing), `url`/`status` messages, and — with consent — `input_*` |
| Daemon | `GET /ws/browser?agent=<id>` rides the one auth gate (ADR-0049), dials the engine's rendezvous port (`internal/rpc/browser_stream.go`: implicit session name + uid/0700/O_NOFOLLOW/liveness checks mirroring the ADR-0082 sidecar) and pumps both ways. Pacing (`config`/`ack`) is forwarded so bounds hold end to end; engine death closes the client |
| UI | `web/desktop/src/components/BrowserSurface.jsx` draws frames on a canvas, letterboxed; the bar shows the live dot, the page URL, and the mode chips |

Frames never ride the ADR-0048 feed and are never persisted; the capture
pill (ADR-0082) remains the replayable in-chat record.

## Modes (per view, not per agent)

| Chip | Meaning |
|---|---|
| **Watch-only** (default) | Frames only. Input from this client is refused with a `watch-only` envelope |
| **Control on** | Consent (ADR-0115): the daemon route `POST /api/agents/{id}/browser-input` flips `<session>.capture/input.json`; the proxy then forwards `input_mouse`/`input_keyboard`/`input_touch`. Consent is keyed to the session file — a new or forked session starts off. `/browser-input` in a TUI writes the same mirror |
| **Design** | Read-only pick: a click maps through the letterbox into viewport coordinates; the surface sends the cropped frame + point + the user's intent as a normal prompt (with image) — the agent inspects the element with its own `agent_browser` tools and edits the code. Design pauses Control forwarding while on |

Control and Design are both desktop-only today; mobile keeps the ADR-0082
capture pill. No remote scrolling: the engine's stream protocol has no wheel
event yet.

## Constraints worth remembering

- The managed-mode prompt API does not run slash commands — tool-style text
  reaches the model as a message (tested live, ADR-0115 alternatives).
- `POST /api/agents/{id}/command` is the TUI door (it stops the managed
  runtime); it is never a browser path.
- The rendezvous layout is a live contract shared with
  `packages/pi-browser-capture`; if upstream moves it, discovery and the
  sidecar break together and the surface degrades to the capture pill.
