# 2026-09-22 — feat/ade-web-tail

This branch pays the last three open code debts in `docs/handoff/open/multi-cli-ade.md`, which are flipped there. The only item still open there is the owner's run on the Windows VM.

What's new: checked, and left unchanged. The dialog shows at most the 3 newest releases (`MAX_RELEASES`), so from 0.3.1 on the 0.1.0 headline "Run real Pi agents" never renders. It stays as history.

Mobile More search: the Settings, Packages and Providers shortcuts now follow the CLI of the last agent opened (`lastCli` in `web/mobile/src/screens/More.jsx`, Pi when there is none), and they no longer say "Pi". Limit: opening a guest agent on mobile opens its terminal and does not update "last".

Found by visual review: in both apps, every CLI shortcut that matched the search added its own "Agent CLIs" group, so the heading repeated. This was already on main. It is now one group (`moreMenuModel.js`, `userMenuModel.js`), with a test for a query that matches all four.

Dead code removed: the `kind === "free"` branches of both CreateForm, the free path of both `createSubmit.js`, and `createFreeAgentSchema`. Its schema tests moved to `freeAgentPickSchema` and `createWsAgentSchema`. Verified: `make test-js` green, `make ci-scoped` PASS. Visual pass 1 FAIL (the repeated heading). Pass 2 PASS on "settings" (one row). Pass 3 PASS on "se" (one heading over the CLI settings and Connectors). visual-review: PASS (v3-more-search-se.png; card 5/5).
