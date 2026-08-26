---
name: quality-gate
description: Run PiCode's code quality gates before declaring work done — fmt, vet, tests, build, changelog. UI work without visual-review PASS (screenshot read) is this gate FAIL.
---

# Quality gate

Run this checklist **before declaring any code work done** in PiCode.
Rules come from `docs/benchmarks.md` (engineering section); the contract
is in `AGENTS.md`. Be honest — a failed gate reported honestly beats a
passed gate reported falsely.

## Steps

1. **Formatting**: run `gofmt -l .` from the repo root.
   - Any output = fix with `gofmt -w` before proceeding.
2. **Vet**: run `go vet ./...`. Must be clean.
3. **Tests**: run `go test ./...`. All pass, no skips without a linked
   issue/TODO in `docs/handoff.md`.
4. **UI build** (if `web/` changed): `cd web && npm run build`. Must succeed.
5. **Build**: run `make build` (UI + Go) or `go build ./cmd/picode` after `make web`. Must compile.
5b. **Running binary**: the process on :8445 MUST be the binary you just built (`go:embed` is compile-time). After a UI/Go build, run `make restart` or overwrite `bin/picode` so the running process self-reloads (mtime watch). Incognito still sees yesterday's UI if the old process is alive. Verify: `curl -sk https://localhost:8445/` asset names match `internal/web/public/index.html`.
6. **Diff review**: `git diff --stat` —
   - One logical change? If not, propose splitting the commit.
   - New code without tests? Add tests (table-driven) before finishing.
   - Interacting conditions (delete/restore/auth/cascade/mode)? A decision
     table exists and every row is tested or named as debt. Two clicks ≠ matrix.
   - Timed UI (jobs, overlays, lists)? Motion on enter/step/exit. Optimistic
     next state, not a static wait then a jump.
   - New non-stdlib dependency? It needs explicit justification in the
     commit/PR description (AGENTS.md rule #3). No justification = remove it.
7. **Changelog check**: is anything in this diff user-visible?
   - Yes → there must be a new entry under `[Unreleased]` in
     `CHANGELOG.md` (Keep a Changelog verbs: Added/Changed/Fixed/Removed).
8. **Docs check**: did behavior or architecture change?
   - Yes → `docs/architecture.md` (and an ADR if architectural) must
     change in the same commit.
9. **Visual gate** (if `web/` or any user-facing surface changed):
   `read` `/skill:visual-review` and `/skill:uiux-review`. Must be PASS
   (empty/blocked/error screenshots **read**, overlayAudit ok, visual-card
   in the reply). Skipped or FAIL → this quality-gate is FAIL.
   `eval` JSON is not a visual pass.
10. **Handoff**: run `/skill:handoff-update` to close the session state.

## Report format

End with a one-line verdict, e.g.:

```
quality-gate: PASS (fmt ✓ vet ✓ 12 tests ✓ build ✓ visual ✓ changelog +1)
quality-gate: FAIL (visual: overlay clipped — see visual-review)
quality-gate: FAIL (vet: 1 finding in internal/server/server.go:42)
```
