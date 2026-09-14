# 2026-09-14 — feat/pty-context-menu: one right-click menu for every PTY pane

Shipped: every `.term-pane` uses the `/term/` catalog (`paneCapabilities` +
`buildTermMenu`). `dataset.termKind` is the handler address, not a visibility
axe. Interactive Send is ADR-0130 door 5 (`deliverToInteractiveAgent`). Attach
/ Ask use `POST /api/agents/{id}/drop` + additive `paths` on `/prompt`
(ADR-0089). Rename/Remove say **agent**; settings land on `#/termset`
(“Terminal defaults”). Continue in pins `sessionPath` (path-only; cwd is
workPath else workspace.path else pane cwd); `liveHolderFor` treats interactive
tmux as live. Chat composer stays generic. `splitOn` reaches the builder.

Verified: `make close` green. Scratch `pty-menu` (:8472): Atlas TUI menu has
Ask / Attach / Send / Rename agent / Settings / Files / Close / Remove;
shell has Run command + Rename terminal, no Attach; settings title is
Terminal defaults; composer still “Save selection as snippet”.
`__picodeOverlayAudit()` ok.

visual-review: PASS (atui-menu.png, shell-menu.png, termset-from-agent.png,
composer-menu.png; overlayAudit ok; card 5/5).

## Next up

- Mobile terminal actions sheet for in-terminal agents (same capability table).

Merge: main moved — merge main, rerun `make close`, then ff from the root.
