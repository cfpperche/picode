# 2026-09-20 — feat/captures-refresh: public captures the merges left inconsistent

Shipped: `docs-site/img/app-canvas.png`, `app-inspector.png` and the matching
`manifest.json` entry, regenerated with `make docs-shots` on today's `main`.

Why: `make ci` on `main` failed at `docs-check` with
`app-canvas.png hash mismatch — edited by hand?`. It was not a hand edit — the
capture and its manifest entry drifted apart while four branches merged their
own captured images through the day (the picture and the hash it is verified
against are one generated pair, so half a merge of that pair is an
inconsistency by construction). Neither my JSX-only branch nor the branch that
landed before it touched those files.

Verified: `node scripts/docs-check.mjs` → `docs-check ok`; `make ci-scoped`
PASS (fmt, vet, hooks, docs); `make close` green, `main` fast-forward ready.
This branch changes generated artifacts only — no source, no behaviour, so no
changelog fragment.

Lesson worth keeping: when a merge conflicts in `docs-site/img/*`, take one
side and immediately run `make docs-shots` on the merged tree; taking "theirs"
and moving on is what leaves the pair inconsistent.

visual-review: n/a — no interface change (the captures are the interface's own
record, regenerated, not edited).
