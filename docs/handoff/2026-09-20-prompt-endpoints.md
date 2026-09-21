# 2026-09-20 — prompt door: the ADR-0143 wire rows, tested (prompt-endpoints)

Branch cut from bc3dfa6e; commit 32fae2fb "server: wire tests for the prompt door's identity rows".

- The board debt is paid: ADR-0143's endpoint branch has tests of its own. The
  resolver's decision rows were already covered (TestResolveCallerIsTheHouseIdentity);
  the wire rows were not.
- `TestPolicySaveTerminalWireRows`: a known terminal saves under `term:<id>` and lists
  as the terminal row carrying the grant; an unknown term id is a 404 that names
  itself; agent+term together is refused 400 ("not both"), leaking nothing.
- `TestPromptDoorIdentityWire`: unknown terminal id → 404; stopped terminal → 409
  named `closed` (real tmux, no session — the shim row only covered the unclassified
  failure); a terminal bound to a stopped pi agent → 409 "not running in a terminal"
  (the agent branch refuses, never falls through to paste into a dead pane);
  agent→its bound running terminal delivers, the paste landing in that session.
- No defect found: every row already behaved as documented (ADR-0089's table,
  ADR-0143's precedence) — tests-only, no changelog fragment.
- Finding, not fixed: `GET /api/browser/policies` iterates `ListTerminals()`
  unfiltered, so a plain shell gets a principal row; ADR-0143 says "the terminals
  that have a CLI running", named "title and CLI" (the row uses `t.Name` only).
- Evidence: `go test ./internal/server/...` ok (239s); `make ci-scoped` PASS.

## Next up
- Owner call: filter the listing to CLI-holding terminals, or accept the wider list.
