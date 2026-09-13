# Git watcher

## Debts

- Removing a workspace (or its last agent) panics the server: `git_watch.go:75` dereferences `groups[path]` (nil) after `diffGit` names a gone path — reproduced 2026-09-13. Fix: skip a path absent from `groups`.
