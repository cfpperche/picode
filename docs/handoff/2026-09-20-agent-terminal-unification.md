# 2026-09-20 — feat/agent-terminal-unification: agent view toolbar
Shipped: mobile agents now keep Chat and Terminal on one agent route; the
switch is in the contextual toolbar and terminal deep links use
`#/agent/<id>?view=terminal`.
Verified: `npm run build:mobile`, `node --test web/mobile/src/lib/mobileRoutes.test.js`,
`make ci-scoped`, and scratch QA with a real tmux-backed Pi TUI.
Visual QA used 390x844 screenshots for Terminal and Chat; overlay audit was
`ok: true`, with aligned 36px toolbar controls and no viewport overflow.
visual-review: PASS
Not done / debts: the remaining runtime extraction from Agent.jsx into the
Agent CLIs terminal implementation, desktop toolbar parity, and physical
iPhone/PWA/IME acceptance remain for follow-up.
Merge: fast-forward ready.

## Next up

- Move the agent interactive pane to the Agent CLIs terminal/session adapter
  while preserving Pi managed chat and unbound shell terminals.

## Debts

- Physical iPhone/PWA/IME acceptance remains external; no device was used.
