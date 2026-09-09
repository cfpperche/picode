# 2026-09-08 — cli-providers: native providers under Agent CLIs

Shipped: ADR-0103; `#/clis/providers/pi` and `/new` in both app-owned frames.
Pi-only provider capability and machine scope; legacy links, menu search,
model/composer shortcuts and OAuth return paths use the canonical route.
Native credential APIs, vault, quotas and verification remain unchanged.
Catalog reads are independent of CLI inventory; refresh errors retain drafts.
Closed editors ignore late OAuth results; narrow tabs reveal the active item.

Verified: `make web`, `make ci-scoped`; 40 focused route/domain/menu tests.
Owned synthetic fixture: both apps passed navigation, key/account operations,
usage, simulated OAuth success/failure/cancellation and login/logout shortcuts.
Real fixture agents remained stopped; credential writes were intercepted.
visual-review: PASS — dark/light desktop/mobile, 320–1920px, 44 overlay/control
checks ok; empty/blocked/error/overlay screenshots read; visual-card 5/5.
Four public documentation captures regenerated and read.
Evidence: `var/screenshots/cli-providers/` (browser results and gate logs).

Not done / debts: real vendor OAuth, real credentials and physical devices.
Merge: fast-forward ready from `7aef3182`; full `make ci` runs on main.
