# Work browser (Phase 3)

Policy: ADR-0128 as amended by 0134 (per-agent tier + origins), split view
ADR-0135. Plan: `docs/plans/desktop-v2.md` (Phase 3). Settings parity is
reviewed by the owner against the reference, one item at a time.

## Next

- **v2a** unused sites — landed; **v2b** agent history — landed (ADR-0146).
- **v2c** annotations — **complete**. Step 3 landed (store + endpoints +
  staged files, ADR-0152); step 4 landed 2026-09-18 (strip with Send N,
  numbered pins, anchored card, chips, one-package Send,
  `btab_annotate_clear`); the mode survives a page load (a
  `NavigationCompleted` hook re-injects the script, 2026-09-19); **step 5
  landed 2026-09-19** — the style inspector (six rows, live preview,
  original → proposed in the note) and the annotation-screenshots row
  (always / ask / never, honoured by the capture). **v2d** Windows Hello
  opener row — landed (the OS screen, not a vault).
- **v3** WebMCP site tools (ADR when the standard lands). Developer mode
  landed 2026-09-16 (ADR-0144).
- The reference's permissions table turned out to be **our Site settings dialog
  transposed** (a row per kind with the tri-state, a per-site log below): the
  missing piece was the "+ Add" and it landed 2026-09-16 (`browserSiteSchema`).
  The Browsing/Downloads/Uploads *columns* would each need new enforcement —
  Browsing is the grant's `{domains, tier}`, Downloads is one global Ask,
  Uploads has no file-chooser hook — so the columns stay out until someone wants
  the gate behind them (ADR-0144's shape, not a restyle).

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
- **v2a**: remove permissions from unused sites (visit recency is in the store).
- **v2b** agent history — landed (ADR-0146: allow/never by default, answered by
  the daemon; the first-use prompt is its own decision, named in the ADR).
- **v3**: WebMCP site tools (ADR when the standard lands); Developer mode /
  raw CDP **landed 2026-09-16** (ADR-0144: machine setting + `full` tier,
  shell re-check, every call audited; the loopback port stays an env var).
- **Never on WebView2**: third-party cookies, images, embedded content — no
  host API; a control there would be theatre.

## Annotations — the target, from the owner's screenshots (2026-09-18, 18:19)

Slices 1 and 2 landed and the owner **rejected them as buggy**: the frozen
picker is gone and the in-page mode exists, but it does not behave like the
reference. Three screenshots were sent showing the *reference* — this is the
spec, to be built literally, not interpreted:

1. **Chrome strip**, replacing the URL bar while the mode is on: a close ✕
   (exit), a trash (discard every annotation), three small icons, and
   **"Send N"** — the count of pending annotations rides the button.
2. **Multiple annotations**, each with a **numbered pin** on its element
   (1, 2, 3 …), the selected one outlined. They accumulate; nothing is sent
   until Send.
3. **The anchored card** at the element: a small icon, the text input, a
   trash (discard this one), a mic, **Cancel** and **Save** — a small white
   rounded card beside or below the element, not a full-width bar.
4. **After Save it collapses to a small chip** on the element showing the
   typed text, with a "…" menu and an "×" to remove it.
5. On Send the whole set becomes **one context** ("5 annotations" in the
   reference's chat) — for us: one package to the agent terminal through
   ADR-0152's prompt door, carrying every pin's selector, styles and crop.

What exists today: the in-page shadow-DOM overlay (hover, pick with outline
and pin, Esc), a single anchored box with Send/Cancel, one POST per Send with
the viewport crop, and delivery through the prompt door. What is missing is
everything above: the strip with the count, the numbers, the accumulating
set, the collapsed chip, and the card's own controls.

The specific bugs were named only as "cheio de bugs"; the next session
reproduces the flow against these screenshots and fixes each difference it
finds, with the owner's print as the acceptance.

## Annotations — the ChatGPT-Work shape (owner's reference, 2026-09-18)

The owner rejected the frozen-panel picker outright and specified the
reference exactly (four screenshots): the page stays **live**, the annotation
UI lives **inside the page**, and the editor offers the element's computed
styles with live preview (Text color / Background / Opacity / Font
family·Size·Weight) plus one Send. Two conclusions from the study and the
crate inspection: no CEF — `webview2-com-sys 0.38.2` already ships
`AddScriptToExecuteOnDocumentCreated`, `ExecuteScript`,
`add_WebMessageReceived`, `SetIsWebMessageEnabled`, `PostWebMessageAsJson` —
and no CDP, so this path never meets the tier gate that killed the frozen
picker.

**Slice 1 — accepted live 2026-09-18** ✓: the toolbar chip + `Ctrl+.`, the
injected shadow-DOM overlay (hover highlight, click to pick with outline and
pin, Esc), and the pick reaching the chrome over the WebMessage channel. The
owner's screenshot shows the blue outline and the pin on the live GitHub page.

Remaining: **slice 2** the comment box anchored to the element (the payload
already carries selector/html/rect/styles), **slice 3** the style inspector
with live preview and original→proposed in the package, **slice 4** the chrome
strip ("Annotating · host", trash, Send), multiple annotations and one Send.

**Trap (paid 2026-09-18, twice):** a work tab is a child **Webview** under the
main window (`WebviewBuilder` in `ensure`), never a `WebviewWindow` —
`get_webview_window` answers "no such tab" for every tab that exists, which
cost two deploys and two wrong id theories before anyone read `ensure`.

## Annotations (backlog, owner-registered 2026-09-15)

Deliberately absent from the settings page: a switch that controls nothing is
a dead control. Build the feature, then the row (Always include / Only when
needed / Never).

1. Annotate mode in the tab: pick an element or draw a rectangle + comment.
2. Capture: `Page.captureScreenshot` with `clip`, plus selector and URL.
3. Store + endpoints: migration, feed event (ADR-0048). — **landed 2026-09-18**
   (migration 055, `browser_annotations`, the two staged files, ADR-0152).
   The picker (1) and the capture (2) remain.
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

- **The built UI and the scratch can disagree.** `make web` is stamp-guarded
  (`var/web.built`), and a scratch embeds the bundle at build time — so a UI
  edit can be "built" while the running page is one revision older. It bit me
  on 2026-09-17: a leftover `setPreviewBusy` call in an old bundle threw
  `ReferenceError` and blanked the app, and I spent a while blaming the new
  code. `rm -f var/web.built` before the scratch build when a UI file changed,
  and grep the *served* asset for a string only the new code has.

- **A new shell command is THREE hand edits plus TWO generated sets**: `generate_handler!` (main.rs),
  `build.rs`'s `AppManifest::new().commands(&[…])` and
  `capabilities/default.json` — miss the second and every invoke dies with
  "not allowed by ACL" while the page swallows it behind a `.catch`. Cost:
  four features (autofill, clear data, open destinations, downloads) shipped
  dead (2026-09-15). The generated sets travel in the same branch:
  `permissions/autogenerated/<cmd>.toml` and `gen/schemas/*` (the latter
  went stale unnoticed for `btab_annotate_clear`, 2026-09-18).
- Edit in the worktree. UI edits that land in the root checkout leave scratch
  and deploy testing stale bundles (happened three times).
- `toast(msg)` defaults to `err`; success needs `toast.ok`.
- **A preview is a mutation, and the note must not read it back.** The style
  inspector applies live inline styles (`!important`, or a page rule marked
  `!important` swallows the preview silently), so the item snapshots the
  element's computed values at pick time (`styles0`) and its own inline
  values (`inline0`). Reading `getComputedStyle` when the payload is built
  returns the preview — the note would then tell the agent "the page looks
  like this" about the one thing it does not. Every exit that discards an
  annotation (Cancel, the card's trash, the chip's ×, the ⋯ menu's Remove,
  Clear, leaving the mode) restores through `inline0`, and two pins on one
  element re-apply each other's proposals instead of wiping them
  (owner 2026-09-19).
- **A settings call after the document exists is too late.**
  `SetIsWebMessageEnabled` is applied at webview creation now (`apply_scripts`,
  beside `SetIsScriptEnabled`): arming annotate mode called it *after* the page
  had loaded and the page's `chrome.webview.postMessage` stayed dead — the
  card, the pins and the chips all worked (they are the page's own DOM) while
  every message vanished and the strip never lit up (owner 2026-09-19).
- **The page→host channel is not a foundation; the host→page one is.** The
  annotate strip polls `btab_annotate_state` (ExecuteScript's *return value*
  comes back on the command's own result, so it needs no page-side bridge)
  every 1.5 s while the mode is on. The page's `postMessage` stays the fast
  path; the pull is what makes Send work when the bridge is dead.
- The page keeps its `__picodeAnnotateV1` handle when it leaves the mode:
  the pull answers `{kind:"off"}` for "this page turned itself off" and `""`
  for "no script here at all" (a navigation in flight). Those two must not
  read the same, or the strip mirrors the wrong one.
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
- **The domains field's hint is a third reading of the rule.** Two matchers
  enforce it (`internal/browser/domains.go`, `desktop-shell/src/origins.rs`);
  `browserDomains.js` only describes it, and is tested as a description. If
  the matchers change, that file lies first — and the lying is cosmetic, never
  a grant.
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

- Batch Send pastes through the prompt door, which takes at most 4 files
  per call — one note stages 2 files (note + crop), so 3+ notes save fine
  but do not deliver in one paste. Chunked pastes vs. one combined note vs.
  raising the cap is a protocol decision (owner) once multi-note Send is
  exercised live.
- [x] The work browser's `docs/architecture/` file — **paid 2026-09-19**:
  `docs/architecture/work-browser.md` (who decides what, the line, tabs, the
  overlay rule, permissions, annotations, evidence, architectural traps).
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
- [x] `btab_layer.toml` (autogenerated) was a dead permission — **deleted
  2026-09-19** (stale since the command left `build.rs`; the schemas regenerate
  without it).
- No JS check catches use-before-declaration in a component (the blank-window
  class).
- ~~v2d's row has never been clicked on Windows~~ — **accepted 2026-09-17**:
  the owner clicked it and Windows opened Sign-in options. The pure allowlist
  tests (`rustc --edition 2021 --test src/external.rs`), the JS payload test
  and the cross-build remain the always-on evidence.
- ADR-0143's endpoint branch still has no test of its own: the resolver's
  decision rows are covered (`TestResolveCallerIsTheHouseIdentity`), the wire
  rows (`term` known/unknown, `agent`+`term` together) are not.
