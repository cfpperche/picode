# 2026-09-22 — feat/docs-dev-flow: the development flow page

Shipped: `docs-site/guide/dev-flow.md` — "Development flow": the change lifecycle (elaboration → worktree iteration → closing docs → owner landing →
owner deploy → post-deploy cleanup) as a hand-built HTML/CSS diagram (no mermaid dependency, `--vp-c-*` vars, `.vp-doc`-scoped), two tables and a
what-this-page-is-not section; new "Development flow" item in the Start sidebar. Nine prose-lint words added to
`styles/config/vocabularies/PiCode/accept.txt` (worktree(s), handoff, subagent, hardlinked, gitignored, auth, paseo) — `make vale` needs them.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, docs), `make docs` green, `make vale` 0 errors, run before `main` moved 7 commits — rerun on the merged branch.
Adversarial review (independent subagent): PASS; three defects fixed — `make land` needed `BRANCH=feat/cascade-delete` (worktree.sh creates `feat/<name>`),
the code-and-docs row overstated ci-scoped/close (the close-summary flags and pre-commit hook enforce it), and a raw-HTML h3 skipped a heading level.
visual-review: PASS — dark 1440, light 1440, dark 360 read from `var/screenshots/devflow-*.webp` (not committed): no horizontal overflow at 360px,
nothing clipped, arrows `aria-hidden`, AA contrast in both themes. Blind spot: Chromium at three widths only, never Firefox or Safari.
Merge: one commit `docs: add the development flow page`; merged and green on `main` (42e04f08); outline debt moved to `docs/handoff/open/docs-audit.md`.
