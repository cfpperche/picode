# Work browser (Phase 3)

Policy: ADR-0128 as amended by 0134 (per-agent tier + origins), split view
ADR-0135. Plan: `docs/plans/desktop-v2.md` (Phase 3). Settings parity is
reviewed by the owner against the reference, one item at a time.

## Next

- **Ask prompt** closes Browser permissions v1 (deferral on the UI thread).
- **Terminal agents as principals** — ADR-0143, steps 1–2 landed; listing + UI
  rows still open (a grant needs a row the human can edit).
- **"New browser tab" button: re-verify in the shell** (no repro in a web
  scratch, 2026-09-15).
- **v2a** unused-site permissions; **v2b** agent history access (needs Ask).
- **v2c** annotations (step 4 is an ADR); **v2d** Windows Hello opener row.
- **v3** WebMCP and Developer mode (raw CDP: ADR, off, `full` only, audited).

## Ask prompt — measured state and the sketch (2026-09-15)

The shell already has the handler (`attach_permission_handler`, btab.rs):
`permission_policy()` is a `Mutex<HashMap<String, bool>>`, the closure maps
the platform kind through `permission_kind_name`, answers from that map, and
reports `btab://permission` so the standings list shows what each site got.
There is **no deferral** — that is the whole of the missing half.

Three edits in `desktop-shell/src/btab.rs`, two in the ACL:

1. `permission_policy()` becomes `HashMap<String, String>` ("allow"/"deny"/
   "ask"); `btab_set_permission_policy` gains `"ask"` beside `allow`/`deny`/
   `default`. Until the dialog offers Ask, no kind can be in this state, so
   the new branch below is unreachable and harmless.
2. In the handler, `"ask"`: `args.GetDeferral(&mut deferral)`, stash
   `(args.clone(), deferral)` under a counter id in a **thread_local**
   `RefCell<HashMap<u64, …>>` (COM objects are apartment-bound and this
   closure runs on the UI thread — the `RECEIVERS` rule), emit
   `btab://permission-ask` `{id, origin, kind}`, and return **without**
   `SetState`. Use fully qualified
   `webview2_com::Microsoft::Web::WebView2::Win32::…` types so the import
   block does not grow. No deferral available → SetState(DENY) as today.
   `None` (no policy) keeps falling through to the platform default — do not
   add a SetState there.
3. `#[tauri::command] pub fn btab_permission_answer(app, id, state, remember)`
   — **sync on purpose** (a sync command runs on the main thread, which is
   where the thread_local can be touched): remove the entry, `SetState`,
   `deferral.Complete()`, optionally remember the kind in the policy map, and
   emit the same `btab://permission` outcome so the list stays truthful.
4. ACL, the usual three edits (`generate_handler!`, `build.rs` commands,
   `capabilities/default.json`) plus the generated `permission/` file.

Then the visible half: Ask as the third option in the per-kind select (and in
the Go vocabulary — `NormalizePermissionDecision` accepts allow/deny today),
plus the prompt surface listening for `btab://permission-ask` (Allow / Block /
Always, the last writing the per-site standing through the API the dialog
already uses).

Verification needs the desktop shell and a page that asks: a local test page
calling `navigator.mediaDevices.getUserMedia({video:true})` in a web tab
triggers `PermissionRequested`. A web scratch cannot see any of it.
Debt to open with this slice: a held request with no answer hangs the site —
add a timeout that denies, or state the risk.
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
  Never — needs the Ask prompt). **v2c**: annotations (below; step 4 is an
  ADR). **v2d**: the Windows Hello passkey opener row (validate the OS URI on
  the machine first).
- **v3**: WebMCP site tools (ADR when the standard lands); Developer mode /
  raw CDP — the reference itself marks it Elevated risk: if it lands it is
  off by default, `full` tier only, audited, warning row, ADR first.
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
- Ask, per the reference, is a policy state we do not offer yet — the dialog
  says so rather than promising a prompt that does not exist.

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
- An HTML popover can never paint over a WebView2 sibling: the options menu
  slides the page down (`MENU_H` in `WebTab.jsx`) instead of flipping z-order.

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

- `btab_layer.toml` (autogenerated) is a dead permission.
- The options menu's page slide is not animated.
- No JS check catches use-before-declaration in a component (the blank-window
  class).
- ADR-0143's endpoint branch still has no test of its own: the resolver's
  decision rows are covered (`TestResolveCallerIsTheHouseIdentity`), the wire
  rows (`term` known/unknown, `agent`+`term` together) are not.
