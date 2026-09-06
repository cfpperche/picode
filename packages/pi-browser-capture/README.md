# pi-browser-capture

A standalone pi extension: opt-in **live browser frames** while any
`agent_browser` tool call runs, plus a persisted final frame for replay —
**without patching** [pi-agent-browser-native](https://github.com/fitchmultz/pi-agent-browser-native)
(ADR-0080).

## How it works

1. The extension observes pi's own `tool_execution_start` / `tool_execution_end`
   events for the `agent_browser` tool.
2. It locates the session's existing upstream agent-browser stream rendezvous
   (`<session>.stream` + `.pid` beside the daemon socket, re-deriving the same
   session name upstream uses) and connects a loopback WebSocket with
   `?maxFps=1` push pacing. Ack pacing is deliberately avoided: a client that
   attaches before the browser exists deadlocks until its first ack.
3. The latest bounded frame (JPEG ≤ 200 KiB, ≤ 1600 px per side, ≤ 1.6 Mpx) is
   mirrored to `<pi-session-file>.capture/current.jpg` + `current.json`
   (atomic latest-wins files) and, when the call ends with frames captured, a
   `browser-capture-final` session entry persists the last frame for replay.
4. A host (PiCode's daemon) watches the capture directory only while a browser
   tool runs and forwards frames to the UI feed.

Nothing here launches a browser, sends input, or enables a server. Reading is
strictly local: uid/mode checks on the socket dir, `O_NOFOLLOW` on the
rendezvous files, pid liveness, loopback only. Windows is unsupported (upstream
has no socket dir there).

## Consent

- `--browser-captures` flag or `/browser-captures on|off|status`.
- The choice is branch-persisted (`browser-capture-consent` entries) and
  restored on session start/tree.
- Frames may contain page secrets. When enabled, they stream to the PiCode
  server during browser calls and the final frame persists in session history.

## Install

Like any pi package: add this directory (or a git/npm ref) to `packages` in
agent settings, or load `extensions/capture.ts` with `-e`. Requires
pi-agent-browser-native ≥ 0.6.6 (upstream agent-browser 0.36 stream rendezvous)
at runtime — installed separately, unmodified.

## Tests

```bash
npm test            # pure-logic tests (naming, JPEG bounds, rendezvous safety)
node scripts/verify-browser-capture-sidecar.mjs <native-checkout> <receipt-dir> [model]
                    # real model + real browser RPC proof (private receipts)
```
