# 2026-09-08 — packages-config: package configuration GUI (pi-roles), Packages list v2

Shipped: ADR-0099 (supersedes ADR-0033 §5 "No PiCode GUI page" for known
adapters). `pipkg.ConfigKindOf` adapter registry; GET/PUT/DELETE
`/api/packages/config` — roles: workspace `.pi/roles.json` + agent overlay
under `store.AgentCwd`, effective merge, unknown-key-preserving atomic writes,
409 before overwriting an unparsable file, `packages.config` feed event.
Desktop: installed packages are a filterable row list with **Configure**;
`#/packages/config/pi-roles` edits both layers with scoped reset (override
seeded from inherited / use workspace value / clear file after confirm).
Verified: go test (pipkg + server table tests), test-js 751, build, vale,
ci-scoped PASS; scratch instance — save round-trip proven on disk (overlay
thinking=high written), override/use-workspace-value cycle, empty and invalid
states with controls locked; `__picodeOverlayAudit()` ok:true (two alignment
fixes: pkc-foot line, pkc-row-head untagged like .mcp-row).
visual-review: PASS (pkg-list, pkg-config-workspace, pkg-config-agent,
pkg-config-empty, pkg-config-invalid all read; card 5/5).
Not done / debts: mobile Packages has no config UI (API additive, mobile
untouched); mid-edit conflict detection deferred — 409 covers unparsable
files only; adapter registry is code — declarative manifest waits for
pi-compact (delivery 2); live-agent application not exercised against a
running pi (copy says "next message", matching the extension's per-input
reload — reasoned, not watched).
Merge: fast-forward ready.
