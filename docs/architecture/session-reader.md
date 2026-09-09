# SessionReader

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Parses Pi session JSONL files (version 3, tree-structured via `id`/`parentId`)
to render session history, branching and diffs in the UI. Read-only. The
cross-CLI readers and writers (ADR-0088) live in `internal/clisession`,
not here.
