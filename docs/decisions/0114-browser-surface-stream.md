# ADR-0114: Browser surface — the daemon proxies the engine's stream WebSocket

- **Status**: superseded by ADR-0117 (feature removed 2026-09-10)
- **Date**: 2026-09-10
- **Boundary**: protocol (a new authenticated WebSocket proxy joins the
  daemon's public surface) and security model (frames and — in a later phase —
  input for the agent's Chromium cross the daemon to any authenticated client,
  including paired remote devices). Study:
  [docs/benchmarks/2026-09-10-browser-surface.md](../benchmarks/2026-09-10-browser-surface.md).

## Context

The user-facing browser surface needs two things PiCode does not have yet: a
live pixel channel from the agent's Chromium into the GUI, and a surface to
render it. The engine already ships the first half: every `agent-browser`
session starts a loopback WebSocket that streams JPEG viewport frames (base64
JSON, latest-wins, per-client `maxFps`/ack pacing) and accepts input events.
It is loopback-only by policy — non-localhost browser origins get a 403 on the
upgrade — so a remote viewer (paired phone, second machine) cannot reach it
directly, and the GUI must not open a second unauthenticated port.

The daemon already holds the keys to the rendezvous: `get_state` returns
`sessionId` + `sessionFile`, and the pi-browser-capture sidecar (ADR-0082)
proved the discovery algorithm — derive the implicit session name
(`piab-<slug>-<id12>-<cwdhash8>`, `-fresh-` rotations by mtime), then read
`<name>.pid` + `<name>.stream` from the socket root under the same safety
checks (uid/0700 dir, realpath, O_NOFOLLOW, ≤16-byte numeric files, pid
liveness). Benchmarks (Cursor, Devin, Manus, Browserbase, browser-use, Orca,
Lovable, Replit, bolt.diy — see the study) converge on exactly this shape:
frames over an authenticated, proxied channel; frames never ride the durable
event feed (the 2026-09-02 study refuses that too).

## Decision

PiCode gains a **browser surface** (a native shell view, not an ADR-0036 app —
apps cannot touch the DOM, and a live canvas is the opposite of a primitive
tree) fed by a **daemon WebSocket proxy**:

1. `GET /ws/browser?agent=<id>` rides the existing auth gate (the one
   `Auth.Wrap` in front of every route, ADR-0049). The handler resolves the
   agent's engine stream port via the rendezvous, dials
   `ws://127.0.0.1:<port>/`, and pumps engine messages to the client verbatim.
2. **Phase 1 is read-only**: client → engine, the proxy forwards only
   `config` (pacing/maxFps) and `ack` messages. `input_*` messages are refused
   with an error envelope; interactive control is a later phase behind its own
   consent decision.
3. No frames in the feed, no frames in transcripts, nothing persisted. The
   proxy is a pipe: what the engine forgets, the daemon forgets.
4. The client renders frames on a canvas inside a `Browser` view of the agent
   tab (the Chat | TUI pattern gains a third state). Mobile keeps the ADR-0082
   capture pill in v1; the stream view is desktop.

## Consequences

**Easier:** any authenticated client — desktop, and later the phone — sees
what the agent's browser sees, live, with take-over possible in phase 2
without new transport. The engine's own controls (per-client `maxFps`,
ack pacing, idle-input reset) keep working through the proxy because the
proxy forwards, not terminates. The 2026-09-02 study's v2 (preview proxy) is
subsumed: same pipe, now bidirectional for pacing.

**Harder / accepted costs:**

- The proxy is a second hop: end-to-end pacing requires forwarding the
  renderer's acks (documented engine requirement); a dumb binary pump would
  queue frames behind a stalled client.
- Pairing scope: an authenticated device can watch the agent's browser. That
  is the same trust level as attaching to its terminal (ADR-0085) — accepted,
  and the reason input stays out of phase 1.
- Engine dependency: the rendezvous layout is a documented, versioned-in-practice
  contract the sidecar already depends on (ADR-0082); if it moves, both
  break together and the surface degrades to the capture pill.
- The proxy must never buffer: frames are latest-wins end to end, so a slow
  client falls behind and then rejoins the present, never replays history.

**Who breaks if we are wrong:** if discovery matches the wrong session
(`-fresh-` rotations), the surface shows another session's browser — mitigated
by the mtime-newest match the sidecar validated, and visible (URL bar shows
the page), not silent. If the proxy leaks to unauthenticated callers, the
auth gate failed for every route; the browser adds no bypass.

## Alternatives considered

- **iframe of the dev server + injected inspector script** (the
  Lovable/bolt.diy shape): best latency for the *project under development*,
  but blind to external sites, needs a proxy-injected script or bundler
  plugin, and a second rendering path. Kept as a possible later complement,
  not the base.
- **Frames on the ADR-0048 feed** (the capture pill's channel): the feed is
  durable audit; frames are ephemeral bulk. Refused by the 2026-09-02 study;
  unchanged.
- **VNC/noVNC into the Chromium**: second transport, worse fidelity, no
  read-only mode. Refused (2026-09-02 study, re-confirmed by the benchmarks).
- **Daemon-side headless browser owned by PiCode** (chromedp and friends):
  forks the engine the ecosystem already drives (ADR-0003: orchestrate, do
  not fork) and splits the "same browser" property — the human and the agent
  would look at different browsers.
- **An ADR-0036 app** for the surface: apps render schema-driven primitives
  and cannot own a canvas or input model; the postmortem the ADR cites
  (VS Code notebooks) is exactly this mistake.
