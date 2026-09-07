# 2026-09-07 — feat/ws-card-line: the workspace card header is one line again

Follow-up to `feat/ws-card-toolbar` (deployed `0.1.0+0dd28c0`), on the owner's
call after reading it on the real fleet: the card's own path/branch line was
noise — every agent row under the card repeats the same path and branch, so
the header stated a third time what the rows already say. The `ContextLine`
comes off the workspace header (and its CSS block with it); the two controls
stay. Files and Git graph still read through the workspace, so a project with
no agents and no terminals reaches its files and its history — the entry point
is the menu, not a pill. `wsLine` stays: the menu asks it whether the folder is
a repository before offering its history. `ContextLine` is private again.

The withdrawal is recorded in `docs/benchmarks/2026-09-07-workspace-card-toolbar.md`
(adopted, then withdrawn the same day) rather than deleted from the study.

Verified: `make close` PASS (captures refreshed — the sidebar images shrank by
~8 KB, which is the removed line). Live on the docs fixture with a
runtime-registered empty git workspace: the menu still opens the graph
(`#/git/w/<id>`, two rows), the collapsed card still shows its face strip,
the empty state keeps its two actions.

visual-review: PASS (sidebar expanded and collapsed, graph from the menu)

Merge: `git merge --ff-only feat/ws-card-line && make ci` from main.
