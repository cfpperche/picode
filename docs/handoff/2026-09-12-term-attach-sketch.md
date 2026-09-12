# 2026-09-12 — feat/term-attach-sketch: sketch pad + growing message field in the attach composer

Shipped: `SketchEditor.jsx` (per shell; Excalidraw direct, lazy — not the pin studio) behind a
**Sketch** button in `TermAttachBar`/`TermAttachSheet`; the drawing leaves as `sketch.png` through
`POST /api/terminals/{id}/drop` and the chip reopens (scene in memory) until Send. Message field:
textarea 1→4 lines (`web/shared/domain/attachText.js`) — desktop Enter sends/Shift+Enter breaks,
phone Enter breaks, Send or Ctrl/⌘+Enter submits. Docs: `docs/architecture/cli-terminal-launch.md`
(prompt door paragraph) + fragment `docs/changelog.d/term-attach-sketch.md`. No ADR (UI refinement
of ADR-0089).

Verified: `make ci-scoped` PASS; 9 tests (`attachText.test.js`); scratch `termattach` (real pi CLI
in tmux) with agent_browser. Desktop 1280×800: empty, 4-line field (96px), 5-line cap + scroll,
sketch draw/insert/re-edit, Send wrote `7c39664a-sketch.png` + pasted caption and `@path`. Mobile
390×844: Enter inserted a newline, 4 lines = 108px, cap toast, sketch → sheet reopened with the
chip → Send wrote `2dcba061-sketch.png`. `__picodeOverlayAudit()` ok in every captured state;
screenshots read from `var/screenshots/ta-*.png` / `tm-*.png`.

visual-review: PASS (card 5/5; known app-wide occlusion below)

Not done / debts: Shift+Enter unit-tested only (CDP cannot hold Shift across key events, so
Enter-send was verified live); a >4 MB sketch PNG was not produced (same readAttachFile path as
files); the blank-sketch error toast covers the pad's Cancel/Insert for 12–30s — the existing
app-wide "toast covers a surface's Close" debt; the mobile blocked-state toast was eval-verified.

Merge: fast-forward into main after `make close` on this tree.
