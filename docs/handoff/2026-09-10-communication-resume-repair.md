# 2026-09-10 — communication-resume-repair: native enrollment and resume

Shipped: OpenCode initialization no longer awaits its own instance API; resumed readiness rejects stale or malformed status. Codex uses the shared per-tool message client without enrollment restart (ADR-0111), with native discovery context and modern-hook precedence. Claude compact footers and resumed pane dimensions are preserved.
Source: `9a678efe`; closing note written from `make close-summary` in the review subagent.
Verified: `make ci-scoped` passed fmt, vet, hooks, eight Go packages and docs; adversarial review resolved six findings. Decision table and native evidence: `docs/plans/communication-resume-repair.md`.
Native: Codex 0.154.0 ↔ OpenCode 1.18.30 and OpenCode ↔ Claude Code 2.1.267 passed both checks, with all four message ACKs. Enrollment needed no extra model prompt; Codex retained PID/session; Claude completed detached at 80×24. The final build resumed the same OpenCode session and reconnected.
Models: Codex gpt-5.6-luna, OpenCode xAI Grok 4.6, Claude Sonnet 5. Z.AI refused the earlier call for insufficient balance; it was not used for the successful exchange.
visual-review: PASS — scratch empty, blocked, overlay, verified and native OpenCode/Claude captures inspected by a subagent; overlay audit passed. Hover was not captured.
Not done / debts: full six-CLI rerun, physical mobile and non-Linux recovery; long wrapped OpenCode footers remain conservative. Unobserved conversations need a native identity event, and native tool approvals still apply.
Merge: fast-forward ready; root integration and full `make ci` remain with the parent. No deployment performed.
