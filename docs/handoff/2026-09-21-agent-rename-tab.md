# 2026-09-21 — feat/agent-rename-tab: renaming an agent now renames its bound terminal tab

Shipped: Store.UpdateAgent renames the agent's bound terminal in the same
mutation when the name actually changes and the agent is bound, announcing
terminal.updated (best-effort; a vanished terminal does not fail the rename).
A non-Pi agent runs as its bound terminal (ADR-0160); the tab strip labels that
t:<id> tab with term.name, previously copied from the agent only at bind time
(attachAgentTerminal / EnsureAgentTerminal) while the rename route patched the
agent row alone. normalizeTerminalName extracted in internal/store/terminals.go
and reused by RenameTerminal.
Verified: TestUpdateAgentRenamePropagatesToBoundTerminal in
internal/store/agent_terminal_test.go — bound terminal follows the rename with
event order terminal.updated → agent.updated, unbound agents untouched, a
same-name patch is a no-op. ci-scoped PASS (fmt, vet, hooks, go[21]).
Blind spot: store-level only, not exercised through a live tab strip UI.

visual-review: n/a
Not done / debts: docs/architecture/managed-principals.md gained the name-loan
invariant paragraph; docs/changelog.d/agent-rename-tab.md written.
Merge: fast-forward ready

## Debts

- (none)
