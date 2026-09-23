# 2026-09-23 — feat/instr-mobile: the New agent line on the phone

The last item of the AGENTS.md plan. The phone's New agent sheet (`web/mobile/src/components/NewCliPrincipal.jsx`) shows the same one line as the desktop dialog, from `createLine` over `GET /api/workspaces/{id}/instructions`. For example: "Claude Code reads CLAUDE.md; AGENTS.md is left out." The phone still has no Instructions tab; the table is a wide-screen view, and this line is the part that matters at creation.

Verified: `make close`; scratch `instrmob` at 390px: Work ▸ workspace actions ▸ New agent, Claude Code picked, overlayAudit ok, screenshot read in a subagent. Nothing deployed.
