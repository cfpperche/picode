# 2026-09-08 — ws-favicon-size: match Work group icons to row marks

Shipped: mobile Work workspace heads use the same 24px slot / 22px glyph
as agent and terminal rows (`WsFavicon` 17→22, `.m-work-group-face`).
Verified: scratch :8471; workspace favicon, folder fallback and row
marks all 22×22; 390/320/light; overlayAudit ok.
visual-review: PASS (workspaces-two-390.png + workspaces-320.png; card 5/5)
Not done: desktop sidebar stays 16px (identity chrome, not this list).
Merge: fast-forward ready.
