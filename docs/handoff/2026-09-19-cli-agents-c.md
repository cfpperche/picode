# 2026-09-19 — cli-agents-c: Fatia C (managed_clis → agents)
Shipped: leftover `managed_clis` rows copy onto `agents` (same id when
free); table dropped (061). GET/POST `/principals` are agents
(`kind=agent`, key=agent id). `DELETE /managed-clis/{id}` aliases
DeleteAgent. Inbox needs-you looks up `AgentByTerminal` (source still
`term:`, Fatia E). Deleting a guest terminal deletes the agent row.
Verified: `make close` PASS; store migrate + AgentByTerminal + principals
HTTP + inbox tests. Blind spot: live DB with Fatia-3 rows unobserved until
deploy. visual-review: n/a (no JSX).
Merge: fast-forward ready.

## Next up

- Fatia E: rekey Inbox / `picode mcp` / grants onto the agent id.
