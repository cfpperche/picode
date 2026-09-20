# 2026-09-20 — feat/agent-toolbar-icons: unify bound agent terminal surfaces

Shipped: bound agents resolve through shared `web/shared/domain/agentTerminal.js`; mobile opens the terminal view for any bound CLI, including Pi, and uses `TermSurface` plus terminal attachments instead of the Pi-only `TerminalDock` path. Desktop Chat/Terminal controls remain icon-only with accessible labels.

Verified: `make ci-scoped`, `make close`, shared/mobile/browser terminal tests, mobile build, and full mobile workspace tests (362 passed). Mobile Vite was exercised against the local PiCode API with Memory/Pi, changes/Muse and browser/Codex; each reached `?view=terminal`, rendered xterm, and exposed the attach action. The attach sheet screenshot was read; `window.__picodeOverlayAudit()` returned `ok:true`.

visual-review: PASS (mobile terminal + attachment sheet screenshot; card 5/5)

Not done / debts: legacy interactive agents without `terminalId` still use the compatibility address until restart, as required by ADR-0162. Main was not changed and deploy was not run.

Merge: fast-forward ready at `a0b4dbde`.

## Next up

- Land `feat/agent-toolbar-icons` on main, run `make ci`, then owner-controlled deploy and live smoke test.
