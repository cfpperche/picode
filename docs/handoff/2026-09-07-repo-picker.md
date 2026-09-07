# 2026-09-07 — feat/repo-picker: filterable GitHub repo picker in the clone form

Shipped: `GET /api/github/repos` (`internal/server/github_repos.go`) lists the
machine's `gh` repositories (`gh auth status` + `gh repo list --limit 100 --json`,
20 s cap, 5-min in-memory cache, failures uncached, `?refresh=1` for Retry) and
answers `{available, reason?, repos?}` so blocked setups carry a visible reason.
Desktop `RepoField.jsx` turns the clone URL field into a cmdk combobox (grouped
by owner, lock on private repos, "Use this URL" row for pasted links, keyboard
first); mobile `RepoField.jsx` adds a tap-to-open picker next to the URL field.
Shared fetch mapping in `web/shared/client/githubRepos.js` (+5 tests).

Verified: Go table tests (gh missing / not logged in / list fails / ok-filters /
cache + refresh bypass) green; 1005 JS tests; `make ci-scoped` PASS; live on the
scratch instance with a `gh` fixture — open/filter/pick/derive, toggle, Escape,
blocked and URL-pasted states; overlay audit ok.

visual-review: PASS (repo-picker-blocked/open/filtered/groups + mobile ×2, card 5/5)

Not done / debts: picker covers GitHub only (`gh` CLI); other hosts stay on the
paste-URL path. Real-account pick was exercised against a fixture `gh`, not a
live `gh repo list`.

Merge: fast-forward ready (`git merge --ff-only feat/repo-picker && make ci`).
