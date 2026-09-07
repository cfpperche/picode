# 2026-09-07 — feat/mobile-ime-firstopen: extra keys wait for a tap

Shipped: extra keys require `userArmed` (pane tap or header icon).
Attach no longer `term.focus()`. Header icon does not preventDefault on
open. `#m-app` pins only when a focused field shrank visualViewport
against the unfocused baseline.

Verified: `keyboardInset.test.js` decision table (attach focus, rest
large inset, user tap). Scratch `:8471` mobile 390×844: first-open no
keybar, header Show keyboard, keys-on, keys-off no leftover gap.
overlayAudit ok.

visual-review: PASS (Chromium). iOS IME still owner acceptance.
Not done: real iPhone IME overlay with this build.
Merge: not yet.
