# 2026-09-21 — docs-in-app-tab: Documentation opens in the app's browser tab

Shipped: the user menu's Documentation item routes through `openWebTab` when
the app is the desktop shell with the Tauri bridge present
(`shellChrome && window.__TAURI__` — the same test `webappOpenPlan` makes);
otherwise it keeps the system-browser anchor. UserMenu gained an `onDocs`
prop threaded from App's sidebar wiring.

Verified: scratch, plain /desktop/ (no Tauri) — clicking Documentation calls
`window.open` with the docs URL and never mounts a work-browser tab (whose
surface cannot host a remote page outside the shell; the pre-fix build
mounted an empty one). Overlay audit ok. The shell branch itself was not
exercised in a real WebView2 — the mechanism is the Browser door's own
`btab_navigate`, in production since ADR-0122.

visual-review: PASS (docs-menu-final.png: menu open, item readable, audit ok)
Not done: the guide's other `_blank` links (TmuxGuardRow, Automations guide,
PeerConnectionDetails) still go to the system browser — same could be done
later if wanted.

## Next up

- Consider routing those remaining guide links through the same shell-aware
  opener.
