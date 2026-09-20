# 2026-09-19 — agent-create-cli: Fatia B (New → Agent)
Shipped: workspace **New → Agent** picks Pi or an installed CLI.
`POST /workspaces/{id}/agents` `{cli}` creates a guest agent plus launch
terminal (`agents.terminal_id`). Hub `#/clis` unchanged. Play on a guest
starts the terminal, not `Runtime.Start`.
Verified: `make close` PASS; scratch `agent-create-cli-picker.png` +
`__picodeOverlayAudit` ok; Cancel closes. Menu is Agent / Shell only.
visual-review: PASS (happy overlay); empty/error UNVERIFIED (Pi always listed)
Not done: Fatia C (migrate `managed_clis`). Topic:
`open/managed-principals.md`.
Merge: fast-forward ready after this note.

## Next up

- Fatia C: migrate `managed_clis` → `agents`
