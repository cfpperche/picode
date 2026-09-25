# 2026-09-25 — feat/mobile-pkg-band: no empty band above "Nothing installed."
Shipped: owner asked to fix the empty strip between the Installed/Marketplace tabs
and "Nothing installed." on the phone's Codex/OpenCode Packages pane. Cause:
VendorBody always rendered `.gpkg-sticky` (sticky, opaque, 8px vertical padding);
only its filter toolbar was conditional (rows > 1), so with 0–1 rows the empty
box drew a 16px band. Fix in web/mobile and web/browser Packages.jsx (desktop had
the same box, invisible on its panel): the wrapper renders only when rows > 1.
The marketplace's own sticky filter is unchanged.
Verified: `make ci-scoped` PASS; scratch probe on codex/opencode/pi (no
`.gpkg-sticky`, empty state follows the tabs) and desktop codex. visual-review: PASS (all four).
Seen, not fixed: phone empty state is left-aligned regular weight; desktop's is centred bold.
Not done / debts: none. Merge: fast-forward ready.
