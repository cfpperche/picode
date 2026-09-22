# 2026-09-22 — feat/cli-door-agents

ADR-0184 slice 1: every user-facing CLI launch creates an agent. `POST /api/agents`
and `POST /api/workspaces/{id}/agents` take `overrides` (checked before any row
exists; `newLaunchAgent` binds the launch terminal). Agent CLIs' New terminal is
**New agent** (profiles included), Sessions' Open in terminal is **Resume as
agent**, the palette's entry opens New agent, and the cross-CLI handoff lands as
an agent (`createCLIAgent`). A Pi launch with overrides is a Pi agent bound to
its terminal; `--session` moves onto `SessionPath`. An Omp agent whose args name
a conversation (`--resume`, `--session-dir`) keeps them instead of its own
session folder. Client door: `web/shared/client/launchAgent.js` (both apps).

`POST /api/clis/{cli}/terminals` still exists (17 test fixtures); slice 3 removes it.

visual-review: PASS (scratch clidoor; desktop 1440 + mobile 390: New agent form,
blocked Grok 400 leaves nothing behind, Resume as agent, palette; mobile select
descenders fixed with `padding-block: 0`; overlayAudit ok; card 5/5).

## Next up

- `feat/cli-door-migrate`: unbound CLI terminals → agents; `term:` grants of migrated terminals → agent id; the four `term:` grants on the owner's instance are orphans (terminals gone) and can be dropped.

## Debts

- Sidebar cold load shows "No workspaces yet" / "No agents yet" for a few seconds before data arrives (seen on scratch, outside this diff).
