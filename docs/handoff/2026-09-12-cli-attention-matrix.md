# 2026-09-12 — cli-attention-matrix: protect native startup and drafts

Implemented: Claude Code waits for a saved conversation before resume; terminal Pi checks drafts before claiming attention; native diagnostics explicitly request execution.
Verified: all six real CLIs passed paired draft/resize, busy and post-resume exchanges with matching replies and both acknowledgements; six identical draft/pane snapshots survived a daemon restart.
Verified: cold Pi/Grok exchange; fresh Claude safely waits and then connects with the same native ID after a first message; final initialized Codex-sender diagnostic passed without manual nudge.
Gates: scoped CI passed before final diagnostic-copy refinement; final closing gates pending.
visual-review: PASS; desktop/mobile happy, waiting, empty and overlays read; fresh Claude shows Idle, Waiting to connect, first-message instruction and Open; overlay audits passed.
Limits: initial Codex/Hermes/OpenCode and resumed Codex/Hermes require a first prompt; an earlier failed Codex diagnostic is recorded rather than counted as passed. Plan: docs/plans/cli-attention-matrix.md.
Merge: pending closing gates and integration.
Deploy: not performed; owner-controlled.
