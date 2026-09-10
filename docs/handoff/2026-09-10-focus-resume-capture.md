# 2026-09-10 — focus-resume-capture: reload completes fullscreen mode on the first real input
Shipped: a reload restores the mode without the browser part (no gesture survived the navigation);
now the first trusted click or keypress reports a `resume` event and `browserIntent` asks for
fullscreen from that handler — keyboard lock rides along (`web/desktop/src/lib/useFocusMode.js`,
`web/shared/domain/focusMode.js`).
Guarded races: Escape and the fullscreen chord are excluded from resuming (they mean "leave" — a
mode that stops while the enter request travels takes the browser back out, with a belt when the
request lands); synthetic input refused (`isTrusted`); one attempt per page load; refit follows the
fs transition (a resume passes no mode transition).
Verified: `make ci-scoped` PASS; scratch QA — reload+keypress → fs+lock restored; reload+click → same;
first-gesture Escape and first-gesture chord both leave the mode with NO stuck fullscreen;
`__picodeOverlayAudit` ok; console clean. Screenshot read (resume-1-restored.png).
visual-review: PASS
Not done / debts: real-window behavior identical to term-key-capture debt (headless cannot prove
browser-side key capture, only our state machine).
Merge: fast-forward ready — `cd /home/goat/picode && git merge --ff-only feat/focus-resume-capture && make ci`.
