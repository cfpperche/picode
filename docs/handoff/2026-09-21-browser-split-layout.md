# 2026-09-21 browser split layout

- Cause: `.workspace-view.agent-split-on{flex-direction:row}` pulled the tab
  strip into the split row (strip renders inside the view only when
  shellChrome=false); the desktop shell owns its own top row, so it never broke.
- Fix: grid — strip spans the top row, agent pane/divider/browser share row 2;
  pane width moved to container var `--split-w` feeding the grid track;
  divider hover, max-mode, and web-tab flex rules restored.
- Verified (scratch, 1400px + 800px): strip/split/max combos, /desktop/ route,
  close-split, overlayAudit ok every state, zero horizontal overflow.
- Proof: var/screenshots/browser-split-*.webp; uiux-review read before edit,
  visual-review PASS with card 5/5 (no overlays in flow; empty + max + narrow read).
- Debts: none. ADR: none (no protocol/persistence/security/process boundary).
