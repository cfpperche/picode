# 2026-09-13 — communication-codex-resume: custom resume argv does not attach

Shipped: `peerResumeExact` is the only attach gate. Leading globals, `--` and extra operands are a different launch. Codex decision table plus Claude `--mcp-config` only on the recorded recipe.
Verified: `TestPeerResumeExactDecisionTable`, `TestPeerLaunchCustomResumeDoesNotAttach`; `make close` next.
visual-review: n/a
Not done / debts: activation acceptance; onboarding rows 9/12; PTY race; non-Linux recovery.
Merge: fast-forward ready after this close.
