# 2026-09-10 — matrix-edges: a Matrix edge grants a mailbox, never a transcript

Shipped (C4, both sessions): ADR-0116, accepted before any code. Migration 044, `GET/POST /api/matrices/{id}/edges`, `DELETE …/edges/{id}`,
`edges` on the matrix read, `matrix.edge.added|removed`; `PeerContacts` and `SendPeerMessage` are the union of *same workspace* ∪ *linked by a
live edge*, derived on every read, never cached; the MCP surface gained no verb. Surface: the header connector (React Flow `Handle` +
`onConnect`, agent and terminal panels only, loose connection mode), the two consent dialogs that are never merged (cross-folder naming both
folders, then the existing `POST /api/communication` enrolment), removal by chip or Delete confirming only while it grants, the broken state
with its reason (`web/shared/domain/matrixGrants.js` is the single judge), a link chip with the count in grid mode, and **Matrix links** in
`#/clis/messages` — every live edge, non-spatially, with a Remove that revokes.

Verified: `make close` green; scratch at `:8473`, two workspaces, four `pi` agents, both themes. Every mailbox claim driven over
`/mcp/communication` with scratch-minted bearers, never read off the UI: `list_contacts` for Delta went `(none)` → `Atlas` on the cross-folder
draw, and back to `(none)` on removal, on panel delete and on a session change; `send_message` was accepted, then refused. Row by row in
`docs/architecture/matrix.md`, *Accepted in a browser (2026-09-10, C4)*.

visual-review: PASS (12 captures, dark + light, each read; card 5/5; `__picodeOverlayAudit()` ok on both dialogs in both themes)

Debts: no keyboard path to **draw** a link (React Flow's `Handle` offers none) — the audit list is the keyboard way to read and revoke; the
grid chip counts but does not name the far end; the audit list reads one matrix detail per matrix (N+1, capped at 50).

Merge: ff-ready.
