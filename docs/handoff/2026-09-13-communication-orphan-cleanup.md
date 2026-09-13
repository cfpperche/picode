# 2026-09-13 — communication-orphan-cleanup: no bearer files outlive their connection

Shipped: the communication worker sweeps `communication/peer_*` setup directories whose connection row is gone or revoked (deleted owner, disable/replace, crash between revoke and removal). Unrelated entries are untouched; live connections keep their setup. ADR-0106's maintenance concern is closed.
Verified: `TestPeerLaunchOrphanSweep` (live kept; revoked/stray/deleted-owner removed; foreign dir/file untouched); `make close` ci-scoped.
visual-review: n/a
Not done / debts: onboarding row 12 (stubborn child) is the remaining communication cut.
Merge: fast-forward ready after this close.
