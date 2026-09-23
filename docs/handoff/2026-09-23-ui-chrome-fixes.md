# 2026-09-23 — feat/ui-chrome-fixes: four UI chrome defects from the agent-exits review
Shipped: (1) Pi icon — pi.dev's favicon took its colour from the OS scheme;
Pi is now an inline mark in neutral #8a8a96 (`PI_MARK`,
`web/shared/domain/terminalCli.js`), and inline marks drop the white face
plate and the round clip (desktop and mobile `app.css`) — the plate is what
made Pi's block logo a solid square. (2) Toast × moved off the divider for
notices with an action row (`.notice.no-head.has-foot`, both `Notice.jsx`).
(3) Dialog Cancel border visible in dark (~1.8:1, was ~1.1:1), including the
Outcomes editor's Cancel. (4) New `--danger-ink` token, light #a8293f (~6.3:1
on its tint, was ~4.3:1). Debt paid in `docs/handoff/open/ui-chrome.md`;
changelog fragment `docs/changelog.d/ui-chrome-fixes.md`.
Verified: `make ci-scoped` and `make close` PASS; `terminalCli.test.js`
covers the inline mark. Visual review on a scratch instance, screenshots read
in a subagent: the first pass flagged the Pi icon (the plate, then a round clip
on the phone); both fixed, final pass on desktop dark/light and phone dark.
Blind spot: not run inside the Windows desktop shell or on a real phone.
visual-review: PASS
Not done / debts: none.
Merge: fast-forward ready
