# 2026-09-24 — session-rule

Rule 8 now matches how the owner works: one branch at a time, and the owner
decides when a session ends (ADR-0105 amendment, 2026-09-24).

- `AGENTS.md` §8, `CONTRIBUTING.md` §6 and `docs-site/guide/dev-flow.md`
  drop "the next task starts in a new terminal"; closing docs and
  screenshots still go through a subagent.
- The ADR-0105 measurement stays the trigger for a re-measure if long
  sessions grow costly again.
