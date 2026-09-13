# Git watcher

## Debts

- Removing a workspace (or the last agent in one) **panics the whole server**: `diffGit` names a path that left `groups` on this tick, and `internal/server/git_watch.go:75` dereferences `groups[path]` (nil). Reproduced 2026-09-13 on scratch `ggws`: `POST /api/workspaces` whose path is a repo, wait one 3 s tick, `DELETE` it, wait one tick → `GET /api/health` `000` and the panic in `var/qa/ggws/server.log`. Found while QA-ing the graph's workspace picker; not that branch's regression. Fix: skip a path absent from `groups` (the removal is what the event would report).
