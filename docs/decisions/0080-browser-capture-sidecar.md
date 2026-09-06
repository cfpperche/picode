# ADR-0080: Browser capture as a standalone sidecar extension

- **Status**: accepted (owner approved the sidecar direction and the phased
  plan — prove over real RPC, integrate the UI, then drop the patch — 2026-09-05)
- **Date**: 2026-09-05
- **Supersedes**: the *emitter placement* of ADR-0076 (bounded captures emitted
  from inside pi-agent-browser-native). ADR-0076's bounded-image contract,
  consent model and tool-agnostic renderer remain in force.
- **Driver**: `packages/pi-browser-capture`

## Context

Browser preview needs pixel frames while `agent_browser` runs. ADR-0076 placed
the emitter inside pi-agent-browser-native as a pinned binary patch
(`patches/pi-agent-browser-native/bounded-captures.patch`, base 0.6.6). The
patch works — real RPC and UI acceptance passed — but it must be re-applied to
every upstream release and any upstream contribution is gated on the
maintainer of a package we do not own (issue #157 is our proposal,
unanswered).

Reading pi's extension docs shows a second extension can observe
`tool_execution_start`/`tool_execution_end` for the `agent_browser` tool, can
re-derive the upstream session name (`piab-<slug>-<sessionHash>-<cwdHash>`,
`-fresh-*` rotations matched by prefix), and can read the same
`<session>.stream`/`.pid` rendezvous the patch read. What it cannot do is emit
into another tool's `tool_execution_update` stream — updates belong to the
executing tool's `onUpdate`.

Two live findings sharpen the design. First, an instrumented run showed
`?pacing=ack` URL pacing deadlocks a client that attaches before the browser
exists: the opening frame counts in-flight, later frames replace it, and acks
can only echo a frame already received — so the sidecar uses push pacing with
`?maxFps=1` and raises the cap only for the final drain. Second, the RPC stream
carries `entry_appended` and `extension_ui_request` events, and custom session
entries are branch-persistent — giving the host live consent state and a
replayable final frame without touching tool results.

## Decision

Browser capture emission moves to `packages/pi-browser-capture`, a standalone
pi extension installed like any other package:

1. It arms on `tool_execution_start` for `agent_browser` when consent
   (`/browser-captures`, `--browser-captures`, branch-persisted) is on.
2. It reads the upstream rendezvous with the same safety checks the patch used
   (uid/0700 dir, `O_NOFOLLOW`, pid liveness, loopback) and mirrors the newest
   bounded frame to `<pi-session-file>.capture/current.jpg` + `current.json`
   (atomic latest-wins, ≤ 200 KiB, ≤ 1600 px/side, ≤ 1.6 Mpx).
3. On `tool_execution_end` it drains briefly, marks the frame final, and
   appends a `browser-capture-final` entry so hosts replay the last frame.
4. PiCode's daemon (phase B, in flight) watches the capture directory only
   while a browser tool runs and forwards frames to the UI feed keyed by
   `toolCallId`; the UI pill subscribes per ADR-0048.

The pinned patch remains the interim deliverable until phase B passes the same
real-RPC and visual acceptance; then `patches/pi-agent-browser-native/` is
removed and `var/native-browser` returns to a clean checkout.

## Consequences

- No patch to maintain and no upstream PR gate; the extension survives
  upstream releases as long as the documented stream rendezvous holds (the
  same dependency the wrapper itself versions in its support matrix).
- Live frames travel over the feed instead of `tool_execution_update`; hosts
  that want them must watch the capture directory (or feed). The renderer
  contract stays "optional metadata, latest wins".
- The final frame lives in a session entry rather than tool-result details;
  replay reads entries, not tool details.
- Upstream contribution, if it ever happens, becomes "adopt this separate
  package" instead of "merge our patch".

## Alternatives considered

- **Keep the pinned patch** (status quo): proven, but couples PiCode to an
  upstream internal layout on every release.
- **Emit in-tool updates from the sidecar**: impossible without patching —
  updates belong to the executing tool.
- **Daemon-side emitter with no extension**: pushes browser knowledge into
  PiCode core and duplicates consent handling; the sidecar keeps it reusable
  by any pi host.
