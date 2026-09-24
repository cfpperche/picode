# 2026-09-24 — pi-official-mark: Pi wears pi.dev's official mark

Shipped: the Pi CLI favicon keeps pi.dev's own three-path art but carries the
official ink again — `PI_MARK` (`web/shared/domain/terminalCli.js`) re-inked
from the neutral #8a8a96 to pi.dev's light-scheme #111111; the dark theme
recolours it through the shared `term-cli-face` invert by extending the rule
to `src^="data:image/svg+xml,"` in both apps' `styles/app.css` (Pi's data URI
is the only one). Test pin updated (`terminalCli.test.js`), changelog fragment
`docs/changelog.d/pi-official-mark.md`.

Verified: `make ci-scoped` green (fmt,vet,hooks,test-js,build); scratch
instance piofficial — seeded agent Atlas (cli pi); computed `filter` is
`invert(1)` in dark / `none` in light on desktop sidebar, `#/clis` and mobile
Work; pixel-sampled the captured faces: #eeeeee ink on dark grounds, #111111
on light (pi.dev's own dark ink is #f6f6f6 — one step off, same as every
inverted mark in the family). Screenshots read in a subagent (ShotReader);
method's blind spot: glyph structure judged at native screenshot scale, no
zoom crops.

visual-review: PASS (six surfaces; card: no clip, mark official, no plate)

Not done / debts: none. Incidental (pre-existing, untouched by this diff):
in mobile Work light the QA workspace row's folder icon at ~(23,101) renders
as a solid dark rounded square — flagged, not chased.

Merge: fast-forward ready.
