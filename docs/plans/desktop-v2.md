# Desktop v2 — the Rust/Tauri shell (ADR-0120)

Status: **Phase 1 in build** (shell skeleton). Owner approved the direction on
2026-09-11; this document is the canonical version of the approved plan.

## Vision

A work application for Windows: an own window (Tauri 2 + WebView2) rendering
the PiCode UI served by the daemon in WSL, a tray with status and actions, and
a browsing context isolated from the personal browser — where **human and
agents** act, through CDP. The Go daemon in WSL remains the single source of
truth; the shell is a client and a supervisor, never a second backend.

## Phases

- **Phase 1 — skeleton (this branch):** window loading the address from
  `server.json` (discovered through `wsl.exe`), tray (Open/Quit),
  single-instance, offline fallback. Windows cross-build from WSL
  (`cargo-xwin`, `x86_64-pc-windows-msvc`). Spike-gate before calling it done:
  pairing in the embedded profile, mkcert, xterm.js/WebGL, SSE, native
  notification.
- **Phase 2 — management inside the shell (owner-corrected 2026-09-11, in
  build):** WSL control lives **only in the shell**; the daemon is untouched —
  no routes, no app, no protocol, and a native-Linux user has no such
  feature. The shell drives the tested Go CLIs as subprocesses (the
  `provision` pattern: the binary inside the distro as a tool, ADR-0020):
  **Overview** renders `picode-desktop.exe disk --json` (both halves, held,
  consumers) in a local page (`ui/disk.html`); **Give back** runs
  `disk-compact --yes --json` (readiness interlock + stop/convert/start, the
  Go path verified live) with plan/running/measured in the window;
  **`--json` added to disk-compact** (Go, additive). **Rejected in review**
  (owner): porting the keepalive to Rust and retiring the Go tray — the two
  coexist, amending ADR-0120's "supervisor" wording (client + subprocess
  orchestrator). Sessions: A — Overview (this branch); B — Give back in the
  window; C — `picode clean` prunes + `.wslconfig` editing.
- **Phase 3 — Embedded Work Browser (owner-corrected 2026-09-13):** not an
  external Chrome — a **browser panel embedded in the shell** (child
  WebView2s, the Tauri multiwebview the shell already carries behind the
  `unstable` feature), tabs + address bar of our own, one shared persistent
  profile under `%LOCALAPPDATA%\PiCode\WebView2`. Agents reach the SAME
  pages the human sees through CDP exposed to the daemon over the
  authenticated channel; the **browser policy** (domains × actions per
  agent) applies host-side and is visible and revocable in the UI.
  Benchmark: the ChatGPT desktop Work browser. `ext/` + `browserhost` +
  ADR-0043 deprecated here. CEF (old Phase 4) is dropped — the spike
  (below) removed its reason to exist.
- **Phase 4 — dropped (2026-09-13).** Embedded CEF was conditional on
  WebView2 failing the browser spike. It did not fail.

## Decisions taken

- **Tauri 2 + WebView2** (ADR-0120): the Chromium engine, Microsoft's
  Evergreen runtime, Rust bindings; CEF is deferred to Phase 4.
- **CDP instead of the extension**: the extension existed because Chrome
  blocks CDP on the user's default profile; in a browser the product launches
  and owns, CDP over a pipe covers navigate/read/act with no open port. The
  extension's trust model (visible, scoped, revocable) becomes shell policy.
- **Cross-build from WSL** with `cargo-xwin`; the default CI gates do not
  require Rust (`make desktop-shell` is a dedicated target).

## Phase 1 risks / spikes

| Risk | Plan |
|---|---|
| Google blocks login in embedded views | Try Gmail in the embedded profile; workaround: UA override or external login |
| xterm.js/WebGL in WebView2 | Terminal render with and without acceleration |
| Feed SSE in webview | Live feed for 30 minutes |
| Pairing in the embedded profile | QR/token once; session persists under `%LOCALAPPDATA%` |
| Duplicated keepalive during transition | Go tray and shell coexist; one-wins in Phase 2 |

## Spike run — 2026-09-11 (evidence + owner checklist)

| Spike | Status |
|---|---|
| Feed SSE (server side) | **PASS** — verified live: `GET /api/events` streams (`event: hello`, bootId + latest); the UI updates without reload. |
| Native notification | **Implemented** (tray item **Test notification**); toast click-through pending owner. Windows shows toasts for unpackaged apps only with a Start Menu shortcut carrying the app identity — a silent drop is the Phase 2 installer requirement. |
| mkcert/HTTPS | **PASS** — owner: no certificate warning, UI over `https://localhost:8445` (2026-09-11). |
| Pairing | **PASS** — owner: UI fully usable, no pair screen (loopback auto-pairs, ADR-0049). |
| xterm.js/WebGL | **PASS** — owner: terminal renders and types, including the live session (2026-09-11). |

**Phase 1 closed 2026-09-11** — five of five spikes settled (one pending the
toast click-through, which only changes the Phase 2 installer scope).

## Phase 3 engine spike — 2026-09-12/13 (lab shipped on `feat/browser-lab`)

The lab (tray → **Browser lab**): a two-webview window — local control strip
(URL, back/forward/reload, Chrome-UA toggle) over a browsed page — built to
answer the engine questions before the product decides panel-vs-tab.

| Test | Result |
|---|---|
| Page paints; no event-loop deadlock | **PASS** after moving webview creation out of the tray handler into `setup` (in-handler creation froze every window — the v1.0 lesson) |
| GitHub login in a fresh profile | **PASS** |
| Persistence across shell restarts | **PASS** (disk profile; quit ≠ logout) |
| **Google sign-in + OAuth (Continue with Google → GitHub) with WebView2's own UA** | **PASS** — the documented embedded-browser block did not trigger |
| Profile sprawl | Fixed — one explicit profile, `%LOCALAPPDATA%\PiCode\WebView2`, shared by every webview |

Consequences: **WebView2 stays the engine** for the work browser; CEF and the
~200 MB Chromium are out. The Chrome-UA override stays in the lab (and becomes
a per-domain fallback lever in the product) because Google's block is
risk-based and machine-variable — one passing machine is evidence, not a law.
### Work Browser parity spec (benchmark: ChatGPT desktop "Work" browser)

Owner requirement 2026-09-13: **the PiCode user gets the same experience as
the ChatGPT Work browser**. Source: owner screenshots 2026-09-12/13 + the
vendor's docs. Every item is a requirement for Phase 3 unless marked (v2).
Mapping to our stack noted inline; slices 1–4 at the end of this section.

**Browser surface (the tab):**
- [ ] Browser pages open as **editor tabs** with a Chrome-style strip:
  per-tab favicon + title, close button, new-tab button (slice 1)
- [ ] Address bar: placeholder "Search or enter a URL"; Enter navigates
- [ ] Back / forward / reload cluster (slice 1)
- [ ] Empty state: "Start browsing — Enter a URL to open a page" — one line
  + the action, never a blank well (slice 1)
- [ ] History dropdown from the address bar (typed URLs first) (slice 3)
- [ ] New tab / close per tab; target=_blank and OAuth popups adopt as new
  tabs, not external windows (slice 1; WebView2 NewWindowRequested)
- [ ] Find in page (WebView2 Find API, our highlight UI) (slice 1)
- [ ] Print (ShowPrintUI) and Zoom (± / 100% / reset, ZoomFactor) (slice 1)
- [ ] **Take a screenshot** button (CapturePreview) — also what the agent's
  screenshot tier uses (slice 2)
- [ ] **Device toolbar**: responsive viewport — Dimensions dropdown,
  width × height, zoom %; off by default (slice 3; controller bounds +
  CDP Emulation)
- [ ] Detach to window / panel⇄tab (v2 — editor tab is the v1 surface)

**Options menu (the ⋮ cluster on the bar):** Find in page · Print · Zoom ·
Show device toolbar · Take a screenshot · Import cookies and passwords… ·
Passwords and autofill › · Downloads · History · Clear browsing data ·
Browser settings (opens Preferences ▸ Browser) — every entry maps to an
item above.

**Settings ▸ Browser (Preferences page section):**
- [ ] Master toggle: "Let the agent control the built-in browser" — per
  agent in our model (slice 4)
- [ ] **Web URL open destination** — where web links open by default (our
  editor tab / external default browser) (slice 1)
- [ ] **Local URL open destination** — same for localhost/dev servers
  (defaults to the work browser) (slice 1)
- [ ] **Show full URL** toggle — origin only vs path+query+fragment in the
  address bar (slice 1)
- [ ] **Browsing data — Clear browsing data** (Profile.
  ClearBrowsingDataAsync masks: history, site data, cache, downloads)
  (slice 1)
- [ ] **Browsing history — Manage** (our own store, recorded from
  navigation events; list + delete) (slice 3)
- [ ] **Annotation screenshots** — when the agent comments on a page,
  include the screenshot (Always include / ask / never) (v2; pairs with
  visual comments on DOM elements — ChatGPT's annotation mode)
- [ ] **Password manager — Manage** (WebView2 password autofill on; the
  store lives in the work profile) (slice 1) · import from Chrome (v2)
- [ ] **Contact info / general autofill — Manage** (IsGeneralAutofillEnabled)
  (slice 1)
- [ ] **Downloads**: Location (default system Downloads, Change), **Ask
  where to save** toggle (save dialog), **Download history — Manage**
  (DownloadStarting event drives all three) (slice 1)
- [ ] **Site settings — camera/mic permissions** per site
  (PermissionRequested) (slice 3)
- [ ] **History access for the agent**: Always ask | always | never
  (slice 4; ask-on-first-use is the v2 approval UX)
- [ ] **Enable site tools** — discover/call WebMCP-style tools exposed by
  sites (v2)
- [ ] **Agent permissions table** — Site or pattern × Browsing × Downloads ×
  Uploads, values Requires approval | Always allow | Never, plus a Default
  row; "+ Add" exceptions. Ours: ADR-0128 `{domains, tier}` where Browsing ≙
  tier (read/act), Downloads/Uploads are `full`-tier scopes; "Requires
  approval" = the v2 ask-on-first-use prompt; v1 ships the manual editor
  (slice 4)
- [ ] **Developer mode — Enable full CDP access**, labeled "Elevated risk",
  off by default: unlocks raw CDP beyond the curated command catalog. Ours:
  ADR-0128's opt-in loopback port toggle (same semantics: elevated risk,
  owner's call, everything else keeps working without it) (slice 2)

**Non-goals kept from the benchmark:** Chrome extensions in the panel
(delegated to the user's real browser — our `ext/`+browserhost until
deprecated); full omnibox with synced profile.

**Surface reach (owner, 2026-09-13): the work browser tab is desktop-only
by nature** — it is a native WebView2 controller hosted by the shell, and
the whole ADR-0128 enforcement stack (navigation gate, CDP bridge, policy)
lives in the shell process; a web page cannot host it, and iframes die on
X-Frame-Options with no work profile. `/browser/` gets the adjacent
conveniences instead: the **Web/Local URL open destination** setting (links
from chat can hand off to the shell's work browser or the default browser)
and a **read-only state card** (title/URL/screenshot of the active tab,
which the daemon already holds via CDP) — v2, useful for Open-on-phone.
Agent commands work from any surface (same daemon); only the canvas is
desktop's.

CDP sub-spike (2026-09-13, from WSL — where the daemon lives): enumerated
WebView2's targets over loopback, connected to the logged-in GitHub tab,
`Runtime.evaluate` read the session identity, `Page.captureScreenshot`
captured the human's view end to end. **Browser Use is feasible on our
engine.** ADR-0128 now fixes the product form: default transport is the
host-API bridge (no port), the loopback port survives as an owner opt-in
with its cost documented, per-agent policy `{domains, tier: read|act|full}`
denies by default, and the layout is **tabs in the editor with a
Chrome-inspired tab strip** (owner, 2026-09-13) — not a fixed side panel.

### Slice 2, increments 1–2 — the host-API bridge (2026-09-13, `feat/browser-cdp`)

`btab_cdp_call` (one tab, one method, one JSON result) goes through
`CallDevToolsProtocolMethod` on that tab's controller; `btab_cdp_events`
records a read-tier event ring per tab (Page/Runtime/Network/Log, the
domains enabled on the first poll) with sequence numbers, so a poller can
tell a quiet page from a ring that overflowed. The gate lives in
`desktop-shell/src/cdppolicy.rs`: a named method catalog per tier, **deny by
default at every tier** — a method the table does not name is refused, never
assumed harmless. The loopback debug port is now an explicit
`PICODE_CDP_PORT` opt-in instead of every launch's default.

Increment 3, first half (2026-09-13, `feat/browser-agent`): the command
channel itself (ADR-0132). The shell's page opens one stream
(`GET /api/browser/stream`), the daemon pushes a command down it, the shell
runs it against the work-browser tab on screen and posts the answer to
`POST /api/browser/result`; the daemon holds the tool call until the answer
or the timeout. No new port, no new credential: the line is the session the
UI already has, and it reconnects like the feed's.

Increment 3, second half (2026-09-13, `feat/browser-policy`): the `browser`
Pi tool and the policy default. The tool (`packages/pi-browser`) names a
**verb** — `snapshot`, `screenshot`, `events` — never a CDP method: the daemon
maps the verb (`internal/browser.VerbFor`), resolves the agent's grant, and
the shell re-checks the method catalog. ADR-0134 fixes the default: an agent
with no grant reads **the tab the human has on screen**, and
`act`/`full` or any other origin needs an explicit per-agent grant (the
setting `browser.policy.<agent>`; the editor that writes it is slice 4).

Still open: slice 4 — the grants editor (Settings ▸ Browser: the tier ×
domains table), the `act` verbs that a grant unlocks, and the navigation gate
(`NavigationStarting` cancels an agent-caused load of an origin outside the
grant) — the catalog gate alone does not stop a page from being loaded.

## Conscious debt

- Optimize-VHD from the tray/app (needs an elevation design).
- Taskbar identity for the managed Work Browser (it is Chrome's icon).
- Icon resolution: the shell's `icons/icon.ico` now carries 16–256 px
  (tools/mkicon.go, the shell variant of the tray generator) — the v1 tray
  ladder topped at 64 px and the taskbar upscaled it blurry.
- Disk history/thresholds (P4 of `wsl-control`).


## Shell app bar (2026-09-12, ADR-0122, supersedes 0121)

The owner rejected the floating-controls frame; the reference is the
ChatGPT desktop app. Every shell window is now undecorated with two
webviews: a 40px local app bar (brand, drag, double-click maximize, flat
Windows caption buttons, close-red hover) and the page below it,
restretched on resize. The bar is local, so the frame is immune to what
the daemon serves and ADR-0121's frame mode, handshake and reserved-slot
convention were reverted from web/. Menus in the bar are deferred.
Requires Tauri's `unstable` multiwebview feature.

## Surface split (2026-09-12)

`/browser/` is the responsive web app; `/desktop/` is the shell's own
bundle — the browser app's App composed with the shell chrome (a
`shellChrome` prop, no window globals in the browser code). The
Management page moved into the desktop bundle (tokens imported from
`@picode/shared`, hand copy deleted). The shell loads `/desktop/`
directly, skipping the launcher picker. Boundary exception: the desktop
package may import exactly the browser package's named exports
(boundaries.mjs COMPOSES).
