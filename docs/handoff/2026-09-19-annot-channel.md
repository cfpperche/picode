# 2026-09-19 — a saved chip, a Send that never lit up (feat/annot-channel)

Owner: chip "teste" saved on the page, strip still bare Send. The page's own
DOM worked; every page→chrome message went nowhere, silently.

## Root cause
`ANNOTATE_RECEIVERS` was keyed by tab id and never pruned, and the arm path
**skipped subscribing** when the key existed (`let already = …contains_key`).
A webview recreated under the same id therefore had no listener while
ExecuteScript still landed in it: pins, card and chips all fine, strip deaf.
`close_inner` pruned RECEIVERS but not this map; `annotate_tabs()` (a set the
arm wrote and nothing ever read) was never pruned either.

## What landed
- Arm always (re)subscribes: `{handler, token}` per tab, old token removed
  first — live channel, no double delivery, no stale key. Close prunes.
  Dead `annotate_tabs` set deleted. Cross-built here: `make desktop-shell`.
- Page: `enter()` announces AND syncs (it used to return early when already
  on, so a re-arm was silent), so recovery keeps the pins.
- Chrome: silence after an arm retries once (re-arm = re-subscribe + report),
  then toasts the truth. Proven in the harness: re-arm → `enter`+`state`,
  count 1, chip and pin intact; real keys still type, Save still saves.
- Guards: Rust 12/12; test-js + build green.

## Debts
- `REINJECT_SCRIPT` is still never called (warning in the cross-build): a
  full page navigation while the mode is on loses the script and the strip
  keeps claiming it is armed. A NavigationCompleted hook is the fix.
