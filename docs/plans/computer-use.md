# Project plan: computer use for agents (the `computer` tool)

Status: approved by the owner in session on 2026-09-17 (capability first: one
grant, unrestricted tools, refinements later). Executes ADR-0148 (proposed).
Study: `docs/benchmarks/2026-09-16-computer-use.md`.

## Objective

An agent, or a CLI in a PiCode terminal, that the human has switched on can
use the Windows desktop through the shell: see a monitor, list and focus
windows, read a window's UI Automation tree, click, type, scroll, drag, use
the clipboard and open programs — with the human's own permissions, the way
Windows-MCP and Agent S work today. Tiers, a window binding, Ask classes, an
app allowlist, a human-activity pause and a sandbox are refinements listed at
the end, in the order the owner chooses.

## Engineer decisions (the owner may override any)

1. **One grant per principal.** `computer.policy.<key>` = `{"enabled": true}`,
   key = agent id or `term:<terminal id>` (`internal/grant`, ADR-0143).
   Missing = off, fail closed; no identity = off. No machine-wide switch in
   v1: the row per principal in Settings ▸ Computer is the switch.
2. **With the grant, everything**: 23 actions (Anthropic's 17 + `windows`,
   `focus`, `snapshot`, `clipboard_read`, `clipboard_write`, `open`) over any
   window and the whole desktop.
3. **A screenshot is one monitor** (the primary by default; `display` and
   `window` optional), downscaled by the shell to ≤ 1280 px wide, returned as
   a pi image block. Coordinates are in the pixel space of the last image;
   the shell keeps `Frame{origin, scale}` per principal and maps back to
   physical pixels — the shell's first physical-pixel space (it is all
   logical today).
4. **Anthropic's vocabulary one-to-one**, `zoom` included; batch per turn
   (stop at the first failure, the rest answer `Not executed…`).
5. **One "desk" worker thread** in the shell serializes capture, input and
   UI Automation: one action at a time on the machine, no COM object crosses
   a thread, UIA never on the UI thread.
6. **No new crate**: `windows` 0.61 feature flags only. No new port,
   credential or Python sidecar.
7. **Light audit**: one `computer.step` event per call on the change feed
   (ADR-0048) — a record for debugging and the dashboard, not a gate.
8. **Honest public page**: with the grant the agent acts with the user's
   permissions, on the user's desktop, without a sandbox; what comes next is
   named.

## Spec

### The tool (`packages/pi-computer`)

| `action` | Params | Answer |
|---|---|---|
| `screenshot` | `display?`, `window?` | image + "Screenshot of display 1 (1280×720, scale 1.5)" |
| `zoom` | `region [x0,y0,x1,y1]`, `scale?` | image of the region; new frame |
| `snapshot` | `window?` (default: foreground), `depth?`, `ref?` | UIA tree: `e3 Edit "Text Editor" @center(640,400) depth=2` (≤ 400 nodes, `N dropped`) |
| `cursor_position` | — | `[x,y]` in the last image |
| `wait` | `duration ≤ 5` | new image |
| `windows` | — | `w132458 notepad.exe "Untitled - Notepad" display=1 [bounds] foreground=yes` |
| `focus` | `window` | `SetForegroundWindow`; `foreground_refused` when Windows declines |
| clicks, `mouse_move`, `left_mouse_down/up` | `coordinate`, `text` = modifiers | "ok" + image |
| `left_click_drag` | `start_coordinate`, `coordinate` | idem |
| `scroll` | `coordinate`, `scroll_direction`, `scroll_amount ≤ 50` | idem |
| `type`, `key`, `hold_key` | `text`, `duration ≤ 5` | idem |
| `clipboard_read` / `clipboard_write` | `text` | text / "ok" |
| `open` | `target` (app, path or URL) | "ok" via `ShellExecuteW` |

Error heads (text the model reads): `disabled` · `not_connected` ·
`foreground_refused` · `capture_failed` · `bad_coordinate` · `stale_ref`
(M4). HTTP: 403 without a grant ("this agent may not use the computer — turn
it on in Settings ▸ Computer"), 400 unknown action (lists the 23), 502 hub
errors verbatim. `parameters` are typebox with bounded arrays (never
`Type.Tuple`). `execute` returns `content: [text, image?]` and
`details: {preview: {image (JPEG ≤ 480 px / 60 KiB), title, seq, final: true},
window}` so the existing "Last capture" thumbnail works without a daemon
change. The prompt guidelines say: the screenshot is one monitor (or the
window asked for) and coordinates are in the last image; screenshot after
acting; `windows` + `focus` before typing into an app; you act with the
user's permissions — confirm with the human before paying, sending, deleting
or typing passwords; a refusal says what is missing.

### Grant and identity (daemon)

`internal/grant` (shared with the browser, landed in M0);
`internal/computer/policy.go` (`Grant{Enabled}`, `SettingPrefix
"computer.policy."`, `Resolve/ResolveCaller/Save`) and `actions.go` (the
closed catalog of 23). Routes: `POST /api/computer/tool` `{agent, term, call,
action, params}` → `{action, output}`; `GET /api/computer/policies` and `POST
/api/computer/policy` `{agent|term, enabled}`; `GET /api/computer/audit`.
Registered in `registerAll`; the OpenAPI regenerates on `make close`. No
migration: settings and events. Handler order: decode → `ActionFor` → params
→ `ResolveCaller` → grant → `Dispatch` (kind `computer`, 30 s; `wait` and
`hold_key` ≤ 5 s) → audit (`computer.step` `{principal, termId, call, action,
outcome, reason, window, display, ms, imageSha256}`; a real agent id or nil)
→ 200.

### Transport (the ADR-0132 line, one hub)

`Command` carries `Kind`, `Principal` and a per-command `Timeout` (M0). The
SSE event stays `command`; the page dispatches on `cmd.kind`; results post
to `/api/browser/result`. Frame: `{"id","kind":"computer","method":"left_click",
"params":{"coordinate":[412,88]},"principal":"agent-1"}`. Result `output`:
`{ok, image (base64), mime, meta {display, window {hwnd, pid, exe, title},
width, height, scale, bounds, seq, cursor, ms, preview}}`; `snapshot` →
`{nodes: [{ref, role, name, value, center, depth}], dropped}`; `windows` →
`{windows: [...]}`; shell refusals in `error` with a fixed head word. Web:
`lib/browserChannel.js` gains `runComputer` and dispatches by kind;
`lib/computerChannel.js` → `createComputerRunner({invoke})` →
`invoke("computer_call", {principal, action, paramsJson})`, injected and
tested like the browser's. One `EventSource`.

### The actuator (shell, Rust, no new crate)

`src/computer.rs` (`ComputerState`: `jobs: Sender<Job>`, mirrored grants in a
fail-closed `OnceLock<Mutex<HashSet<String>>>` pushed by the page through
`computer_set_grants`; the desk thread created on first use with
`CoInitializeEx(MULTITHREADED)`, `SetThreadDpiAwarenessContext(PER_MONITOR_AWARE_V2)`
and one `CUIAutomation`; `computer_call` sends a job and waits
`recv_timeout(20 s)` like `btab_cdp_call`); `src/desktop.rs` (monitors,
windows, foreground); `src/capture.rs` (`BitBlt` of a monitor, `PrintWindow`
of a window, cursor drawn, WIC downscale and PNG/JPEG); `src/input.rs`
(`SetCursorPos` + `SendInput`, `type` as `KEYEVENTF_UNICODE`, `release_all`
on any refusal); `src/uia.rs` (`ElementFromHandle`, cache request,
`ControlViewWalker`); `src/clipboard.rs`; `open` through the `btab_open_path`
path. Pure modules under `lib.rs`, tested with `rustc --edition 2021 --test
src/<mod>.rs` (there is no cargo test lane): `geometry.rs`, `keys.rs`,
`b64.rs`, `axfmt.rs`. Commands (each one = `generate_handler!` + `build.rs`
+ `capabilities/default.json` + the generated, committed permission toml):
`computer_windows`, `computer_displays`, `computer_call`,
`computer_set_grants`, `computer_preview`. `Cargo.toml` gains the `windows`
features `Win32_Graphics_Gdi`, `Win32_Graphics_Dwm`, `Win32_Graphics_Imaging`,
`Win32_UI_Accessibility`, `Win32_UI_HiDpi`, `Win32_UI_Input_KeyboardAndMouse`,
`Win32_System_DataExchange`, `Win32_System_Memory`,
`Win32_System_SystemInformation`. A **Computer lab** (tray item, own
capability, the Browser lab pattern) is the owner's manual acceptance on
Windows.

### Web, settings and docs

Settings ▸ Computer (`components/ComputerPage.jsx`): one row per principal
(agents and terminals with a CLI, as Agent permissions lists them) with a
switch, saved through `POST /api/computer/policy`; an audit list like the
Developer-mode one; `Item`/`SwitchCtl` lifted from `BrowserPage.jsx` into
`components/settingsControls.jsx`; the menu entry gated by `inShell`
(desktop only, like Browser). Per step: `details.preview` → `captureOnEnd`
→ `ToolCapture.jsx`; `stepLabel` in `web/shared/domain/turns.js`. Dashboard:
`computer.step` counted in `handleSessionStats` → a "Desktop steps" tile.
Docs: `docs-site/guide/computer-tool.md`, `docs/architecture/computer-tool.md`,
a paragraph in `docs-site/guide/packages.md`, a changelog fragment.

## Milestones and gates

### M0 — Foundations (`feat/computer-m0`)

Scope: ADR-0148 (proposed); this plan; `internal/grant` + the browser's
`ResolveCaller` delegating to it; `Command.Kind/Principal/Timeout` in
`internal/browser/hub.go` with tests; `browser screenshot` returns an image
block instead of a temp path (`packages/pi-browser`); changelog fragment.

Gate: `make ci-scoped` green; browser tests unchanged; an agent sees the
`browser screenshot` image without calling `read`.

Owner acts: reads and approves ADR-0148.

### M1 — Actuator and lab (`feat/computer-m1`)

Status 2026-09-18: code landed on the branch (the desk thread, GDI capture
with WIC scaling, SendInput, UI Automation snapshot, clipboard, `open`, the
five commands and the Computer lab); the pure modules are tested, the cross
build is green; the owner's DPI × monitor matrix on the lab is the gate.
Known limits of this slice: `PrintWindow` black frames fall back to a screen
capture; `focus` reports `foreground_refused` when Windows keeps the focus;
the lab window itself appears in a screenshot of the monitor it sits on.


Scope: `desktop-shell/src/{computer,desktop,capture,input,uia,clipboard,computerlab}.rs`,
the pure modules, the five commands, `ui/computerlab.html`,
`capabilities/computerlab.json`, the `Cargo.toml` features.

Gate: `rustc --test` on the pure modules; `make desktop-shell` green; in the
lab: monitors and windows listed; screenshots of a monitor and of a window
at 100 % and 150 % on two monitors with correct `scale`/`origin`; a click on
a printed crosshair lands where the page drew it; `type` into Notepad and
`snapshot` reads it back; `ctrl+a`; `focus`; clipboard; `open notepad.exe`.

Owner acts: runs the lab and reports the DPI × monitor matrix.

### M2 — End to end (`feat/computer-m2`)

Status 2026-09-18: landed on the branch — `internal/computer` (grant,
catalog), the four routes with `computer.step` audit, kind dispatch on the
shell line, the grants mirror into the shell, Settings ▸ Computer (switch per
principal, recent steps), `packages/pi-computer`. The first real run with an
agent is the owner's.


Scope: `internal/computer/{policy,actions}.go` + tests;
`internal/server/{computer,computer_policies}.go` + tests; `registerAll`;
`lib/computerChannel.js` + kind dispatch in `browserChannel.js`;
`ComputerPage.jsx` + `settingsControls.jsx` + `UserMenu.jsx`;
`packages/pi-computer` (package, extension, `logic.ts`, tests, README); the
`make test-js` line; a paragraph in `docs-site/guide/packages.md`.

Gate: `make ci-scoped` (Go rows: grant missing/off/on, agent vs terminal,
unknown action, no shell; JS: channel and package, the 23-literal grep);
scratch: no grant → 403 with the path to Settings; grant → a screenshot with
the image in the chat, a click, typing, a launched program.

Owner acts: first real run with one of their agents and with a CLI in a
terminal.

### M3 — Visuals, dashboard, docs (`feat/computer-m3`)

Status 2026-09-18: landed on the branch — the capture as "Last capture"
with its source and time, action step labels, the Desktop panel on the
dashboard (`desktop` on the stats route), the public guide, and
`docs/architecture/computer-tool.md`.


Scope: `details.preview` → `ToolCapture`; `stepLabel`; `session_ops.go` +
`Store.EventsOfTypeSince`; `dashboardStats.js`, `DashboardView.jsx`;
`docs-site/guide/computer-tool.md` + `config.mjs`;
`docs/architecture/computer-tool.md`.

Gate: `make ci`; Vale; visual review on a scratch instance.

Owner acts: reads the public page.

### M4 — Evaluation and hardening (`feat/computer-m4`)

Scope: 12–20 Windows tasks in OSWorld's JSON shape under
`docs/benchmarks/computer-tasks/` + `scripts/computer-eval.mjs` against a
scratch shell → pass@1, pass^3, steps, wall-clock, cost; UIA `RuntimeId`
refs + `stale_ref`; Graphics Capture (`Graphics_Capture`, still 0.61) for
windows `PrintWindow` returns black; optional OCR (`Media_Ocr`).

Gate: a report in `docs/benchmarks/` with the first measurement.

Owner acts: picks the tasks.

### M5 — Policy refinements (unscheduled; one branch each, owner-ordered)

(a) window binding + read/act/full tiers; (b) mechanical Ask classes
(`IsPassword`, launching executables, clipboard writes) on the existing Ask
bar; (c) app allowlist; (d) human-activity pause and foreground checks — the foreground check landed 2026-09-18 as ADR-0156 (`foreground_changed`), after the first live run typed into the human's own terminal; the activity pause stays open; (e)
integrity check (UIPI) and a step budget; (f) a Windows Sandbox spike via
`wsb` as a disposable arena. Each amends ADR-0148.

Every milestone: `make worktree NAME=<branch>`, `make ci-scoped` while
iterating, `make close`, fast-forward on `main`, one handoff note (≤ 25
lines), a visual proof on a scratch instance when UI is touched. Deploy is
the owner's, in batches.

## Validation

- Go: `go test ./internal/grant/... ./internal/computer/... ./internal/server/...`.
- JS: `make test-js` (`computerChannel.test.js`, `packages/pi-computer/test`).
- Rust: `rustc --edition 2021 --test src/{geometry,keys,b64,axfmt}.rs`; `make
  desktop-shell`; the rest is the Computer lab on Windows.
- Scratch (M2): `scripts/qa-scratch.sh start computer` + `make desktop-shell`;
  a Pi agent with `pi-computer` in the "This agent" scope and a CLI in a
  terminal; no grant → 403; grant → screenshot, `windows` + `focus` + `type`
  in Notepad, `open` Explorer, `clipboard_write` and paste; off → 403 again;
  one feed line per call.

## Risks

- Unrestricted by design: an agent with the grant is the user at the
  keyboard; screen-content injection succeeds 7–86 % of the time without a
  policy layer (study §1.4). The ADR and the public page say so; M5 answers.
- `SetForegroundWindow` needs the "last input" on our side: `focus` can fail
  while the human uses another app (`foreground_refused`).
- `PrintWindow` returns black for `WDA_EXCLUDEFROMCAPTURE`/GPU windows →
  `capture_failed` in v1, Graphics Capture in M4.
- UIPI drops `SendInput` into elevated windows silently → M5(e).
- DPI: coordinates in image space, physical on screen; `scale`/`origin`
  travel with every capture; the lab tests 100 % and 150 %.
- `details.preview` lands in the session JSONL: capped at 480 px / 60 KiB.
- One desk thread: a 5 s `wait` delays another agent's screenshot.
- A Claude-backed agent assumes the screenshot is the screen and coordinates
  are absolute: the guideline says otherwise.
- ADR numbers collide across parallel sessions: `make adr` picks at commit.
