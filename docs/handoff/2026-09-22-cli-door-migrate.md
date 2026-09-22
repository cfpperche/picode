# 2026-09-22 — feat/cli-door-migrate

ADR-0184 slice 2. Migration 067 binds every launch terminal no agent owns
(sign-in terminals, named "<CLI> sign-in", excepted) to a new agent
`a-<terminal id>` (free agents and off-workspace folders get `work_path`),
copies its `term:` grants onto that agent (an agent's own grant wins) and drops
every `term:` grant. Terminal identity now holds no grant: a caller with only a
terminal id resolves to the agent bound to it (`callerAgentID`), else keeps
`term:<id>` for audits and a browser session drive while `computer.Resolve`
reads off and `browser.Resolve` the default; grant edits for a shell answer 400;
the policy lists show agents only; delivery refuses an unbound terminal (403).

Dogfood on a copy of the owner's DB: migrated cleanly, 10 agents untouched, the
4 orphan `term:` grants (terminals long gone) removed. `checklist-mirror`, the
one unbound CLI terminal measured earlier today, no longer exists.

visual-review: PASS (scratch clidoor2, desktop: Computer and Browser pages list
agents only, empty/no-match copy, shell refused 400 by the API; overlayAudit ok).

## Next up

- `feat/cli-door-close`: remove the route (17 fixtures), sign-in terminals internal (`kind`), shown on their card, reaped on credential/exit/15 min/boot.
