# Work browser (ADR-0128, ADR-0132, ADR-0134, ADR-0135, ADR-0143, ADR-0144, ADR-0146, ADR-0152)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per
> subsystem). Edit here; the index only links.

The work browser is a **real Chromium (WebView2) inside the desktop shell**:
tabs the human browses, and — when a grant says so — the same pages an agent
can read and act on through a closed verb catalog. It is not an iframe, not a
screenshot pipeline and not a second process: the shell window owns the
native webview children, and everything the app draws around them is
geometry, not stacking.

An **annotation mode** (v2c, ADR-0152) rides the same tabs: the human pins
notes on live elements, optionally proposes style changes, and one Send
delivers the set to an agent through the prompt door that already exists.

Plan and the running list: `docs/plans/desktop-v2.md` (Phase 3),
`docs/handoff/open/work-browser-tabs.md` (dated traps — read before editing
`btab.rs` or `annotate.js`).

## Who decides what

| Question | Decided by | Where |
|---|---|---|
| Who is calling | daemon | `internal/grant.Key`: an agent id, else the agent bound to its terminal, else `term:<terminal id>` with no grant (ADR-0184), else nobody (ADR-0134, ADR-0143) |
| May they drive their own split | the binding is the consent: an identified caller is act on any http(s) URL in that tab; closing the split revokes | `internal/browser/session.go` (ADR-0172). A caller with no identity stays read-only on the tab on screen (ADR-0134). Raw CDP still needs the full tier and Developer mode (ADR-0144) |
| Which tab a command runs in | the split bound to that principal, opened by the agent if it is missing | `web/browser/src/lib/sessionBrowser.js` + the pane's binding (ADR-0135, ADR-0172) — never another tab |
| Which actions exist | the closed verb catalog, mirrored | `internal/browser/verbs.go` ↔ the shell's catalog; raw CDP needs the machine's developer mode **and** the full tier, and every call is audited (ADR-0144) |
| Site permissions (camera, mic, …) | the shell's policy map, tri-state per site × kind | `desktop-shell/src/permissions.rs`; the store logs the decision and holds standings |
| May an agent read history | the daemon, only when the human turned it on | ADR-0146 (`historyAccess`) |
| What reaches an agent from an annotation | the store row + staged files + the prompt door | ADR-0152 (see **Annotations** below) |

## The line

```
agent tool (pi-browser) → POST /api/browser/tool
  → browser.Hub.Dispatch(Command{Kind, Method, Params, Principal, Timeout})
  → the shell page's one EventSource on /api/browser/stream
  → web/browser/src/lib/browserChannel.js dispatches on `kind`
  → invoke("btab_cdp_call" | "btab_screenshot" | …)  (WebView2 COM, UI thread)
  → POST /api/browser/result  → the tool answer
```

One stream, one port, one credential (ADR-0132) — the computer tool
(ADR-0148) and annotation-free browser actions share it, dispatching on
`kind`. Commands are neither replayed nor persisted: a request lives only as
long as the tool call waits. A shell that reconnects opens a new stream and
becomes the active one.

**Two channels, one wire.** Permission asks travel the same stream in the
other direction (`btab://permission-ask`, below). Do not invent a second
channel for a new push: the page already listens.

## Tabs

- A work tab is a **child `Webview` under the main window** (`WebviewBuilder`
  inside `ensure`), never a `WebviewWindow` — `get_webview_window` answers
  "no such tab" for every tab that exists (this cost two deploys and two
  wrong id theories on 2026-09-18). Commands resolve tabs through
  `find_webview(&app, id)`, which accepts both id shapes the app uses
  (`w:3` from the strip's native id and `3` from the React one).
- **Parking is two conditions**: the native view hides on tab switch **and**
  on route change (`onPane`), or the page covers the settings views. The
  agent split renders `WebTabSurface` too — a prop added to the tab instance
  must be added there as well (a settings link silently no-op'd through `?.`
  for a day).
- **A sized popup stays a window**: `window.open(url, name, "width=…")`
  carries window features and keeps `window.opener` (OAuth); only unsized
  requests are adopted as tabs.
- Tabs carry a title, a favicon and a history stack; `btab_meta` is the
  receipt the UI paints from. Downloads land in the shell's download folder
  and are announced on the feed.
- **Ctrl+R / F5 on the chrome reloads the page on screen.** The chrome
  webview turns WebView2's own accelerators off so a terminal keeps
  Ctrl+R (readline) and a work tab does not reload PiCode. The chrome
  then reloads the visible work webview (`btab_reload`) or PiCode itself.
  Page webviews keep the engine accelerators, so a focused page still
  reloads itself.

- **Header globe** (shell row): click opens a split bound to the selected
  agent or agent-CLI terminal — the same door as the pane's **Open browser**.
  Shift+click, and the right-click **Open in new tab**, mint a tab with no
  agent. A second click while that split is already open does not mint
  another webview. Right-click is this menu, not the generic window menu.

## Live native layers (ADR-0161)

The new shell hosts the trusted React document in `main-content`, a
transparent native child above the work pages. Its Windows region is the
window minus visible page rectangles plus the rectangles of floating HTML
layers. The same geometry controls paint and pointer input: the page stays
visible and interactive through the holes. A modal backdrop adds its whole
region without freezing the page underneath.

`web/browser/src/lib/nativeLayers.js` invalidates its region cache when viewport
width, height or device pixel ratio changes, even with no active native page
or floating overlay. This keeps the full chrome region current on maximize
and restore. It observes geometry and serializes the
`chrome_layers` IPC. `desktop-shell/src/layers.rs` validates the caller and
bounds, applies the native region, and restores the chrome's stacking order
after a page is shown. Native page ancestors clear their CSS backgrounds;
all overlays remain the real React components, in the same document, with
no DOM cloning or page injection.

The bridge is restricted to the trusted `main-content` webview and the
existing Tauri origin ACL. External pages never receive layer permissions.
Tab/route changes still hide inactive pages. Explicit screenshots and
annotation captures retain their own APIs; the new overlay path calls none
of them. An older shell without the injected protocol marker keeps the
legacy frozen-backdrop path until the shell is upgraded too.

The shared overlay vocabulary still drives the geometry audit; native
composition additionally includes modal backdrops, tooltips and focus-edge
controls. A native screenshot is required to verify Windows stacking:
`window.__picodeOverlayAudit()` alone cannot see sibling WebViews.

## Permissions: site × kind, tri-state

`allow` / `deny` / `ask`, resolved **site entry over kind entry over the
platform default**. `"ask"` holds the platform's request through
`GetDeferral` in a UI-thread map and emits `btab://permission-ask`
`{id, tab, origin, kind}`; the bar renders between the toolbar and the page
(a native sibling, so it makes room above the page) and
`btab_permission_answer(id, state, remember)` completes the request — **sync
on purpose**, COM objects on the UI thread. A 60 s watchdog, or closing the
tab, **denies** rather than hanging the page.

| Site entry | Kind entry | Action | Standing reported |
|---|---|---|---|
| allow / deny | — | `SetState(ALLOW/DENY)` | true |
| ask | — | defer → prompt | the answer's |
| — | allow / deny | `SetState(ALLOW/DENY)` | false |
| — | ask | defer → prompt | the answer's |
| — | — | platform default (no `SetState`) | false |

Standings live on the shared profile (`SetPermissionState`); the store row
carries `standing`, and the load-time push hands back **standings only**, so
an Allow-once can never become permanent. A request from a tab the human is
not looking at is denied when its watchdog fires — that limitation is what
the bar's own placement makes honest.

Never on WebView2: third-party cookies, images, embedded content — no host
API. A control there would be theatre.

## Annotations (v2c, ADR-0152)

The page stays **live**. The overlay (hover outline, numbered pins, the
anchored card, the chips) is **DOM injected into the page** inside a closed
shadow root; the chrome is the strip that replaces the URL bar.

| Piece | Where |
|---|---|
| The injected script and its guards | `desktop-shell/src/annotate.js`, `annotate.rs` (script constants + tests) |
| Arming, the navigation hook, the pull door | `desktop-shell/src/btab.rs` (`btab_annotate_mode`, `…_clear`, `…_state`, `…_overlay`) |
| The strip, the Send path, the crop | `web/browser/src/components/WebTab.jsx`, `AnnotateStrip.jsx`, `lib/annotate.js` |
| The store, the staged files, the note | migration `browser_annotations`, `internal/server/browser_annotations.go` |

Rules that are load-bearing:

- **The page→host channel is not a foundation; the host→page one is.** The
  strip polls `btab_annotate_state` (ExecuteScript's return value rides the
  command's own result — no page-side bridge) while the mode is on; the page's
  `postMessage` stays the fast path. A page whose bridge is dead still lights
  the strip up (owner 2026-09-19: the card and chips worked while every
  message vanished — `SetIsWebMessageEnabled` is applied at webview creation,
  never after the document exists).
- **The picture shows the page, not our UI.** Pins, chips and the card are DOM
  inside the page, so the crop hides the overlay around the capture
  (`btab_annotate_overlay`) and always shows it again.
- **A preview is a mutation, and the note must not read it back.** The style
  inspector applies live inline styles (`!important`, or a page rule marked
  `!important` swallows the preview silently). The item snapshots the
  element's computed values at pick (`styles0`) and its own inline values
  (`inline0`); the note carries the original **and** a `/* proposed */`
  section. Cancel, the trash, the chip's ×, the menu's Remove, Clear and
  leaving the mode all restore through `inline0`.
- **One Send, one package**: one row per pin, a note + a crop per pin, and one
  message through the prompt door carrying the paths (`batchMessage`).
  Delivery resolves the bound session, never "the first terminal running".
- Keystrokes in the card must not reach page hotkeys: the shadow root stops
  the shared event list before the page sees a retargeted host `div`.

## Developer mode, downloads, history, openers

- **Developer mode** (ADR-0144) is a machine setting plus the `full` tier;
  the shell re-checks and every raw CDP call is recorded
  (`/api/browser/developer/audit`).
- **Downloads** ask first when the human turned Ask on; data-URL downloads are
  blocked inside WebView2, so files are written from Rust and the toast names
  the path.
- **History** is the human's: chrome-only until `historyAccess` is turned on
  (ADR-0146), and clear/delete live in Settings ▸ Browser.
- **Openers** for things WebView2 does not own (passwords, Windows Hello
  sign-in) are rows that open the OS surface. No vault, no fake enumeration.
- **Installed webapps** (ADR-0147) reuse the browser's engine with their own
  chrome — see `docs/handoff/open/installed-webapps.md`.

## Evidence, and what only Windows can prove

| Layer | Evidence |
|---|---|
| Daemon (policy, verbs, hub, domains) | Go tests: `internal/browser/*_test.go`, the decision tables in `policy_test.go` and `domains_test.go` |
| The shell's re-checks and catalogs | host tests where they are pure (`permissions.rs`, `origins.rs`, `annotate.rs`, `preview.rs` — run with `rustc --edition 2021 --test src/<file>.rs`; host `cargo test` is broken here) and `cargo xwin build` for the whole shell |
| The page's logic | `web/browser/src/lib/*.test.js` (channel, prefs, annotations, overlay vocabulary, the legacy still/cover decisions) |
| The injected overlay | a real browser harness (`scripts/fixtures/annotate-hotkey.html` + the script injected) — behavior, screenshots, key shields |
| Windows-only paths | the owner's live run: the COM capture and native hide behind `preview.rs`'s decisions, the Ask deferral and its watchdog, the navigation re-inject hook. The capture's decision rows (valid, empty, failed, not painted, timeout, tab gone) and the hide/restore table (`coverDecision`) are host-tested; the COM calls and the on-screen stacking are not |

## Traps (architecture-level; the dated list is in the topic file)

- A tab is a child `Webview`, never a `WebviewWindow`.
- COM interfaces are not `Send`: event receivers stay on the UI thread
  (`RECEIVERS` thread-local in `btab.rs`), never in Tauri state.
- Tauri commands that create or drive webviews must be `async`; a command that
  must touch a thread-local (the deferral map) is the exception, and sync for
  that reason.
- **A new shell command is three hand edits plus two generated sets**:
  `generate_handler!` (`main.rs`), `build.rs`'s
  `AppManifest::new().commands(&[…])` and `capabilities/default.json`, plus
  `permissions/autogenerated/<cmd>.toml` and `gen/schemas/*`. Miss the second
  and every invoke dies with "not allowed by ACL" while the page swallows it
  behind a `.catch` — four features shipped dead that way (2026-09-15).
- The built UI and a scratch instance can disagree: `make web` is
  stamp-guarded, and a scratch embeds the bundle at build time. Remove
  `var/web.built` and grep the *served* asset when a UI change looks ignored.
