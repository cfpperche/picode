# 2026-09-10 — work-back-land: Back from an agent/terminal lands on its Work view

Shipped: mobile Back no longer dumps you on a generic Work section. `parentHash`
gained an owner argument (`web/mobile/src/lib/mobileRoutes.js`): with a
workspace it returns `#/work/workspaces/<wsId>` (parsed as route.id), a free
agent → `#/work/agents`, free/unknown terminal → `#/work/terminals`, unknown
agent owner (fleet not answered) → legacy `#/work`. Ownership helpers in
`web/mobile/src/lib/workBack.js` (`agentOwnerWs`, `termOwnerWs` — a payload
without workspaceId counts as free). `Work.jsx` scrolls the owning
`.m-work-group` just below the sticky `.m-list-head` (offset via its height;
re-runs on list changes so feed patches/clamping settle; no-op once placed);
`PullScreen` forwards the scroll box via `surfaceRef` (callback or ref).

Verified: node --test (356 mobile, incl. new mobileRoutes + workBack decision
tables), `make ci-scoped` PASS, scratch instance wbl: agent→group, terminal→group,
free agent→Agents, free terminal→Terminals, section persisted to localStorage,
`__picodeOverlayAudit` ok.
visual-review: PASS (wbl-3c-back-to-workspace.png, wbl-4-back-free-term.png read)
Not done / debts: desktop app untouched (its sidebar keeps context); a search
query held in `rememberedQuery` can filter out the focused group (scroll no-ops,
acceptable); scrolled-into-view has no highlight flash (v1 kept plain).
Merge: fast-forward ready.
