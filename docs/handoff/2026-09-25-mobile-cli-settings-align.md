# 2026-09-25 — feat/mobile-cli-settings-align: CLI Settings fields stack full width on the phone
Shipped: owner asked to fix field alignment in a CLI's Settings on the phone. Measured at
390px: selects sat beside the name and started wherever it ended (x=107, 147, 239); text
fields stopped at 225px; switches at the right. Fix in web/mobile/src/components/agent-clis.css
(phone media query, #cli-settings-view/#cli-models-view): a row whose control is a
select/textarea/text input stacks — name + source line above, control 100% width (14→376);
switch rows stay on one line; role rows (.set-row-role, Omp Models) and composite rows (Model
input + Choose…, Also read + Add, Pi Defaults group) keep their layout. Rhythm: 6px inside a
stacked row, 10px extra top margin on every row → ~20px between settings.
Verified: `make ci-scoped` PASS. visual-review pass 1 FAIL (name closer to the field above,
10 vs 17px) → rhythm; pass 2 PASS but lower halves uncaptured (window scroll does not move
the phone scroller) → recaptured via scrollIntoView, PASS; nit (switch rows 12px after
stacked rows) → uniform margin; pass 4 PASS on Codex and Pi Settings top/bottom, Omp Models.
Noted, not fixed: Pi Defaults composite ends at x≈362 (not 376); "All models" caption at
x≈18; "Choose…" label ellipsized at 390; Omp Models role selects end at x≈365.
Not done / debts: none. Merge: fast-forward ready.
