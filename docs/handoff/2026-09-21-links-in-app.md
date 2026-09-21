# 2026-09-21 — links-in-app: every chrome link opens in a PiCode browser tab

Shipped: `lib/externalLinks.js` (the shell bridge) no longer hands external
anchors and `window.open`s to the system browser — it dispatches
`picode-open-link`, and App.jsx opens a PiCode browser tab (`openWebTab`)
when the Tauri bridge is present, falling back to the system anchor behavior
otherwise (a remote page cannot be hosted in a plain-browser tab). The click
capture now catches external anchors with or without `target="_blank"` —
anchors that navigated the webview in place were the same dead end. Covers
Documentation, Full changelog, setup guides, "How it works", help links —
every http(s) link in the shell.

Verified: unit tests for the bridge rewritten for the event handoff
(9/9); scratch, plain /desktop/ without the bridge — dispatch records
`window.open` with the guide URL, hash unchanged, no work-browser tab
mounted. Blind spot: the shell branch (bridge present, WebView2 hosts the
page) was not exercised in a real desktop shell — the mechanism is the
Browser door's own `btab_navigate`.

visual-review: PASS (links-plain-fallback.png: no empty work-browser tab; the
guide opened externally as the caption promises)
Not done: mobile is out of scope (no browser tabs there).

## Next up

- If the system-browser escape is ever needed per link, the browser prefs
  (`localOpenDest`/`webOpenDest`) are the place to hang it.
