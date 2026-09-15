# 2026-09-15 — feat/custom-provider-fields: meta-ai-style custom provider config
Shipped: GUI custom-provider form now covers per-model display name,
input text/image, cost USD-per-1M (all-or-nothing), supportsUsageInStreaming
compat, and per-level provider values (e.g. xhigh→high). Backend/schemas/
models.json round-trip; fragment docs/changelog.d/custom-provider-fields.md.
API key flow unchanged: literal into auth.json, exact cheaperinference parity
per owner decision.
Verified: gates green; end-to-end QA on scratch instance passed
create→models.json/auth.json→roster→Edit prefill→partial-cost refusal.
visual-review: PASS (cpf-meta-ai-form.png + cpf-meta-ai-top.png, overlayAudit
ok both surfaces, card 5/5).
Merge: fast-forward ready.
