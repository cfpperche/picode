# 2026-09-25 — feat/mobile-page-width: phone Agent CLIs and app pages use the page gutter only
Shipped: phone CSS only (web/mobile). Inner blocks no longer stack side padding
on the 14px gutter (--m-pad): .cli-detail/.cli-editor, .cli-tabs-frame,
.cli-catalog, .peer-body/.peer-workspace and the CLI notice/guard/loading/
settings-body variants keep vertical padding only. CLI settings/models rows hang
their 2px "overridden" lane in the gutter (margin-left −10px); catalog tile face
drops the desktop −6px pull; app group items inset 12px→4px. Agent CLIs content
now starts at 14px (was ~30); app pages 27px inside the card border. Desktop untouched.
Verified: `make ci-scoped` PASS; scratch at 390×844, text-inset probe over 21
phone routes; visual-review pass 1 FAIL (Messages 30px, Codex rows, tile icon)
fixed, pass 2 PASS. Not checked on a physical phone.
visual-review: PASS
Not done / debts: CLI settings value selects are content-sized, left edges
do not align (noted, not fixed).
Merge: fast-forward ready.

## Debts

- the overridden-setting marker in its new gutter position is not verified (no overridden setting on the scratch)
