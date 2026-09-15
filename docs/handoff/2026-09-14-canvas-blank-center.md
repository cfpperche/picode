# 2026-09-14 — feat/canvas-blank-center: a native surface owns its container again

Shipped: `.native-surface { display: flex; flex-direction: column; overflow: hidden;
scrollbar-gutter: auto }` in `web/browser/src/styles/app.css`, next to `.app-surface`.
The page-frame change (34cbe5c1, 2026-09-14) dropped that column from `.app-surface`,
so `.cv-stage { flex: 1 }` had no flex parent: an open canvas measured 0px tall (the
plane was invisible) and *No canvas yet.* sat at the top of a 593px pane instead of its
middle. Canvas and the QA demo (ADR-0109) keep the column; the page frame is untouched.
Rule clause in `docs/benchmarks.md`; guard in `web/tools/app-surface.test.mjs`.
Verified: `make ci-scoped` PASS; scratch `center` (:8473) — empty state centres in a
593px stage (title 299–318 of 40–633, button 36px = `--ctl-h`), a canvas with a live
terminal panel + a text panel fills 1036×593 edge to edge at 100 % (toolbar, camera and
minimap on the bottom edge), stage == surface width (no scrollbar gutter), overlay audit
`ok: true`. Error state (`ft-msg` + Try again, forced by patching fetch) reads at the top.
visual-review: PASS (var/screenshots/cv-blank-centered.png, cv-plane-panels.png,
cv-blank-error.png read; card 5/5)
Not done / debts: the QA demo surface (`PICODE_DEMO_APP=1`) was not re-captured — it
gets the same restored column, unverified visually.
Merge: fast-forward ready

## Next up

- none local
