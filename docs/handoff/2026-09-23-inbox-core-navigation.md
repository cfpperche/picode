# 2026-09-23 — feat/inbox-core-navigation: Inbox becomes core navigation

Built: ADR-0208 moves the Inbox view/action/badge into `internal/inboxview` and `/api/inbox`, with old app API and hash aliases. The desktop header opens `#/inbox` with a pending badge; Apps no longer lists Inbox. Desktop and mobile use the core route, and saved Inbox app tabs are dropped.
Docs: architecture, public guide, OpenAPI, changelog fragment, and affected screenshots updated.
Verified: `make ci-scoped` passed fmt, vet, hooks, six Go packages, JS tests, build, and docs; strict docs-check passed. One flaky tmux test passed on isolated rerun.
visual-review: PASS — scratch browser screenshots read for empty, pending, done, error, old link, Apps grid, mobile, and 180px sidebar; overlay audit passed and no horizontal clipping was seen. The docs-site captures were also read; `app-providers` shows the intermediate sign-in step described by its caption.
Limit: scratch Chromium review does not establish native Windows shell behavior.
Merge: `make close` and `make land` remain to be run after reconciling the branch with current `main`.
