# 2026-09-19 — cli-agents-e: guests call as their agent (Fatia E)
Guest CLI launches now carry `PICODE_AGENT_ID` next to `PICODE_TERM_ID`
(`launchIdentityEnv` in cli_launch.go), so `picode mcp` resolves the agent
principal (grant.FromIDs is agent-wins) and grants given to the guest
agent in Settings ▸ Computer / Browser match the caller — no per-terminal
setup. Same spelling Pi agents already get from `Agent.SpawnEnv`.
Inbox: `cli-needs-you` items are filed for the agent (source agent),
close when the CLI moves on, when the agent is deleted or when its
terminal is deleted (cleanup in DeleteAgent/DeleteTerminal). The Inbox
app's Open terminal action follows the agent's bound terminal; Pi agents
keep ADR-0060 (no escape hatch). Migration 062 rekeys open terminal-keyed
items onto the agent and closes orphans.
Verified: unit tests (identity env decision table, migration, app actions,
needs-you filing), full store+apps, server package, ci-scoped PASS.
No UI change — visual-review n/a.
Blind spot: live guest (Claude Code) with a real grant flip observed only
via unit tests, not against the desktop shell.

## Next up

- Fatia F: automations and Inspector reach guests through the prompt door
  (ADR-0089/0107).
