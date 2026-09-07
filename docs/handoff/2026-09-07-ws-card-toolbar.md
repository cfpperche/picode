# 2026-09-07 — feat/ws-card-toolbar: the workspace card reaches its own files and history

Shipped: the sidebar's workspace header carries the card's own folder/branch
pills (`wsLine`, the same `ContextLine` its rows use) — branch opens
`#/git/w/<id>`, folder `#/tree/w/<id>` — so a project with no agents and no
terminals reaches its history. The server has answered `/api/workspaces/{id}/git`
since ADR-0030; the gap was the browser. Four latent holes closed with it:
`GitGraphSurface`/`CommitDetail` sent a workspace owner to `/api/agents/` (now
`ownerBase`, `web/shared/domain/gitOwner.js`), `readGitOwners` downgraded it to an
agent on reload, `gitHash`/`gitRoute` knew no `w`, and the hash liveness check
looked a workspace up among the agents — one `ownerExists` now answers for file,
git and tree tabs. Toolbar: five hover-only icon buttons became a **New** menu
(agent · shell terminal · Agent CLI terminal) and one overflow (Files, Git graph,
Sessions, Remove), both visible at rest, Git graph hidden on a non-repository and
Sessions while the workspace has no agents (409). Empty state gained its actions.
Study: `docs/benchmarks/2026-09-07-workspace-card-toolbar.md`.

Verified: `make ci-scoped` and `make close` PASS (captures refreshed). Live on the
docs fixture with a runtime-registered empty git workspace: graph, commit detail
and uncommitted row read through the workspace, the tab survived a reload, both
menus, both empty-state actions and the folder pill exercised.

visual-review: PASS (sidebar default/narrow/collapsed, graph, commit, uncommitted,
both menus; dark theme via the refreshed `www/img/app-fleet.png`)

Debt: at the 180 px minimum a *collapsed* card with four or more occupants
ellipsizes its name hard — the new 56 px trigger gutter costs it that much.
Merge: `git merge --ff-only feat/ws-card-toolbar && make ci` from main.
