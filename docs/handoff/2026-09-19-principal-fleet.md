# 2026-09-19 — principal-fleet: Inbox when a managed CLI needs you
Shipped: bound CLI `needs-you` files one blocking Inbox FYI
(`cli-needs-you`); idle/working marks it done; Inbox **Open terminal**
goto `term:<id>` (desktop + mobile). Unbound shells stay chips-only.
Push rides `inbox.created`. Deploy readiness already covers terminals.
Verified: `make close` PASS; Go tests for once/dedup/unbound + Inbox
action. Blind spot: not clicked on a live Inbox.
visual-review: UNVERIFIED (no scratch of the Inbox button)
Not done: Fatia 2 (`picode mcp` on the principal). Topic:
`open/managed-principals.md`.
Merge: fast-forward ready.

## Next up

- Fatia 2: `picode mcp` tools on the bound principal (ADR-0154)
