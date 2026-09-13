# Git watcher

## Debts

- Removing a workspace (or its last agent) panics the server: `diffGit` names a path that left `groups` that tick, and `internal/server/git_watch.go:75` dereferences `groups[path]` (nil) — reproduced 2026-09-13 on scratch `ggws` (POST a repo workspace, a 3 s tick, `DELETE`, a tick → health `000`). Fix: skip a path absent from `groups`.
