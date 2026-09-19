# 2026-09-19 — principal-mcp: picode mcp on a bound principal
Shipped: binding Claude Code, Codex or OpenCode as a managed principal
sets launch Tools to every `picode mcp` family when unset. Explicit
Tools (including empty) stay. Other CLIs stay Connectors-only.
`Identity.Principal` is `grant.FromIDs`. Grants unchanged (`term:<id>`).
Verified: `make close` PASS; HTTP create + fill decision table.
Blind spot: no live Claude Code launch of the injected servers.
visual-review: n/a (no UI)
Not done: Fatia 3 (create a CLI runner from the workspace). Topic:
`open/managed-principals.md`.
Merge: fast-forward ready.

## Next up

- Fatia 3: create a CLI runner from the workspace, not only `#/clis`
