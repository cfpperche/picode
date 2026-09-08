# 2026-09-08 — feat/docs-site-rename: public docs site www/ → docs-site/

Shipped: `git mv www docs-site` in one atomic commit (52 renames + 31 files
updated). `www/` is reserved for the future product website; Pages URL
unchanged (cfpperche.github.io/picode). Updated together: Makefile
(`WWW_STAMP` → `DOCS_STAMP`), pages.yml (trigger/cache/artifact paths),
ci.yml npm cache, scripts (close, deploy-batch, worktree, ci-scope + tests,
docs-llms `docsSite` var, docs-check, docs-shots, docs-video-manifest),
VitePress editLink, .gitignore, .vale.ini, picode-openapi comments, living
docs (AGENTS, CONTRIBUTING, README, docs/README, guidelines, benchmarks,
architecture, screenshots README, design roadmaps, uiux-review skill,
docs-videos/BRIEF — its stale `www/videos/` path fixed too). ADR-0086
emended in place (owner-approved, "née www/"); ADR index row notes it.
Verified: `make ci` full matrix PASS in the worktree (an earlier FAIL was a
shared go-build cache race with parallel sessions — clean on rerun);
ci-scope unit tests green; Vale 0 errors / 29 files; repo sweep shows only
intentional `www` mentions left (CHANGELOG entry, ADR née, favicon probe).
visual-review: n/a (no UI change).
Not done / debts: screenshot fingerprints are stale-advisory
(docs-shots.mjs hashes into the fingerprint) — the next `make deploy-batch`
recaptures on its own. Historical records keep the old path on purpose:
docs/handoff/*, docs/handoff-archive.md, docs/benchmarks/2026-*.
Merge: fast-forward ready.
