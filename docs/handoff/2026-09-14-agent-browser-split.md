# 2026-09-14 — agent-browser-split

Branch `feat/agent-browser-split` — slice 2 (the channel reads the split
binding, 979b9223) plus the owner directive: **the split survives a relaunch**.

## Done

- Split layout + last url per pane persist in localStorage
  (`picode-agent-splits` / `picode-agent-split-urls`); boot prunes panes whose
  host tab is gone (and their urls), and the webview is recreated at the last
  url the first time the host tab is shown — after the surface's own bounds
  push, so `ensure` never lands a 900x640 webview on top of the UI.
  `webSeqRef` is seeded past the restored ids. Tests: openTabs.test.js 8/8
  (`readAgentSplits` junk rejection + clamp, `filterAgentSplits` prune).
- Visual (scratch :8473, Chromium): seeded layout reloads back — pane at 30%,
  drag to 60% writes through (storage + flexBasis agree), max restores 100%,
  dead host tab pruned. Evidence: var/screenshots/split-restored-60.png.
  overlayAudit ok. visual-review: PASS.
- Live proof of the round trip earlier today: snapshot (962 KB AX tree) and
  screenshot (PNG 1053x1400) through the bound pane on the owner's desktop —
  the stale `picode-shell.exe` (pre-CDP capabilities) was the ACL refusal;
  rebuilt + swapped + relaunched by hand (see the debts below).


## Debts

- The tab-strip × and expand toggle live in the pane toolbar, which renders
  only in the shell; their write path shares the same persist effect and tests
  but got no Chromium click.
