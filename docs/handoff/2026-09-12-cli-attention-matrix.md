# 2026-09-12 — cli-attention-matrix: protect native startup and drafts

Implemented: Claude Code waits for a saved conversation before resume; terminal Pi checks drafts before claiming attention; native diagnostics explicitly request execution.
Verified: all six real CLIs passed paired draft/resize, busy and post-resume exchanges with matching replies and both acknowledgements; six identical draft/pane snapshots survived a daemon restart.
Verified: cold Pi/Grok exchange; fresh Claude safely waits and then connects with the same native ID after a first message; final initialized Codex-sender diagnostic passed without manual nudge.
Gates: make close PASS on merged main content; merged browser build PASS; main full CI remains the integration gate.
visual-review: PASS; browser/mobile happy, waiting, empty and overlays read; merged /browser and /mobile rechecked; fresh Claude waiting instruction/Open and overlay audits passed.
Limits: initial Codex/Hermes/OpenCode and resumed Codex/Hermes require a first prompt; an earlier failed Codex diagnostic is recorded rather than counted as passed. Plan: docs/plans/cli-attention-matrix.md.
Cleanup: seven owned terminals removed, own scratch server stopped, zero remaining processes with scratch HOME; 11 passed diagnostics independently checked against replies and both acknowledgements.
Merge: fast-forward ready; integration pending.
Deploy: not performed; owner-controlled.
