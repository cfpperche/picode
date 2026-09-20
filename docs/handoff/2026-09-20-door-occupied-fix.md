# 2026-09-20 — door-occupied-fix: image sends from mobile were blocked on clean composers
Owner report: sending messages with images to any CLI agent terminal from
mobile was refused "occupied — finish or clear the draft" with a clean
composer. Root cause: the Fatia F gate classified any pane it could not
read as empty as occupied — failing closed on rendering drift (dialog
open, missing frame rules, new suggestion format).
Fix: the gate is three-state (peerComposerState) — recognized empty →
verified delivery; positive draft evidence (bright text past the prompt
marker; grok border body; pi frame content) → 409 occupied; anything
unclassifiable delivers with the demoted "unverified" receipt — the
pre-Fatia-F behaviour. Codex/Hermes/claude dim suggestions never count as
drafts (measured: typed input carries no dim). Reproduced e2e on a
scratch claude pane that could not be classified: 200 unverified (was
409). Unit tests cover ghost, draft, drift, grok border, pi frame.
Blind spot: on unclassifiable panes the paste is blind by design (the
pre-F behaviour) — the receipt says so.

## Next up

- Nothing queued in this plan; ∞ (managed mode per CLI) stays deferred
  until ADR-0091 is re-measured.
