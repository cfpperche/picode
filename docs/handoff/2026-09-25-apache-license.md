# 2026-09-25 — apache-license: PiCode relicensed under Apache-2.0

Shipped: ADR-0218 (accepted, owner-approved 2026-09-25) moves PiCode from
PolyForm Noncommercial plus a signed commercial license to Apache-2.0.
`LICENSE` is the verbatim apache.org text; `LICENSE-COMMERCIAL.md` is gone;
`NOTICE`, `LICENSING.md`, README badge and section, CONTRIBUTING,
`docs-site/license.md`, getting-started, two architecture docs and the
pi-roles/pi-compact READMEs follow. ADR-0028 amended (packages stay MIT);
ADR-0098 amended (SignPath stays out of reach once `ee/` exists).
Rule: paid team-control features live only under a top-level `ee/` (none
exists yet), with a commercial `ee/LICENSE` when its first file lands.
Verified: dependency audit of Go modules (`cmd/...`), web production npm
deps and desktop-shell crates finds no copyleft; MPL-2.0 and OFL-1.1 fonts
are weak/compatible only. docs-check, vale and `make close` green.
visual-review: n/a
Not done / debts: publishing, `ee/LICENSE`, trademark and the copyright
holder are tracked in `docs/handoff/open/licensing.md`.
Merge: fast-forward ready (bb75fa4f1 plus this note).
Follow-up feat/adr-new-title: `make adr` escapes TITLE for sed and writes via a temp file (no orphan on failure).
Same branch: markdown Live tests read a full parse (flaked twice on today's gates; proven with a 22 KB doc).

## Next up

- Owner pushes to GitHub, then checks the detected license (open/licensing.md).
