# Licensing

PiCode is Apache-2.0 since ADR-0218 (`feat/apache-license`, 2026-09-25);
`packages/` stay MIT (ADR-0028). Paid team-control features live only under a
top-level `ee/` with its own commercial license. Detail: `LICENSING.md`.

## Next

- Owner pushes to GitHub (publishing the Apache versions is irrevocable); then confirm GitHub detects `Apache-2.0`, not `NOASSERTION`: `gh api repos/cfpperche/picode --jq .license`.
- Write `ee/LICENSE` (commercial, lawyer-reviewed) in the branch that lands the first `ee/` file; none exists yet.
- Register the trademark once the pending project rename is decided: Apache-2.0 grants no trademark rights (section 6).
- `NOTICE` names `cfpperche` as copyright holder; switch it to the legal entity (with a copyright assignment) if a company is incorporated.

## Debts

- [ ] `docs/benchmarks/2026-09-14-tachyon-fleet-orchestration.md` line 6 still says PiCode is PolyForm; left as-is because the study is dated. Revisit if its cross-license statement matters.
- [x] `scripts/adr-new.sh` breaks when `TITLE` contains `/` (it is the sed delimiter) and leaves an orphan `NNNN` file behind; found while seeding ADR-0218. Paid on feat/adr-new-title: the title is escaped for sed and the file is written through a temp file.
