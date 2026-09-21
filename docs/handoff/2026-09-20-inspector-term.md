# 2026-09-20 — feat/inspector-term: Inspector toggle on the live terminal header
Shipped: after the owner saw no Inspector button on mobile — the print showed a TUI agent's TERMINAL view, where the toggle only existed on the chat view header — the Inspector toggle now renders on the live terminal header (first icon, before the keyboard and ⋮) and on plain #/term screens. The drawer stays open while the route remains agent/term.
Verified: ci-scoped PASS; visual-review PASS (term-header.png, term-drawer.png — toggle first, drawer 200 OK).
visual-review: PASS
Merge: fast-forward ready.

## Debts

- Agent-in-terminal-view drawer exercised via plain #/term (same component path); a live TUI agent was not running in the fixture
