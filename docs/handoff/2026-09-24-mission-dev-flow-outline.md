# 2026-09-24 — feat/mission-dev-flow-outline: dev-flow phases in the docs outline

Shipped: the six phase headings in `docs-site/guide/dev-flow.md` carry ids and
theme permalink anchors, so VitePress "On this page" now lists "1 Elaboration"
…"6 After deploy" and each entry navigates; the owner pill carries VitePress's
`ignore-header` class so outline titles stay clean. Paid the
`docs/handoff/open/docs-audit.md` dev-flow outline debt; changelog fragment
`docs/changelog.d/mission-dev-flow-outline.md` written.
Verified: `make ci-scoped` PASS (docs-check, docs build incl. dead links, vale
0 errors). Live vitepress preview at 1280 px and 375 px: outline entries for
#elaboration, #landing, #after-deploy click-navigated (hash set, heading lands
at the 134 px navbar offset); anchor geometry checked (hover # top-aligned,
14 px into the card's left padding, opacity 0 at rest). Blind spot: hover and
scroll-spy exercised in headless Chromium only, not Safari/Firefox.
visual-review: PASS (4 captures, `var/screenshots/devflow-*.png`, read by a
subagent; card 5/5 — outline readable, hover # clears chip/title/pill, phone
cards wrap long code spans with no clipping or horizontal overflow).
Landed: `819228fb7` on main; full `make ci` PASS. Mission
`mission_WUFG3EEGTSC3LAAM54DRTWPGJF` accepted at v19 after the Omp terminal
and its processes stopped. The worktree and branch were removed.
Pilot finding: completed history survived cleanup, but the working folder and
its `kind: file` evidence paths became unavailable. Six hashed screenshots and
the accepted record were preserved under `var/qa/missions-pilot/<mission-id>/`;
the durable product debt is in `docs/handoff/open/missions.md`.
