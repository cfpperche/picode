# 2026-09-22 — feat/ci-go-timeout: the Go gate outgrew its ceiling

The CI run that verified the three test-hermeticity fixes came back `Go (ubuntu-latest) failure` with **no failing test**: `FAIL github.com/cfpperche/picode/internal/server 1800.070s` — exactly `-timeout 30m`. The package sat at 1691 s on the 0.5.0-era suite (94% of the ceiling) and ADR-0184's launch tests, which spawn real terminals, took it over.
Fix: the ceiling is 50m, with the measurement written into the comment beside it.
Not touched: `GO_TEST_SHARDS=1`. The workflow comment records it as a measured decision ("a hosted runner has two cores… and the extra concurrent processes each start their own tmux server, which is what broke internal/tmux"), so changing it is the owner's call, not maintenance.
Debt recorded in `docs/handoff/open/process.md`, cheapest candidate first: a CI job of its own for the heavy packages; sharding revisited — the reason it was turned off was concurrent tmux *servers*, and the tmux tests now keep to their own socket and server; trimming the suite, 365 serial tests by design (ADR-0086).
Verified: `make ci-scoped` PASS (full) — a workflow path is global scope, so all 78 packages ran, `internal/server` in 4 shards at 105.9 s each — and the YAML parses.
No fragment: a CI ceiling is not user-visible.
Blind spot: whether 50m is enough is only knowable from the next run; the suite's true duration was never measured past 30m.
