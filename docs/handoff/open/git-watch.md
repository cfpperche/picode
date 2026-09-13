# Git watcher

## Debts

- Removing a workspace (or its last agent) panics the server: `git_watch.go:75` dereferences `groups[path]` (nil) for a gone path. Fix: skip a path absent from `groups`.
