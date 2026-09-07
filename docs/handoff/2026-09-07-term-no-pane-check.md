# 2026-09-07 — feat/term-no-pane-check: no checklist strip on the pane

Shipped: desktop `TermSurface` no longer renders `.term-check`. Sidebar
cards and the phone agent row still show the operator line (ADR-0081
amendment). Pi TUI keeps its own step card.

Verified: `make ci-scoped`. Scratch desktop: terminal with a posted
checklist has no strip above xterm; sidebar card still shows the line.

visual-review: PASS (desktop term with posted 2/3 plan: no .term-check,
sidebar card still "edit the pane 2/3", overlayAudit ok).
Not done: live Pi TUI owner glance.
Merge: not yet.
