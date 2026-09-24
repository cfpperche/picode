# 2026-09-24 — feat/mission-pi-native: native Pi Missions tool
Shipped: embedded Pi `mission` extension, launch wiring for managed RPC, interactive agent Pi and PiCode Pi terminals, with existing `/api/missions/tool` attribution and authority.
Boundary: `/api/missions/tool` wire contract is unchanged; no `nativeSession` request field was added. Writes derive session identity from PiCode state. Assigned reads remain available before session binding. Proposed ADR-0213 narrowly amends ADR-0200 for first-session binding authority.
Verified: Pi 0.87.1 scratch RPC and TUI catalog probes both returned `["read","bash","powershell","edit","write","grep","find","ls","mission"]`; the TUI listed `pi-mission.ts`. Isolated `qa-scratch` managed Pi dogfood acknowledged and reported Mission `mission_ZWIHZGBI3OMZYNW65EBAJ2ZULV` at v3/v4 as `mission-reporter-ecbd92`; both writes persisted session `pi:/home/goat/picode/.worktrees/mission-pi-native/var/qa/mission-pi-native-20260924/home/.pi/agent/sessions/mission-reporter-ecbd92/2026-09-24T15-43-47-247Z_73b46741-50bc-4b96-9de9-4d21defa65ec.jsonl`. A separate interactive PiCode Pi terminal acknowledged and reported Mission `mission_5D23D6DEWOKIYZPR6XLSEQRZSJ` at v3/v4 as `mission-terminal-reporter-2c5eb6`, bound to `pi:45953b13-67c7-420f-aea0-0cbcd76f35a6`.
Provider: copied Anthropic OAuth failed refresh with `invalid_grant` before tool use. Using only the prior Pi Settings pilot's `model_change` metadata (`xai/grok-4.6`), both isolated Pi runs completed. No credential values or native transcript were retained. Scratch agents and daemon were stopped; port 8472 is free.
Limits: end-to-end tool calls are verified for one Pi version and `xai/grok-4.6`; other providers remain unverified. The TUI probe plus separate live terminal Mission proves interactive calls. Go tests cover mutation-field validation, retries, first-session binding, stale-session refusal, and pre-binding reads. Final `make close` and full `make ci` passed.
visual-review: n/a
Not done / debts: native Pi endpoint dogfood is complete in isolated managed and interactive scratch sessions; broader provider validation remains open.
Merge: Mission `mission_QG5WFYAYDAASAGSO4JYEI5VFNX` accepted v15 at reviewed revision `f5ca6f38e`; branch landed by fast-forward at `08f7aa06b` and full CI passed. No deploy. Sanitized scratch results and accepted Mission are under `var/qa/missions-pilot/`.

## Next up

- Owner may deploy from the root when ready; native Pi reporting becomes available after the running PiCode binary is updated.

## Debts

- [x] Native Pi managed and interactive scratch Missions both persisted acknowledge and report under PiCode-observed sessions (2026-09-24); exact IDs, actors, versions and session bindings are recorded above. Broader provider validation remains open in `docs/handoff/open/missions.md`.
