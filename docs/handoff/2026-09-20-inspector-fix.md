# 2026-09-20 — feat/inspector-fix: mobile Inspector actions follow the checked-out worktree
Shipped: adversarial-review fixes for feat/mobile-inspector. (1) GitActions sheet derives branch/upstream from the followed checkout instead of the anchor root, giving actionStatus parity with the desktop rail. (2) Root group wears its own hint header when multiple checkouts render. (3) Agent glance clears on a 409 folder-move instead of freezing counts.
Verified: make ci-scoped PASS; visual-review PASS (adv-groups.png, adv-actions.png). Follow-mode command derivation itself is code parity with desktop Inspector.jsx actionStatus — the fixture root is always dirty, so the pill branch was not screenshot-exercised.
visual-review: PASS
Merge: fast-forward ready (1 commit).

## Debts

- Follow-mode pill branch has no visual coverage: fixture root is always dirty, so the followed-checkout pill was never screenshot-exercised
