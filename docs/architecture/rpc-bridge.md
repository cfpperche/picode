# RPCBridge

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`internal/rpc`: JSONL client for `pi --mode rpc` (strict `\n` framing via
bufio.Scanner, command/response correlation by id, event fan-out, exit
propagation) plus the **managed runtime** (ADR-0006): task delivery engine
claiming from the store (`prompt` waits for `agent_settled`; `steer` /
`follow_up` send while the turn is running), finished delivered/failed with
audit), per-agent event hub feeding `GET /ws/agent?agent=<id>` (events +
`enqueue` input). Thinking text is never projected to the status surface.
Extension dialogs (`select`/`confirm`/`input`/`editor`) surface as RPC
`extension_ui_request`. Managed snapshot includes `waiting` + `dialog`.
The GUI shows one compact stepper per turn: labeled pills for answers (click
to go back), a filterable dropdown (cmdk) for the current field. After the
last answer the thread keeps one definition line (`vision — xai/grok-4.5 ·
medium`), persisted for reload. **Yes / No / Cancel / type** POST
`/api/agents/{id}/ui` (`ReplyUI`). Notify is a toast, not waiting, unless a
form just finished.
Timeout on the request dismisses the card (pi auto-resolves).
Auto-approve policy stays undecided. Track C:
[conversation-control-roadmap.md](design/conversation-control-roadmap.md).

**Tool captures (ADRs 0057/0076):** a tool may emit
`details.preview = { image, url?, title?, ts?, source? }` in partial results
and the final result. The host accepts only base64 PNG/JPEG captures up to
200 KiB, 1600 pixels per side and 1.6 million pixels; remote URLs never load.
`internal/toolpreview` bounds metadata before RPC web fan-out and transcript
presentation; the shared browser parser checks again before rendering.
Captions omit URL credentials/query/fragment and say **Last capture**. Invalid
captures show an unavailable state; decode failures offer Retry. The final
result is authoritative, including absence of a capture; completed tools
ignore late updates. Expanded text omits preview image bytes.

Desktop rejects stale selection/request/socket generations and reconciles the
live suffix with history by tool identity. Mobile owns socket reconnection and
history reconciliation on snapshot/settle, instead of appending an uncorrelated
history fetch from the screen. This is not a replayable frame stream: missing
partials are not cached server-side, and the existing slow-consumer hub can
drop events. Emitter limits and integration acceptance remain required.

The built-in renderer is tool-agnostic; capture emission must be explicitly
enabled in a package loaded by the agent, globally or at workspace/agent scope.
Emission is moving to `packages/pi-browser-capture`, a standalone sidecar
extension (ADR-0082): it mirrors bounded frames to
`<pi-session-file>.capture/` during `agent_browser` calls and persists the
final frame as a session entry, instead of patching pi-agent-browser-native.
Real-RPC proof, the daemon capture-directory bridge and live/replay UI
rendering (desktop + mobile) all passed against an unpatched 0.6.6 checkout.
The superseded in-package patch and the panel are tracked in the
[delivery plan](plans/browser-preview.md).
Pi owns raw session persistence; host validation cannot remove pixels already
written there or bound the raw RPC input before decoding.
