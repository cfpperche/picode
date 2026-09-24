# 2026-09-24 — skills-scopes: fix scope chips and order on the Skills tab

Shipped: chips Global / workspace / agent (spec field `AgentScope`, set for
pi and omp; the report names the agent only when it is this CLI's own agent
and in that workspace; the agent chip shows what that agent loads, or that
it runs isolated, with "Open its packages"). "This machine" renamed to
"Global" (house term); "All" removed. Add skill dialog says Global and lists
it first. Chip order fixed: Global, then the workspace by name, then the
agent by name. The count line follows the selected chip. The trust notice
shows only under workspace/agent chips. The declaration-order gate caught a
read-before-declaration in the desktop pane (would have blanked it on load);
fixed.
Verified: Go tests (`TestAgentScopeOnlyForPiAndOmp`), domain tests
(`cliSkills.test.js`), `make ci-scoped` PASS, scratch QA with API + DOM
checks.
visual-review: PASS (round 1 FAIL — count line summed every scope while the
table showed one chip; fixed and recaptured).
Blind spot: the agent chip cannot yet list skills that only one agent loads
(per-agent skill sets are slice 4).
Merge: fast-forward ready.
