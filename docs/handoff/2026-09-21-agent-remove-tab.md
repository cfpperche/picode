# 2026-09-21 — feat/agent-remove-tab: land on a neighbour tab when the selected agent is removed

Removed tab now lands on a neighbour via `pickNextTab(ids, removed)` in
`web/browser/src/lib/openTabs.js` (commit 9eeb6dec) — right neighbour of the first removed tab, else
nearest left, else null so `showHome` takes over. Fixes the dead surface where removing the selected
agent left `selectedId` null with other tabs open: no dashboard (`noTabs` false), no "gone" card
(`missing` false). `removeAgent`, `removeWorkspace` and `closeTab` share it through a new
`adoptTab()` in App.jsx; a CLI agent's bound terminal tab id joins `removedTabs` (ADR-0160);
`closeTab` no longer picks the last tab nor runs selection side effects inside a `setTabs` updater.
UI refinement only — no architecture boundary crossed, no ADR owed.

Verified: 5 new unit tests for `pickNextTab` in `openTabs.test.js` (16/16 pass); `make ci-scoped`
PASS (fmt, vet, hooks, test-js, build). Visual: qa-scratch E2E on :8475 — removing the selected
agent with remaining tabs selects the right neighbour (Bee→Dee); last tab removed lands on the
dashboard; unselected removal keeps selection; `closeTab` × picks the right neighbour. Screenshots
in `var/screenshots/` (remove-agent-before/after-neighbour/confirm-overlay/dashboard.webp);
overlayAudit ok; visual card 5/5. Blind spot: that E2E used the scratch, not a real tmux session.

visual-review: PASS
Merge: fast-forward ready.

## Next up
## Debts

- [x] CLI-agent bound-terminal removal (ADR-0160) has no E2E coverage; unit tests only. — paid 2026-09-21 in feat/agent-remove-undo: omp agent with bound terminal removed on a qa-scratch while its `t:` tab was selected; neighbour selection, terminal-row delete and tmux teardown verified (see 2026-09-21-agent-remove-undo.md).
