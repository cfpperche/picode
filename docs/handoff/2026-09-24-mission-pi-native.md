# 2026-09-24 — feat/mission-pi-native: native Pi Missions tool
Shipped: embedded Pi `mission` extension, launch wiring for managed RPC, interactive agent Pi and PiCode Pi terminals, with existing `/api/missions/tool` attribution and authority.
Boundary: `/api/missions/tool` wire contract is unchanged; no `nativeSession` request field was added. Writes derive session identity from PiCode state. Assigned reads remain available before session binding. Proposed ADR-0213 narrowly amends ADR-0200 for first-session binding authority.
Verified: `make ci-scoped` and `make close` passed on the committed candidate; it is clean and fast-forward ready. Pi 0.87.1 scratch RPC and TUI probes used isolated HOME, data and session directories, with no live daemon. A probe extension read `pi.getAllTools()` at `session_start`; both returned `["read","bash","powershell","edit","write","grep","find","ls","mission"]`; TUI also listed `pi-mission.ts` as loaded.
Limits: probes prove registration in Pi 0.87.1 only. They did not call the mission endpoint or validate a live assigned write; no provider/model was configured. Go tests cover integration, stale-session refusal, and show/context access before session binding.
visual-review: n/a
Not done / debts: owner dogfood must exercise acknowledgement and reporting in managed and interactive Pi against a live mission.
Merge: ready for owner review and separate fast-forward; no deploy.

## Next up

- Owner dogfood of a real assigned Pi mission in managed and interactive modes; confirm first acknowledgement binds the PiCode-observed session.

## Debts

- [ ] Mission tool registration is verified on Pi 0.87.1 scratch; live attributed writes in managed and interactive Pi remain unverified. See `docs/handoff/open/missions.md`.
