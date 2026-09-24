# 2026-09-24 — feat/restore-starts: Undo and Bring back restore the same way

Shipped: the owner asked for Undo and Bring back to match (Undo used to leave the agent stopped).
- The shared `restoreAgent` in `web/shared/client/launchAgent.js` restores and starts the agent: a terminal launch/start with resume, else the Pi agent's managed/start.
- The shared `restoredToast` in `web/shared/domain/agentHistory.js` gives both paths one message. Both are unit-tested.

Verified: `make ci-scoped` PASS.
visual-review: PASS (`var/screenshots/restore-starts-*.png`, not committed): Undo and Bring back gave identical results — same id, running, conversation visible, live composer.
Not done: not deployed (owner's call).
Merge: fast-forward ready.

## Next up

- Agent history phone screen, in its own branch.
