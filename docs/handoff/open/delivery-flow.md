# Delivery flow

Plan: docs/plans/delivery-flow.md
Research: docs/benchmarks/2026-09-21-delivery-governance.md
D0 evidence: docs/plans/delivery-flow-evidence.md
D0 surface contract: docs/plans/delivery-flow-design.md

## Next


## Debts

- [x] The tool picker had no delivery option, so it could not be switched on for an agent whose launch settings had been customized — a launch that never touched them already received every family — paid 2026-09-21 (feat/delivery-tool): `PICODE_TOOL_FAMILIES` offers **Delivery**, `internal/server/cli_tools_test.go` holds that list against `mcptool.FamilyNames()`, and the scratch proved the path form → config → plan (`mcp_servers.picode-delivery.args=["mcp","delivery"]`). `packages/pi-delivery` gives an agent whose CLI is Pi the same tool, held against the MCP schema by `internal/mcptool/delivery_test.go`.
- [x] Vendor CLI processes have exercised the delivery tool end to end (2026-09-21, scratch only) — **all eight catalog CLIs whose binary was on this machine**: **Claude Code** (headless with PiCode's `--mcp-config`, and driven through PiCode's prompt door), **Codex** (failed first — see the identity debt below — passes after the fix), **OpenCode** (its MCP children inherit; the live call was not made because the account's default model is outside the owner's plan), **grok**, **omp**, **Antigravity (`agy`)** (all three inherit the launch environment), **Hermes Agent** and **Muse Code** (declared with the identity set explicitly on the server entry, which those two require). Every declaration landed with `principal` = the agent id and the project read shows it. All eight run under `--dangerously-skip-permissions`/`--yolo`-style flags where their print mode demanded a tool approval.
- [ ] Hermes and Muse hand their MCP servers an **allowlist** of environment variables (13 and 8 names; measured with an env probe — Codex hands none, `agy`/grok/omp/claude/opencode inherit), so a PiCode tool server reaches them with no identity and answers `no identity`. A literal per-server `env` works for both (measured: 0 → 2 `PICODE_*`), but a connector written at user/workspace scope has no per-terminal value to write — the launch path would have to carry it, as it now does for Codex (a design decision, not a fix).
- [ ] Historical land/deploy queue wait has no authoritative start timestamp; D0 measured seven source lookups only. Capture owner-task time in the observation pilot and operation wait after D3/D4.
- [ ] D1b reports incomplete coverage after 1,000 receipt files/changes or 32 checkout statuses; add paginated older evidence and on-demand omitted checkout reads before scaling beyond the measured local pilot. No evidence is pruned.
- [ ] Native Windows ACL inheritance and physical mobile behavior were not exercised by the Linux scratch observation tests.
