# 2026-09-23 — prose-audit-w3: wave 3 of the architecture prose audit
Shipped: wave 3 (native surfaces & apps). canvas.md carried all ten
drifts — the same `web/desktop/src/...` path class as wave 1 (the shell
bundle only bootstraps; the code lives in `web/browser`). The other
seven files verified clean: computer-tool's closed catalog of 23
actions counted in `internal/computer/actions.go`, capture caps
1280/480 in computer.rs, Docker monitor limits confirmed in
`Validate()` (30/60/300 s, retention 7/30, 32-project cap, 4
concurrent, 128 containers), preview ticket 32-byte token + 26-char
base32 label + 1 MiB overlay, `/api/extension/*` routes present.
Verified: `make ci-scoped` PASS; fixes grounded in greps. Method blind
spot: canvas.md's 1437 lines were swept for paths/numbers/ADRs, not
read paragraph by paragraph; historical amendment dates accepted as
recorded.
visual-review: n/a
Not done / debts: wave 4 (stable, ~20 files) remains.
Merge: fast-forward ready.
