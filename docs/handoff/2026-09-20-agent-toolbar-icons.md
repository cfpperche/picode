# 2026-09-20 — feat/agent-toolbar-icons: unify bound agent terminal surfaces

Shipped: bound agents resolve through shared `web/shared/domain/agentTerminal.js`; mobile opens the terminal view for any bound CLI, including Pi, and uses `TermSurface` plus terminal attachments instead of the Pi-only `TerminalDock` path. Desktop Chat/Terminal controls remain icon-only with accessible labels.

Verified: `make ci-scoped`, `make close`, shared/mobile/browser terminal tests, mobile build, and full mobile workspace tests (362 passed). After landing at `60406180`, full `make ci` passed on main. Prior UI probes used a desktop-sized mobile viewport against production, not compliant scratch/device acceptance; Memory was Claude, not Pi.

visual-review: UNVERIFIED. The previous PASS and 5/5 claim is withdrawn; no touch-scroll or physical-device acceptance was performed.

Not done / debts: mobile Agent and Terminal remain separate screens; the touch-to-wheel handler exists only in Terminal. Menu parity is incomplete. Legacy agents without `terminalId` still use `TerminalDock`; their stop cleanup needs verification. This is a partial implementation, not completed terminal unification.

Merge: landed at `60406180`; owner requested forced deployment with the remaining gaps disclosed.

## Next up

- Complete shared mobile terminal behavior and verify touch scroll, menu parity and legacy cleanup on scratch before claiming acceptance.
