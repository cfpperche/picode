# 2026-09-23 — feat/agents-md-trim: AGENTS.md keeps the rules, docs/agents/ keeps their detail

The owner approved the trim the study proposed (§10.6) and then approved the diff itself. AGENTS.md went from 331 lines and 21,261 bytes to 214 and 13,053. Hermes no longer cuts it on small-window models (20,000-character floor), it is well under Antigravity's 24,000-byte cap, and it is close to Claude Code's 200-line guidance.

Every rule stayed, because the other CLIs do not follow links on their own. Moved verbatim to `docs/agents/`, each linked from its rule: the board rules (`handoff.md`), how the git hooks enforce worktree isolation (`git-guards.md`), the restart / pkill / tmux incidents (`processes-and-tmux.md`), and the repo map (`repo-map.md`). Rationale that is already recorded in ADRs was dropped from AGENTS.md (the 2.7M-token figure, the ADR count, the app-door list). A rule-by-rule comparison caught three details the first cut dropped, and they are back: skipping the decision table for polish, "mixed heights is FAIL", and `make cert-timer`.

Rule 8 ("the next task starts in a new terminal") is unchanged; on 2026-09-23 the owner chose to continue in one session, and whether that becomes the rule is still the owner's call.

Verified: `make close`; docs-living pass. Nothing deployed.
