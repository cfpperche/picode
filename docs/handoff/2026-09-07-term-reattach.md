# 2026-09-07 — feat/term-reattach: terminals reattach after a phone lock

Shipped: `web/shared/client/termSocket.js` (+tests) owns the /ws/term attach
lifecycle — exponential backoff 1s→10s, 12 failed attempts per burst, kick on
visibility/online/remount, one "Session ended. Reopen the terminal." line on
give-up, kick restarts a burst. All four term clients (mobile/desktop
ShellTerm + TerminalDock) reuse the same xterm across reattach; key/resize
handlers register once; dock `term.reset()` only on first open; `closeTerm`
drops without reattach. `window.__picodeTerms` QA hook in both lib/terms.js.
Verified: `make ci-scoped` PASS; 7 new unit tests (382 js total); E2E on
scratch (mobile viewport): drop→reattach keeps tmux copy-mode [25/25]
pixel-identical and interactive; away-and-back while detached revives the
same entry; killed session gives up cleanly and heals when recreated.
Decision table (socket state × user action → behavior) fully covered by the
unit tests above; rows exercised live in the scratch E2E.
visual-review: PASS (term-reattach-1..7 PNGs read; overlayAudit ok)
Not done / debts: iOS wake with a zombie OPEN socket relies on the server's
close frame arriving on network resume (standard path; not force-testable in
Chromium). Server-side pong deadline (60 s) unchanged.
Merge: fast-forward ready.
Accepted live 2026-09-07: owner confirmed the fix on their phone session
after deploy 0.1.0+b0087b1 — lock/wake reattaches without exit/re-enter.
