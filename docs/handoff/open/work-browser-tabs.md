# Work browser (Phase 3)

Policy: ADR-0128 as amended by 0134 (per-agent tier + origins), split view
ADR-0135. Plan: `docs/plans/desktop-v2.md` (Phase 3). Settings parity is
reviewed by the owner against the reference, one item at a time.

## Next

- **v2a** unused-site permissions; **v2b** agent history access (the Ask
  machinery landed 2026-09-15; the history grant is its own path).
- **v2c** annotations (step 4 is an ADR); **v2d** Windows Hello opener row.
- **v3** WebMCP site tools (ADR when the standard lands). Developer mode
  landed 2026-09-16 (ADR-0144).
- The reference's permissions table (Site or pattern × Browsing × Downloads ×
  Uploads + Default row): the data exists, the shape does not.

## Options menu — landed (2026-09-15)

The ⋮ menu matches the reference (ChatGPT Work): Find in page · Print ·
Zoom (−/100%/+/reset) · Take a screenshot · Passwords and autofill › ·
Downloads · History · Clear browsing data · Browser settings. Show device
toolbar and Import cookies and passwords… stay out until they exist.

Two facts the slice established:

- **Everything opens over the page now, by geometry.** Any visible floating
  layer that intersects the tab (the shared vocabulary in
  `floatingLayers.js`) hides the native view and shows the frozen page
  (`btab_preview`, raw PNG over IPC). It began with the options menu
  (2026-09-15) and the "New workspace" dialog cut in half by the page; the
  class-only `.dlg` observer that fixed those missed the command palette and
  the editor's own tab menus, which is the 2026-09-16 sweep.

Find drives the WebView2 Find API (`ICoreWebView2Find` via the environment's
`CreateFindOptions`); Zoom/Print are the controller's `ZoomFactor` and
`ShowPrintUI`; the dialog items ride `browserDialogs.js` into Settings ▸
Browser (option A, owner 2026-09-15).

## Ask prompt — landed (2026-09-15)

Browser permissions v1 is complete. The shell's policy map is tri-state
(`allow`/`deny`/`ask`) keyed by site and kind; the decision table lives in
`desktop-shell/src/permissions.rs` (site over kind over the platform default,
host-tested). `"ask"` holds the platform's request through `GetDeferral` in a
UI-thread map and emits `btab://permission-ask` `{id, tab, origin, kind}`;
`btab_permission_answer(id, state, remember)` is **sync on purpose** (COM
objects on the UI thread) and completes it. A 60 s watchdog, or closing the
tab, denies instead of hanging the page.

| Site entry | Kind entry | Action | Report `standing` |
|---|---|---|---|
| allow / deny | — | SetState(ALLOW/DENY) | true |
| ask | — | defer → prompt | the answer's |
| — | allow / deny | SetState(ALLOW/DENY) | false |
| — | ask | defer → prompt | the answer's |
| — | — | platform default (no SetState) | false |

Answer: Allow / Block answer once; **Always allow** also writes the site's
standing (the shell map now, the store through the report). The store row
carries `standing` (migration 051): the dialog's per-kind rows and the
prompt's Always are policy, every other row is the decision log, and the
load-time push hands back only standings — an Allow-once can never become
permanent. The bar renders between the toolbar and the page (a native
sibling, so it makes room above it, the `MENU_H` pattern).

First live run is the owner's: deploy + restart, set Camera to Ask, open a
page that calls `getUserMedia` in a work-browser tab, answer the bar, reload
to see the remembered standing.
- **Terminal agents** — ADR-0143 **done** (identity, resolver, listing, rows).
  (agent → terminal → unmanaged), the resolver keys `term:<id>` beside the
  bare agent key, the listing returns terminals with a CLI running, the UI
  gives each principal a row and says an outside-PiCode `pi` stays read-only.
  Tests per decision-table row.
- **New-tab button**: an earlier note claimed a pre-existing crash on a main
  scratch; measured again 2026-09-15 and it did **not** reproduce in the web
  scratch (no error, root alive, `#/` reached). Re-verify in the desktop shell
  before touching code — the web path has no `window.__TAURI__`, so the shell
  is the likely home. Same class as the blank window, which no linter catches.
- **v2a**: remove permissions from unused sites (visit recency is in the
  store). **v2b**: agent access to browsing history (Always ask / Allow /
  Never — the Ask machinery landed 2026-09-15, but a first-use history grant
  is its own path). **v2c**: annotations (below; step 4 is an ADR). **v2d**:
  the Windows Hello passkey opener row (validate the OS URI on the machine
  first).
- **v3**: WebMCP site tools (ADR when the standard lands); Developer mode /
  raw CDP **landed 2026-09-16** (ADR-0144: machine setting + `full` tier,
  shell re-check, every call audited; the loopback port stays an env var).
- **Never on WebView2**: third-party cookies, images, embedded content — no
  host API; a control there would be theatre.

## Annotations (backlog, owner-registered 2026-09-15)

Deliberately absent from the settings page: a switch that controls nothing is
a dead control. Build the feature, then the row (Always include / Only when
needed / Never).

1. Annotate mode in the tab: pick an element or draw a rectangle + comment.
2. Capture: `Page.captureScreenshot` with `clip`, plus selector and URL.
3. Store + endpoints: migration, feed event (ADR-0048).
4. Delivery to the agent — a new user→agent input path: ADR before code.
5. The settings row, honoured by the capture step, once 1–4 exist.

## Password manager (finding, 2026-09-15)

Measured in `webview2-com-sys 0.38.2`: the whole password surface is
`Is`/`SetPasswordAutosaveEnabled` and `Is`/`SetGeneralAutofillEnabled` — no
enumeration, no per-entry edit, no import. The SDK ships a runtime-UI opener
where one exists (`ICoreWebView2_6::OpenTaskManagerWindow`) but there is no
password-manager equivalent. Windows Hello and passkeys are OS surfaces.

Two honest paths, both owner-gated: **opener rows** (cheap, no vault) or
**PiCode owns the vault** (DPAPI store, fill injection through CDP, Hello
unlock through WinRT `UserConsentVerifier`, CSV import) — a security-model
decision, ADR before code.

## Permissions — implementation notes (measured in webview2-com-sys 0.38.2)

- `add_PermissionRequested` is on the base `ICoreWebView2`;
  `PermissionRequestedEventHandler` ships in webview2-com's `callback`.
- Args carry `PermissionKind()`, `Origin()`, `SetState(state)`,
  `GetDeferral()` — the deferral is what makes Ask work.
- States `COREWEBVIEW2_PERMISSION_STATE_{DEFAULT,ALLOW,DENY}`; kinds
  AUTOPLAY, CAMERA, CLIPBOARD_READ, GEOLOCATION, MICROPHONE, NOTIFICATIONS
  (+ sensors, MIDI, local fonts, file system).
- Standings live on `ICoreWebView2Profile4::SetPermissionState(kind, origin,
  state)`; the shared work profile means one call covers every tab and the
  ones created later.
- Ask is a live policy state: `GetDeferral` holds the request, the tab
  answers through `btab_permission_answer`, and a 60 s watchdog denies an
  unanswered one (the section above).

## Traps (paid for, keep them paid)

- **A new shell command is THREE edits**: `generate_handler!` (main.rs),
  `build.rs`'s `AppManifest::new().commands(&[…])` and
  `capabilities/default.json` — miss the second and every invoke dies with
  "not allowed by ACL" while the page swallows it behind a `.catch`. Cost:
  four features (autofill, clear data, open destinations, downloads) shipped
  dead (2026-09-15).
- Edit in the worktree. UI edits that land in the root checkout leave scratch
  and deploy testing stale bundles (happened three times).
- `toast(msg)` defaults to `err`; success needs `toast.ok`.
- `.web-tab-toolbar button` beats bare class selectors — prefix toolbar
  button overrides with `.web-tab-toolbar`.
- `um-popover` is styled for the user menu; do not reuse it elsewhere.
- Data-URL downloads are blocked inside WebView2: write files from Rust and
  toast the path.
- Tauri commands that create or drive webviews must be async (a sync command
  runs on the main thread). A command that must touch a thread-local —
  the deferral map — is the exception, and sync for that reason.
- `webview2-com-sys 0.38` pins windows/windows-core 0.61; the completed
  handlers come ready-made.
- COM interfaces are not `Send`: keep event receivers on the UI thread (the
  `RECEIVERS` thread-local in `btab.rs`), never in Tauri state.
- **An HTML layer can never paint over a WebView2 sibling.** The rule is
  geometry, not a memory of which overlays exist: `floatingLayers.js` collects
  every visible layer from the shared vocabulary (`OVERLAY_SELECTORS` in
  `web/shared/domain/overlayAudit.js` — role+state for Radix, the app's own
  classes for the rest) and the tab hides the native view when any of them
  intersects its rectangle, showing the frozen page behind (`btab_preview`).
  The old `MENU_H` slide is gone (2026-09-16: the owner's editor tab menu came
  up invisible under the page — a class-only list never saw the palette
  either). A new floating surface belongs in the shared list, or both the hide
  rule and the clipping audit lose sight of it.
- **A sized popup must stay a window.** `window.open(url, name,
  "width=…,height=…")` carries window features; adopting it as a tab severs
  `window.opener`, and the OAuth popup dead-ends on a blank bridge page
  (x.com + Google, 2026-09-15). `on_new_window` allows feature-sized requests
  to the runtime and adopts only unsized ones as tabs.
- **A windowed WebView2 cannot be painted over.** Any HTML overlay that
  should cover a work tab must hide the native view first: the menu freezes
  the page into a still (`btab_preview`) and hides it; an open `.dlg` hides
  it through `appOverlays.js`. Bounds tricks (sliding the page) are the old
  workaround and are refused now.
- **The agent split renders WebTabSurface too.** A prop added to the tab
  instance must be added there as well (2026-09-16: `onBrowserSettings` was
  tabs-only, so every settings link in the split's ⋮ menu silently no-op'd
  through `?.`). Same for parking: the tab hides on tab switch AND on
  route change (`onPane`), or the page covers the settings views.
- **A grant entry the matchers read literally opens nothing.** `*` was the
  live case: the domains field accepted it (the charset allows `*`), both
  matchers compared it to the host, nothing matched, and nothing said so.
  Since 2026-09-16 `*` means any http/https host in both matchers, and
  `browserDomains.js` says in the field what each entry covers — including
  "matches nothing" for the shapes that still do (`*example.com`, `*.`, `.`).
- **`rustc --edition 2021 --test src/origins.rs` runs that module's tests
  without cargo** (the host `cargo test` is broken here); it needs the test
  module's `use super::{…}` to list what the tests call.

## Scope (v1): Pi only, and read for a TUI

The tool is a pi package, so reach follows install scope: **This agent** is
private to a managed agent, **This machine** (`~/.pi/agent`) and **this
project** are visible to a plain `pi` too. Identity is an assertion, not proof
(ADR-0134) — a gate on a missing id would be cosmetic, so v1 documents the
behavior instead of faking a boundary. Owner: validate in use, revisit scope
after.

**Name collision (found 2026-09-13):** npm already has a `pi-browser`
("Playwright-backed pi extension", keyword `pi-package`). Ours is local-only,
so nothing breaks — but the Packages gallery searches npm, and that package's
tool is also called `browser`. Publishing ours needs a name decision, and the
gallery hit is worth a look before anyone installs it expecting this one.

## Debts

- The work browser has no `docs/architecture/<subsystem>.md` file: its shape
  lives across ADR-0128/0132/0134/0135/0143/0144 and the plan. Worth one file
  the next time a browser slice lands.
- The Ask deferral/answer/watchdog path has never run outside Windows:
  `cargo xwin build` plus the pure decision table are the evidence here; the
  owner's live check is the first run.
- An Ask waits only in the tab that asked: a request from a tab the user is
  not looking at is denied when the 60 s watchdog fires. That is the
  limitation the Ask bar's own placement makes honest — nowhere else shows a
  waiting request.
- The COM capture + native hide half of the still/overlay path has no
  automated test (Windows-only); the JS decode gate is unit-tested
  (`lib/previewStill.js`).
- **Device toolbar** and **Import cookies and passwords…** are the two
  reference options-menu entries still missing; import has no WebView2 API
  (cookies would go through CDP).
- `btab_layer.toml` (autogenerated) is a dead permission.
- The options menu's page slide is not animated.
- No JS check catches use-before-declaration in a component (the blank-window
  class).
- ADR-0143's endpoint branch still has no test of its own: the resolver's
  decision rows are covered (`TestResolveCallerIsTheHouseIdentity`), the wire
  rows (`term` known/unknown, `agent`+`term` together) are not.
