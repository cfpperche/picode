# 2026-09-09 — feat/sidebar-no-version: brand name only

Shipped: sidebar header is `PiCode` without `v0.1.0+…`. Version stays in
the user menu (`#um-ver`). CSS `.brand-ver` gone.

Verified: `make close` green. Scratch `sidebar-no-version` at
`http://localhost:8473` — header screenshot (light + dark) read; eval
`brand=PiCode`, `brandVerCount=0`, `umVersion=PiCode v0.1.0+e3b5f07`;
`__picodeOverlayAudit()` ok.

visual-review: PASS (sidebar-brand.png, sidebar-brand-dark.png; empty
workspaces one line + Add workspace; overlayAudit ok)

Not done / debts: none.

Merge: fast-forward ready (`186f1dbf` + captures `172da5c8`).
