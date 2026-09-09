# 2026-09-09 — docs-adversarial: adversarial docs-vs-code sweep; picode --version

Shipped: five confirmed drifts fixed. `picode version`/`--version`/`-v` prints the
build identity (ADR-0098's install verifier runs it; the flag used to fall through
to `serve()` and start a server). docs-site `api.md` pairs via `picode pair`
(`/api/auth/pair/start` never existed) and no longer claims `PICODE_INSECURE=1`
skips pairing (auth mode Off is the ungated path). ADR index: blank line that broke
the table from 0090 down removed. routes.md: `fileDocument.js` path updated to the
post-ADR-0072 desktop/mobile split. Fragment: `docs/changelog.d/docs-adversarial.md`.
Verified: sweep covered /api endpoints (generated + hand-written), make targets,
PICODE_* env vars, file paths, Go symbols, feed events, CLI catalog, pairing flow.
`make close` green on the merged tree (go + docs + vale); fragment validated by the
fixed `--check`.
visual-review: n/a (no UI change)
Not done / debts: the sweep hit the pre-fix changelog hook (main's 6794260a fixed
it); my first commit accidentally folded `process-review.md` — repaired here by
restoring CHANGELOG.md and carrying my own fragment only. Worth knowing: hook
escape hatch (`PICODE_ALLOW_SWITCH=1`) was needed once for that repair.
Merge: fast-forward ready (`git merge --ff-only feat/docs-adversarial && make ci`).
