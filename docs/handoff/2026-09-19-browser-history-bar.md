# 2026-09-19 — the address bar remembers (feat/browser-history-bar)

The browser's remaining items, and the owner's calls on them (one session):

## What landed
- **History dropdown** (`WebTabAddress.jsx` + `lib/addressHistory.js`): opens
  on focus, ranked typed-first then newest (the spec's own words), one row per
  page, the page you are on skipped, ↑/↓/Enter/Esc, one honest line when there
  is nothing ("No pages here yet…" / "No match in your history.").
- **One component, two panes**: the desktop pane offers every visit; the pane
  with no desktop shell passes `localOnly` (it can only frame loopback). The
  duplicated url bar markup is gone with it.
- **The layer rule holds**: the list is `role="listbox"` + `data-state`, which
  the shared vocabulary already matches — so a native WebView2 parks behind it
  and the clipping audit sees it (`overlayAudit ok:true`, hit on
  `[role="listbox"][data-state="open"]`).
- **Copy made honest**: "Search or enter a URL" → "Enter a URL" (a typed
  phrase used to become `<https://the phrase>`, i.e. an invalid-URL error), with
  a guard test so the promise cannot creep back without the feature.

## Owner's calls recorded in the topic file
| Item | Call |
|---|---|
| Import cookies and passwords… | **refused** (passwords need PiCode's own vault — a security-model arc; cookies via CDP are fragile under Chrome's app-bound encryption). Parity list and topic updated; the ⋮ entry stays out. |
| History access "Always ask" | **deferred** until an agent actually asks; the cheap shape then is a notice with one-click Allow, not a held call. |
| Device toolbar | **native half first** ("Responsive width": bounds + `ZoomFactor`, no ADR); CDP device emulation is a later ADR. |
| Search from the address bar | **not built**; the copy stopped promising it. |

## Evidence
9 new JS tests (ranking, dedupe, skip-current, query, local-only, junk,
layer guard, copy guard) + 435 app tests green; visual review on a scratch
instance: `histbar-dropdown.png`, `histbar-empty.png` (both read), the pick
verified end-to-end (click → value set → pane loaded), overlayAudit ok.

## Next up
- Device toolbar, native half (when wanted).

## Debts
- The desktop pane's own run (a real WebView2 parking behind the list) is
  Windows-only; the layer is the same one the ⋮ menu already proved there.
