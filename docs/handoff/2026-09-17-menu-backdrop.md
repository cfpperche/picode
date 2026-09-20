# 2026-09-17 — menu-backdrop

The owner: "por que quando clico no menu a página pisca".

## What it was

An HTML menu cannot paint over a native WebView2 sibling, so the browser tab
freezes the page into a still, hides the webview and draws the menu on the
still. The old decision table hid the page as soon as the menu opened, with or
without a still — so a click without a fresh capture (the hover prefetch is
1.5 s old, or the pointer arrived fast) showed the host's gray for the length
of a capture, then the still and the menu. That gray is the blink.

## Fix

`menuOpen` is only set once there is something to sit behind the menu: the
cached still (prefetched on hover **and** focus now), or a finished capture —
with a 600 ms ceiling so a stalled capture still opens the menu, over the gray,
instead of a click that appears to do nothing. The failed-capture row is
unchanged; `previewBusy` became dead state and went with it.

## Not verified

The menu exists only in the shell (a plain browser renders no ⋮ button), so a
scratch cannot show this path — the live check is the owner's: open a browser
tab with a site, click ⋮, and the page must stay put while the menu appears.

## Trap paid for here (recorded in the topic)

A scratch embeds the UI at build time and `make web` is stamp-guarded, so the
running page was one revision behind the source: an old bundle’s leftover
`setPreviewBusy` threw `ReferenceError` and blanked the app, which looked like
a regression from this change. `rm -f var/web.built` before the scratch build.
