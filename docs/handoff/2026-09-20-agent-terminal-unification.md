# 2026-09-20 — feat/agent-terminal-unification: agent view toolbar
Shipped: mobile agents now keep Chat and Terminal on one agent route; the
switch is in the contextual toolbar and terminal deep links use
`#/agent/<id>?view=terminal`. Non-Pi agent rows resolve their bound terminal
through the Agent CLIs terminal surface; Pi interactive rows retain their
existing attach path for now.
Verified: `npm run build:mobile`, `node --test web/mobile/src/lib/mobileRoutes.test.js`,
`make ci-scoped`, and scratch QA with a real tmux-backed Pi TUI.
Visual QA used 390x844 screenshots for Terminal and Chat; overlay audit was
`ok: true`, with aligned 36px toolbar controls and no viewport overflow.
visual-review: PASS
Not done / debts: the managed Pi interactive migration to the Agent CLIs
terminal/session adapter, desktop toolbar parity, and physical iPhone/PWA/IME
acceptance remain for follow-up.
Merge: fast-forward ready for this UI slice; not deploy-ready as the full runtime unification.

## Next up

- Migrate the managed Pi interactive pane to the Agent CLIs terminal/session
  adapter, then align the desktop toolbar without changing unbound shell
  terminal ownership.

## Debts

- Physical iPhone/PWA/IME acceptance remains external; no device was used.
