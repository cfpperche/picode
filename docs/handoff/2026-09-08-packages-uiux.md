# 2026-09-08 — packages-uiux: adversarial pass on the pi-roles config page

Shipped (owner asked for an adversarial review + design-system refinement):
selects/inputs now share the house recipe (`.pkc-select`/`.pkc-input` =
`--ctl-h`, `--radius-ctl`, bg-panel on bg-base rows, accent focus) instead of
raw native controls; breadcrumb ⇄ scope-switcher on one balanced top row;
builtin roles carry one-line descriptors; model fills the row, thinking fixed
180px; dirty state is loud (accent foot border + accent file path + line);
thinking select is disabled until a model is chosen (it was possible to set
thinking without model — dirty draft, silently dropped on save); stray `}`
rendered below the last row (edit artifact) removed; "Thinking: unchanged"
placeholder fits.
Verified: ci-scoped PASS (test-js, build); scratch instance in light AND dark
(dark-first holds) — workspace + agent scopes, dirty state, audit ok:true.
visual-review: PASS (uiux-workspace, uiux-presets, uiux-dirty, uiux-dark,
uiux-agent-dark all read; card 5/5).
Not done / debts: same as 2026-09-08-packages-config.md; selects are native
elements styled to tokens (no Radix select — native still beats a styled fake
per AGENTS.md style).
Merge: fast-forward ready.
