# 2026-09-19 — agent-menu-consistency: consistent workspace agent menus

Implemented: shared agent lifecycle wording and ordering, scoped Pi launch settings,
mode-preserving Pi restart, confirmation/busy/error feedback and separated removal.
Documented the capability/state decision table in managed-principals architecture;
menu tests cover Pi modes, CLI states, bindings and unsupported launch adapters.
Verified: ci-scoped passed on the preceding draft; scratch browser QA passed on the
final implementation for menus, scoped routes, start/stop/restart dispatch,
confirmation cancellation and failed managed stop preventing replacement start.
Lifecycle responses were simulated in the browser; no real model process was run.
visual-review: PASS — empty, capability-blocked, menus, confirmations, restart error
and light/dark hover screenshots read in a subagent; overlayAudit ok, card 5/5.
Evidence: var/screenshots/agent-menu/ and var/agent-menu-qa.log (gitignored).
No protocol, persistence or security boundary changed; deployment was not performed.
Merge: fast-forward ready at close-summary against main 9d660ca3.
