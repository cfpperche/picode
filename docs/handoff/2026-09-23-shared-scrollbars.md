# 2026-09-23 — feat/shared-scrollbars: one scrollbar sheet for every surface, the Management window included

The owner's screenshot showed the desktop Management window with Windows' light default scrollbar on the dark page; the compact 8px bar existed only as copies in the browser and mobile app sheets.

Shipped (65138b6fd): the generic rules live once in `web/shared/styles/scrollbars.css` (exported in `web/shared/package.json`; the boundaries check refuses unexported shared imports), imported after `theme.css` by browser/desktop/mobile `index.css` and `management.html`; app sheets keep only their sidebar tweak.
Verified: built CSS has the generic rule once per bundle in the same cascade position; getComputedStyle gives 8px and the thumb colour; `make test-js` and `make ci-scoped` green. Page-only change: live after `make deploy`, no `make desktop-restart` needed.
visual-review: PASS on a headed capture. Blind spot: headless agent-browser runs with `--hide-scrollbars`, so a headless capture shows no scrollbar at all; use `--headed`.
Merge: `make close` had not run when this note was written.

## Next up

- Owner runs `make deploy` to put the shared scrollbar sheet live (page only).
