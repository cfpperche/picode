# 2026-09-13 — feat/terminal-padding: split leftover xterm columns left/right

Shipped: after FitAddon floors columns, leftover pixels split left/right via
`termSlackPad` / `applyTermSlack` in `web/shared/domain/termFit.js` and CSS auto
margins on `.xterm-screen` (browser `app.css` + mobile; desktop shares that CSS).
Changelog fragment `docs/changelog.d/terminal-padding.md` already present.

Verified: `make close` green (ci-scoped: fmt,vet,hooks,test-js,build). Scratch
`qa-scratch.sh start termpad` at http://localhost:8473. Desktop `/desktop/` live
term: leftover 4px → 2/2; viewport 1371 leftover 2px → 1/1. Ruler shot
`var/screenshots/term-pad-ruler.png` (gitignored). Mobile: 7px → 3/4 (≤1px).
`window.__picodeOverlayAudit()` ok.

visual-review: PASS
uiux-review: PASS
Not done / debts: none for this padding.
Merge: fast-forward ready (`cd /home/goat/picode && git merge --ff-only feat/terminal-padding && make ci`). Deploy is the owner's call.
