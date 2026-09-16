# 2026-09-16 — browser-menu-split: the split pane's dead links, a still that proves itself

Three bugs from the owner's live check (screenshots 10:10–10:11, agent
split with x.com loaded):
1. The ⋮ menu's settings links did nothing there: the split-pane
   WebTabSurface never got `onBrowserSettings` (tabs-only prop, and the
   `?.` swallowed it). Passed in both places now.
2. The menu hid x.com behind uniform gray: `btab_preview` can resolve with
   empty/undecodable bytes (first live run of the raw-bytes IPC path) and
   the tab treated any blob URL as a good still. Now the shell rejects
   empty captures, and the tab only hides the page behind a still that
   decoded to real pixels (`Image` onload + natural size); a failed capture
   keeps the menu usable over the host background. The `covered` decision
   table is in WebTab.jsx; the "is this capture worth it?" decision lives
   in the tested `lib/previewStill.js`.
3. Found while verifying (2): opening any settings view with a work tab
   selected left the page painted over it — the tab parked on tab switch
   but never on route change. Both instances now park while the route is
   elsewhere (`onPane`) and restore on return.
Verified: stub scratch (empty capture → no still + `visibility false`;
1×1 PNG → still decodes + `visibility false`; Browser settings →
`#/browser` parked, back → restored); node tests 365/365 (+5 new);
`cargo xwin build` ✓.
Needs deploy + desktop-restart (shell change in btab.rs).
visual-review: PASS on the stub paths read (menu-fallback.png,
menu-still.png); the real still over x.com is the owner's click.

## Next up

- Owner: split with x.com → ⋮ → the page must freeze visibly behind the
  menu; Browser settings → Settings ▸ Browser with no x.com bleeding
  through; example.com too (tells a blank capture from a blank page).

## Debts

- The COM capture + native hide still have no automated test
  (Windows-only); the decode gate and the empty-reject are unit-tested.
