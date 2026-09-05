# Browser preview delivery plan

Owner approved the revised phases 1–4 on 2026-09-05. The renderer is built
in; capture emission is opt-in in a Pi package loaded by the agent (global,
workspace or agent scope). Nothing is installed automatically. Dependencies
run on the agent's host, not on the observer's phone.

Adaptation: [Cursor's inline screenshot/activity pattern](../benchmarks/2026-09-02-live-browser-preview.md),
with explicit **Last capture** wording rather than a claim of continuous video.
Desktop and mobile own their UI independently (ADR-0072).

## Increments

1. Bound the existing contract and document privacy (ADR-0076).
2. Protect tool state during transcript reconciliation; gate stale requests
   and socket events by agent/connection; handle unavailable images in both apps.
3. Spike the installed native browser package without modifying the global
   installation. Prove partial and final capture emission through real Pi RPC.
   Prefer an upstream change; do not replace the native tool with a second
   browser driver or claim final-result hooks deliver intra-call frames.
4. Integrate an opt-in emitter only after step 3 succeeds. Measure normal,
   cancelled and slow-consumer flows; run full CI and isolated visual QA.
5. Later: an explicitly opened last-capture panel with source identity.
6. Separately approved: authenticated streaming and actual control transfer.

## Host decision table

| Conditions | Action / acceptance |
|---|---|
| No preview metadata | Ordinary tool row; no empty preview panel |
| Valid bounded PNG/JPEG data URI | Render last capture; click opens existing image viewer |
| External URL, invalid base64/format or oversized image | No image request; compact unavailable state |
| Image fails to decode | Unavailable state and Retry action; never a broken image well |
| Optional timestamp/source absent | Legacy capture accepted; no invented age/session |
| Older timestamp during the same call | Keep newer capture |
| Update after tool completion | Ignore; final result is authoritative |
| Final result has no preview | Clear transient capture, matching replay |
| Slow history read overlaps live tools | Reconcile by toolCallId; do not erase active/newer state |
| Same tool in history and socket | One row; completion beats pending history |
| Old request/socket responds after agent/connection change | Ignore |
| Two concurrent history requests | Only the newest request may apply |
| New history has no matching tool identity | Do not attach another session's capture |

## Implementation checkpoint — 2026-09-05

The host increment (1–2) is implemented in `feat/tool-preview-hardening`.
Go capture-policy tests and a native-session-file projection test cover the
server boundary without rewriting Pi files. Shared capture/reconciliation
matrices and both reducers cover accepted/refused data, ordering, terminal
results and duplicate tool starts. Browser fixtures proved successful delayed
history reads preserve one tool/frame, mobile reconnect and rapid selection
changes, final capture plus ignored late update, and both apps' image controls.
Ten state screenshots were read; overlay audits passed. `make ci` passed with
662 frontend tests plus Go/package/build/docs/Vale gates. This is **host QA**,
not an end-to-end browser-package acceptance claim.

### Emitter spike: supported hooks inspected, integration not delivered

Inspected installed `pi-agent-browser-native@0.6.6` and Pi `0.85.1`:

- Native `browser-run/prepare.js` calls `onUpdate` with textual command
  progress and session metadata, not a capture. Actual native screenshot calls
  during QA returned verified artifacts, not `details.preview`.
- Pi's public `tool_execution_update` handler is notification-only
  (`ExtensionHandler<ToolExecutionUpdateEvent>`). `tool_result` supports a
  final patch (`ToolResultEventResult`). `getAllTools()` exposes metadata, not
  another tool's execution callback. Observing final results alone therefore
  does not establish intra-call capture emission.
- No companion was installed, no native tool was overridden, and no arbitrary
  artifact path was read by a new production integration. Native global
  configuration and the installed service remain unchanged.

**Next:** implement/propose an opt-in change at the native package's own
orchestration boundary, preserving session locks, permission gates, artifact
verification and cancellation. Prove simple calls first, then `script`, `job`
and `qa`. The existing [upstream issue #157](https://github.com/fitchmultz/pi-agent-browser-native/issues/157)
is the discussion point; its original loose-URL proposal needs the revised
capture policy before a PR. A package companion remains an option only if a
supported delegation/partial-emission interface is demonstrated. This is an
integration cost, not a permanent refusal.

**Open acceptance / FAIL debt:** no real emitter-to-Pi-RPC run, no measured
slow-consumer/cancellation matrix, and no real-browser-session isolation proof.
The hub still drops on overflow and does not cache missed partials. Desktop
socket reconnection beyond its existing lifecycle remains unproved; the mobile
retry/reconciliation path was exercised with a fixture. Same-agent session
replacement during a pending session-changing API call also needs dedicated
acceptance; the implemented guards cover selection, request generation and
resolved history-path changes, not a new session-ownership protocol.
Phases 3–4 and the later panel/stream are not complete or enabled.

### Reproduce host visual QA

Build the UI and embedded `cmd/picode-docs-fixture`; run it on the private
address `127.0.0.1:18785`. Do not point this fixture at the installed service.
Generate an init script by prepending
`globalThis.__captureFixtureImage = "data:image/png;base64,...";` (the bounded
`www/img/app-fleet.png` encoded as base64) to
`scripts/fixtures/tool-preview-init.js`. Open the fixture's Atlas agent in a
fresh native `agent_browser` session with `--init-script <generated path>`.

The fixture accepts `?capture=none` for the ordinary row. Its
`captureQA.mode('valid'|'blocked'|'broken')`, `finish()`, `late()` and
`disconnect()` methods inject deterministic socket events. History succeeds
but takes 1.6s, so the original overwrite race is exercised rather than hidden
by an aborted request. Inspect the app through normal controls, capture and
read PNGs, then stop only the exact fixture process you launched.

## Merge review — 2026-09-05

The owner authorized merge and deployment of the host increment. Integrations
had already taken ADR-0075, so the capture decision is now ADR-0076 with no
policy change. Combined main `make ci` passed; ten host screenshots were
recaptured and read. A preexisting mobile toolbar overflow was corrected with
wrapping; Send remains inside 320/390px viewports while Stop is visible.
Receipts are in `var/capture-merge/`; deployment state belongs in the handoff.
This does not change the pending real-emitter acceptance below.

## Integration acceptance (not certified by renderer fixtures)

- Installed upstream version and supported hooks recorded; global settings untouched.
- Real tool call: intermediate and final captures, URL/title provenance,
  reload, failed capture, cancellation, and session switch.
- `script`, `job` and `qa`: prove intra-call cadence or explicitly mark unavailable.
- Slow consumer: measure frame bytes and event delivery; do not claim that
  frontend size checks bound Pi's own session files or the raw RPC scanner.
- Desktop/mobile: empty, blocked, decode error, running, settled and image viewer;
  light/dark and narrow widths. Read images, audit overlays, run `make ci`.
- QA uses private data/ports and exact process identities; no service restart.
