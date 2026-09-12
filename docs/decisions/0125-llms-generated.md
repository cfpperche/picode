# ADR-0125: llms.txt is generated where it is served, not committed

- **Status**: accepted
- **Date**: 2026-09-12
- **Boundary**: persistence — which build outputs live in git, and which are produced at the point of use

## Context

`docs-site/public/llms.txt` is a whole-file, deterministic render of the docs
site's structure (title, summary, curated page list, API pointers) for LLM
crawlers. It was committed, and `make close` regenerated and committed it
whenever a branch touched `docs-site/` or `internal/`: 6 commits in ten days,
each one rewriting the file end to end, plus a `docs-check` gate that failed
the whole run when the committed copy lagged a source edit (this branch hit it:
one edited guide page, one red `make close`).

Two artifacts of the same shape are treated differently, and the difference is
who reads them. `docs-site/public/api/openapi.json` is *content the site
renders*: the API page loads it into Scalar's viewer at runtime and links it as
a download, so a local `npm run dev` (which has no Go toolchain in the loop)
must find it in the tree — it stays versioned and parity-checked. `llms.txt`
has exactly one consumer: a crawler that reads the deployed site. Nothing in
the repository or the site loads it.

## Decision

`docs-site/public/llms.txt` is produced where it is served — `make docs`
(VitePress build, GitHub Pages) and `make deploy` — and is git-ignored. No
branch carries it, `make close` no longer regenerates or commits it, and
`docs-check` no longer checks parity for it (there is no committed copy to be
stale). `make llms` keeps writing the file on demand for a manual look.

## Consequences

- One whole-file artifact leaves the merge surface; the last two commits of a
  branch cannot fight another branch's regeneration of it.
- A clone that never built the docs has no `llms.txt`, which is correct: the
  built site has one because `make docs` runs before VitePress.
- The gate that caught staleness disappears with the file. It was catching a
  problem that only exists because the file was committed; a mistaken `llms`
  render now ships at the next build, and the crawler-facing content is
  derived from the same sources the site itself renders.
- If this is wrong (crawlers mattering more than a merge surface), restoring it
  is one `git add -f docs-site/public/llms.txt` plus the parity check.

## Alternatives considered

- **Keep it committed and make the check advisory** (like the public capture
  fingerprints). The commit noise stays, and `docs-check` would stop noticing
  a stale file entirely — worse than either.
- **Treat `openapi.json` the same way.** It is rendered by the site's API page
  at runtime; generating it only in `make docs` breaks a plain VitePress
  `npm run dev` session, which is how the docs are actually iterated on.
- **Generate it in CI before the checks** (so the file is always current in a
  run). CI is not where the file is read — the site is — and it would put a
  write into a read-only job.
