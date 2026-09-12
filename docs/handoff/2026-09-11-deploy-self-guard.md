# 2026-09-11 — the deploy guard stopped counting its own caller

`fix/deploy-self-guard` → `main` (4174e15c). ADR-0086 amended.

## What was wrong

`GET /api/deploy/readiness` answers about the whole fleet and cannot know
who asked. A session that runs `picode deploy` from its own PiCode pane is
mid-turn *because* it is asking, so it read itself as busy and the guard
could never open for it. Seen twice today: `make deploy` from an agent
terminal refused naming that same terminal, and a watcher armed to poll
for an empty list would have burned its full two hours on a window that
only exists in the blink between the caller's own turns.

## What changed

`guardDeploy` drops the busy owner whose id matches `PICODE_TERM_ID` or
`PICODE_AGENT_ID` before deciding. Nothing about the route changed — the
WSL reclaim path reads it unfiltered, which is right, since there the
caller is not a pane being restarted.

## Watch for

- An agent pane deploying still ends its own turn on restart; that is the
  consent, not a bug.
- `--force` now has one meaning only: ending someone else's turn. If a
  refusal ever names only the caller again, the env var is missing from
  that pane.
