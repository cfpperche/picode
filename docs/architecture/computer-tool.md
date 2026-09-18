# Computer tool (ADR-0148)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

An agent, or a CLI in a PiCode terminal, uses the Windows desktop through
the desktop shell. One grant per principal is the whole policy of v1
(capability first, the owner's call of 2026-09-17); tiers, a window
binding, confirmation classes and a sandbox are named refinements, each an
amendment to ADR-0148. Study and plan: `docs/benchmarks/2026-09-16-computer-use.md`,
`docs/plans/computer-use.md`.

## Who decides what

| Question | Decided by | Where |
|---|---|---|
| Who is calling | daemon | `internal/grant.Key`: agent id, else `term:<terminal id>`, else nobody (ADR-0143) |
| May they use the computer | daemon, then the shell's mirror | `internal/computer.Resolve` reads `computer.policy.<key>` = `{"enabled":true}`; missing or broken is off. The page pushes the enabled keys to the shell (`computer_set_grants`) at load and on `setting.updated`, and `computer_call` refuses on that copy too |
| Is the action real | daemon, then the shell | the closed catalog of 23 (`internal/computer/actions.go`, mirrored by `computer.rs`) |
| What the action does | shell | `desktop-shell/src/computer.rs` on the desk thread |
| What is recorded | daemon | one `computer.step` event per call, allowed / refused / failed |

## The line

`pi-computer` (`packages/pi-computer`) → `POST /api/computer/tool`
`{agent, term, call, action, params}` → `browser.Hub.Dispatch` with
`Command{Kind:"computer", Method:action, Params, Principal:key, Timeout}` →
the shell page's one `EventSource` on `/api/browser/stream` → `browserChannel.js`
dispatches on `kind` → `createComputerRunner` → `invoke("computer_call",
{principal, action, paramsJson})` → `POST /api/browser/result` → the tool
answer `{action, output}`. No second stream, port or credential (ADR-0132).

Answers: a capture is `{ok, image (base64 PNG ≤ 1280 px), mime, meta{display,
window, width, height, scale, bounds, seq, cursor, preview (JPEG data URL ≤
480 px), ms}}`; `snapshot` is `{window, lines[], dropped, nodes}`; `windows`
is `{windows[], displays[]}`; the rest one small object. Shell refusals are
`error` strings with a fixed head word: `disabled`, `no such window`,
`bad_coordinate`, `foreground_refused`, `capture_failed`, `unknown_action`,
`timeout`; the daemon adds `not connected` (no shell) and the grant refusal
that names Settings ▸ Computer.

## The actuator (shell)

One "desk" thread, created on first use: `CoInitializeEx(MULTITHREADED)`
for UI Automation, `SetThreadDpiAwarenessContext(PER_MONITOR_AWARE_V2)` so
every rectangle is physical, one job at a time (the interaction lease), no
COM object across threads. `desktop.rs` enumerates monitors (primary is
display 1) and top-level windows (visible, uncloaked, titled, not tool
windows; the exe through `QueryFullProcessImageNameW`); `capture.rs` takes a
monitor by `BitBlt`+`CAPTUREBLT` with the cursor drawn, or a window by
`PrintWindow(PW_RENDERFULLCONTENT)` with a screen fallback when it paints
black, then scales and encodes through WIC; `input.rs` is `SendInput`
(absolute moves over the virtual screen, buttons, wheel, virtual keys with
the extended flag, text as Unicode events); `uia.rs` walks the control view
with a cache request (name, type, id, rect, enabled, offscreen, password,
value); `clipboard.rs` is CF_UNICODETEXT; `open` is `cmd /C start`.

Coordinates: the model sees images; `geometry::Frame` (pure, tested)
records each returned image's origin, physical size and image size per
principal, maps `[x, y]` back to the pixel centre, and refuses points
outside the image (`bad_coordinate`). `zoom` makes a new frame over a
region at up to 1:1. The pure modules — `geometry`, `keys` (xdotool names →
virtual keys), `b64`, `axfmt` (snapshot lines) — run with
`rustc --edition 2021 --test src/lib.rs`; there is no cargo test lane.

Commands (each registered in `main.rs`, `build.rs` and
`capabilities/default.json`): `computer_displays`, `computer_windows`,
`computer_call`, `computer_set_grants`, `computer_preview`. The Computer lab
(tray, `ui/computerlab.html`, its own capability) drives them by hand.

## What the human sees

Settings ▸ Computer (`ComputerPage.jsx`): one switch per principal through
`GET/POST /api/computer/policies|policy`, the recent steps from
`GET /api/computer/audit`. The chat: the tool's `details.preview` lands as
the step's "Last capture" (`toolPreview.js`, the `agent_browser` path), and
the step label reads the action (`toolArgs.js`, `turns.js`). The dashboard:
`desktop` on `GET /api/sessions/stats` (steps, time, refused/failed in the
window, from `Store.EventsOfTypeBetween`).

## Known limits (v1)

Unrestricted by design behind the switch; screen content is not filtered
for injected instructions. `SetForegroundWindow` may be refused while the
human works elsewhere (`foreground_refused`). `PrintWindow` paints some
windows black (fallback to the screen). Input into elevated windows is
dropped by UIPI and reported by count. One desk thread: a five-second
`wait` delays other principals. Windows only.
