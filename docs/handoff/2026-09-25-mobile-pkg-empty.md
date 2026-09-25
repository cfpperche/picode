# 2026-09-25 — feat/mobile-pkg-empty: phone Packages empty state centred, bold title
Shipped: owner asked to fix the phone Packages empty state (left-aligned, grey,
regular weight; desktop's is centred with a bold title). The phone-only override in
web/mobile/src/styles/mobile-settings.css (v2 phone redesign a8e74de59, 2026-09-07,
no recorded reason) was removed, so the shared `.pkg-empty` shape applies: centred,
600/14px title, action centred under it.
Verified: `make ci-scoped` PASS; scratch 390px probe on codex/opencode/pi (title
centre x=195, 600 14px). visual-review: PASS (all three).
Blind spot: the suggested-picks variant ("<CLI> offers these…") did not appear on
the scratch; it now uses the desktop's centred layout, unseen on the phone.
Left as is: Connectors pane `.mcp-empty` keeps the same phone left/muted override (not asked).
Not done / debts: none. Merge: fast-forward ready.
