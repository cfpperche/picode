# 2026-09-12 — feat/settings-layers: Pi Settings, one layer at a time

Shipped: the refinement in `docs/plans/cli-settings-ux.md` (marked implemented).
The pane edits one layer at a time — a labelled switcher writes
`layer=global|project|agent` onto the route beside `agentId`, the body shows
that layer only, and the file it writes is named under it. Rows carry
provenance from the API's `has` map (`Set here` + accent bar, `Pi default`,
`From This machine`) and **Use inherited**, backed by a new `patch.reset[]` in
`PUT /api/pi-settings` (`internal/pisettings` deletes exactly those keys;
compaction/steering/follow-up are live-applied from the effective values after
a reset). The keyboard map is a **Keys** sub-tab (`tab=keys`) with no layer,
and the pane opens ~1000 px tall instead of ~6000.

Verified: `make ci-scoped` PASS (17 paths). Both settings harnesses were ported
(not duplicated) and are green against the docs fixture:
`scripts/qa-cli-settings.mjs` — 31 results over desktop + mobile (switcher, the
route and its reload, provenance, the reset round trip, untrusted workspace,
free agent, deleted agent, `focus=scoped-models`, unknown reset refused) — and
`scripts/qa-cli-settings-recovery.mjs` — 13 rows (draft retention per layer,
the eight restart boundaries, malformed-file quick sheet, keys with unreadable
defaults). `internal/pisettings` gained reset unit tests and the server a
`livePatch` test. `__picodeOverlayAudit()` ok in every capture.

visual-review: PASS (desktop-global, desktop-keys, desktop-context,
desktop-narrow, mobile-untrusted read; card 5/5)

Not done / debts: P2 (a filter over the layer rows) was dropped — six rows do
not need one, and the Keys tab keeps its own filter. Clicking the *pane* tab
resets the sub-tab and the layer (pane links are plain navigation). The agent
layer sits outside the native-defaults fieldset on purpose (the malformed-file
row). Process: I edited the tree in the root checkout by mistake for a few
minutes; it was restored clean before any commit and the same patch was applied
in the worktree. `docs/handoff.md` is at its 8 KB cap, so nothing was added
there; this note carries the pointer.

Merge: fast-forward ready.
