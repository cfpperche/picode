# 2026-09-17 — claude-grok-correlation: finale attempt blocked by Claude quota, guards vindicated

Attempted the last open correlation (claude → grok) on scratch `localhost:8472`: both terminals connected, identity confirmed, no expired login this time. `check_EGQOMAZS7QDLN7TVBOOTBYYZA3` expired with zero messages created — the Claude account's weekly quota is exhausted ("Usage limit reached · continuing automatically at 12:10am"), its composer footer changed to a limit notice, and the sender attention correctly refused the unknown layout (silent pre-claim refusal) instead of pasting into it. The guards behaved as designed in front of a genuinely unready composer.
Verified: connections + identity on both ends; history shows no message for the check; both panes captured. Evidence: `var/qa/claude-grok-finale/` (root, credentials stripped).
visual-review: n/a (no code changed)
Not done / debts: retry claude → grok after the quota reset (owner's account); `open/communication.md` onboarding bullet rewritten with the new cause and retry condition. No changelog fragment — nothing user-visible changed.
Merge: fast-forward ready (`make land`).
