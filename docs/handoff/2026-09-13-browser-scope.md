# 2026-09-13 — feat/browser-scope: who may call the browser tool

Docs-only session. The owner's decision for v1 (accepted): the browser tool
stays Pi-only, and **a plain pi TUI works too** — in read mode. Nothing in the
code refused it: `internal/rpc/runtime.go` sets `PICODE_AGENT_ID` + `PICODE_DATA`
for a managed agent (which is how a package finds server.json and the token),
while a terminal launch injects only `PICODE_TERM_ID`/`PICODE_TERM_URL`, so a
TUI resolves to the default policy and reads the tab on screen. The difference
is identity, not connectivity. Gating on a missing id was considered and
rejected as cosmetic: the same process can assert one (ADR-0134), so v1
documents the behavior instead of faking a boundary.

Written: `docs-site/guide/browser-tool.md` (verbs, install by scope, the grant
tiers, the two things it needs, managed vs TUI), a line in
`docs-site/guide/packages.md`, the sidebar entry, and the scope paragraph in
`docs/plans/desktop-v2.md`. Owner validates in use; scope expansion is a later
conversation.

**Found:** npm's `pi-browser` is someone else's Playwright-backed extension
that also registers a tool named `browser`; the Packages gallery searches npm,
so a gallery search offers that one. Ours is local-only today — no breakage —
but publishing needs a name decision. Recorded in `work-browser-tabs.md`.
visual-review: n/a (docs only).
Not done: the grants editor, the shell navigation gate, runtime acceptance.

## Next up

- Grants editor (Settings ▸ Browser) so `act` becomes reachable.
- Shell navigation gate mirroring `browser.AllowsOrigin`.
