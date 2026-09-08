# 2026-09-08 — packages-cards: installed cards restored, search standardized

Shipped (owner call, same day as ADR-0099 shipped): the Installed tab is the
card grid again — preview frame, name + scope tag, description (known
adapters) or source path, meta (kind/version/behind/path always visible) —
with Configure/Update/Remove in the card foot. `.pkg-search` and the count
chip now share the system control language: `--ctl-h`, `--radius-ctl`,
`.dlg-input` border/background/focus. Row-list CSS removed.
Verified: ci-scoped PASS (test-js 766, build); scratch instance — filter
("roles" → 1 of 5), Configure click → config page, overlayAudit ok:true.
visual-review: PASS (pkg-cards.png, pkg-cards-filter.png both read; card 5/5).
Not done / debts: unchanged from 2026-09-08-packages-config.md (mobile config
UI, mid-edit conflict detection, declarative adapter manifest with pi-compact).
Merge: fast-forward ready.
