# 2026-09-07 — feat/cli-prompt-impl: ADR-0089 drop + paste + attach UI

Shipped: `POST /api/terminals/{id}/drop` and `/prompt`. Desktop attach
bar on a running Agent CLI `TermSurface`. Mobile header Attach → sheet.
Workspace browse from the terminal cwd. Caps 4 × 4 MB. Inspector type
on a CLI terminal is 409 `cli`.

Verified: `TestTerminalPromptDecisionTable` (missing, shell, drop, size,
escape, too many, dead pane, paste, busy, inspector). `planAttachFiles`
JS. Scratch `:8471` desktop bar empty+chip; mobile sheet. overlayAudit ok.

visual-review: PASS (term-attach empty/chip desktop, sheet mobile)
Not done: iPhone Photos sheet (owner). Per-CLI mention polish (D7).
Merge: ff as aca6628b; `make ci` green; deployed `0.1.0+aca6628`.
