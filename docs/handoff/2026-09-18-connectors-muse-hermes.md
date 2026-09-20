# 2026-09-18 — feat/connectors-muse-hermes: Guest drivers Muse + Hermes (Phase 4, ADR-0150)
Shipped: Phase 4 closes connector parity — every catalog CLI now has a Connectors pane (9/9, ADR-0150).
Muse: one settings file at ~/.config/muse/settings.json; mcp_servers block {transport: stdio|streamable_http,
command/args/env | url/headers, enabled toggle}; schema_version + mode + unknown keys survive (JSON codec,
map + indent 2); project scope refuses ("no per-workspace file"). Hermes: one config at ~/.hermes/config.yaml;
edited as a yaml.Node tree (new dep gopkg.in/yaml.v3 v3.0.1, justified in the commit per ADR-0150 decision 3)
so comments, key order, auth/tools fields survive and ${VAR} stays verbatim; malformed YAML refuses before any
write. Both: AuthHint copyable "muse|hermes mcp login <name>", Live "" (honest "configured"), presets
vendor-neutral, import/auth refusals by the phase-1 pattern. Dispatch: connectors.For + connectorDrivers += muse,
hermes. UI: two CONNECTOR_DRIVERS entries; zero JSX/CSS (pane is data-driven). TestMCPRejectsCLIWithoutDriver
moved from cli=muse to cli=cursor (muse has a driver now).
Verified: go build ./... ; go test ./internal/... -count=1 green (llama TestConnectionTimeoutAndTransport
flake re-run alone: ok) ; cd web && npm test 0 fail ; node --test integrations.test.js 8/8 ; make web ok ;
make ci-scoped PASS (full). Fixtures pin HOME/USERPROFILE/XDG_CONFIG_HOME inside t.TempDir(); mtime scan
before/after the suite: only a live .grok process bumped its own logs/sessions — zero config files touched.
Merge: fast-forward ready

## Next up

- Phase 5 — definition import between CLIs + status parity (live reporting where a signal exists).

## Debts

- Muse/Hermes config formats written from vendor docs (settings.json mcp_servers + schema_version; config.yaml mcp_servers) unverified against real muse/hermes binaries; accumulates with the phase-3 opencode/grok debt under topic docs/handoff/open/connectors-parity.md (not yet created — first writer seeds it).
