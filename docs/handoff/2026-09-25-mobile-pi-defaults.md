# 2026-09-25 — feat/mobile-pi-defaults: Pi Defaults fill the settings row on the phone
Shipped: owner asked to fix Pi › Settings › Defaults ending short on the phone. ConfigFields
puts Provider, Model and Thinking selects in `.cfg-fields.cfg-row` (flex, wrap), each sized to
its text (provider 14–149, model 14–258, thinking 264–363 vs the 376 edge). The phone media
query in web/mobile/src/components/agent-clis.css, scoped to #cli-settings-view, sets Provider
`flex: 1 1 100%`, Model `flex: 1 1 0; min-width: 0`, Thinking `flex: none`. Other ConfigFields
mounts (create dialogs, automations) untouched.
Verified: `make ci-scoped` PASS; 390px → provider 14–376, model 14–271, thinking 277–376;
360px → 14–346 with the same split; no overflow. visual-review: PASS at 390 and 360.
Not done / debts: none. Merge: fast-forward ready.
