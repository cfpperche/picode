# 2026-09-25 — feat/md-live-longpress: press and hold opens a link in Live on touch
Shipped (1bdd583f7): on touch, press and hold (500ms, 10px slop) opens a link in the
Live editor like Ctrl/⌘+click: a relative file opens, `#anchor` jumps to the heading,
the web opens in a new tab. Works on the editor and in rendered table cells. A short
tap still edits the link; the first tap in an editor on a coarse pointer shows a
one-time "Press and hold to open" hint (clamped on screen, fades after 2.2s, ignores
touches), and link tooltips read the same on coarse pointers. Files: the pure tracker
web/shared/editor/longPress.js (driven-clock test, 2 tests), wiring in mdLive.js, CSS
in md-live.css. The open-topic Next item for this is removed.
Mechanism (QA round 1): after a hold the node the finger went down on is usually gone
(CodeMirror virtualizes the scrolled line, or the view is replaced), so its touchend
reaches nobody, not even document capture. Chrome then synthesized a click that moved
the cursor (line 50 instead of 42) or focused the newly opened README's editor, which
would open the phone keyboard. Fix: after a hold fires, `swallowNextTap` swallows the
next mousedown/mouseup/click at document capture for 800ms and the press state resets
on a timer; vibrate is called only after user activation (no Chrome console message).
Verified with real CDP touch (Input.dispatchTouchEvent + touch emulation, driver
scratchpad/touchd.mjs). Round 1 FAIL: the leaked tap. Round 2: anchor stays on line 42
(2/2); README opens unfocused (3/3); table cells work (2/2); hint appears once; a normal
tap after a hold works; desktop Ctrl+click and plain click unchanged; no console errors
but a pre-existing 404 for a missing README image; overlay audit ok, visual card 5/5.
Screenshots: var/screenshots/md-press/. Caveat: CDP touch emulation never fires
`contextmenu`, so refusing the menu was verified with page-JS events only.
Not caused here: on the phone, Back after opening a file via a link goes to the Files tree, not the previous file; in Live, raw `<kbd>` shows in the link colour.
visual-review: PASS (2 rounds). Landed 66a104871 (Next/debts: docs/handoff/open/markdown-preview.md); the first merge-gate `make ci` failed on server TestCLIAdapterPreviewMatchesExecution (TempDir cleanup "directory not empty" — the second time today, see md-live), the rerun was green.
