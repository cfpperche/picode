# 2026-09-19 — agent-continue-remove: Continue in… on agents + removal hardening
Owner reported (live): CLI agent rows lacked Continue in… that terminal
rows have, and removing an agent left the row in the sidebar — a second
remove then failed "agent not found".
Continue in…: `agentRowMenu` now carries `terminalHandoffMenu` of the
bound terminal (ADR-0088) — same submenu, same targets; the desktop
AgentRow renders one and two-level submenus and routes picks through
`onContinueTerm(term, target)` (Sidebar passes it through). Hidden while
the terminal has no pinned session.
Removal: the scratch on current main reproduced removal working; the
owner's production failure matches a lost delete response — the server
completed, the client treated the error as fatal and skipped the
refetch. `removeAgent` now treats "not found" as already-deleted and
always refetches the fleet (removal is rare; the feed still patches the
rest).
Verified: scratch :8472 — Continue in… submenu with all targets
(screenshots read: agent-continue-in.png), overlayAudit ok, remove → row
leaves, server clean. Node tests 10 pass; ci-scoped PASS. Mobile rows
have no menu (unchanged).

## Next up

- Fatia F: automations and Inspector reach CLI agents through the prompt
  door (ADR-0089/0107).
