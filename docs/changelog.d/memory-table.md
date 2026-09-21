### Added

- **The Memory pane is a table you can audit with.** Beside every memory it now
  shows how many other memories cite it, its size, when it last changed, and
  the things worth acting on: a memory the index does not name, and a link that
  points at a file that is gone. Any column sorts, the kind chips filter with
  their counts, and the file the CLI loads every session stays pinned at the
  top. On a phone the same numbers ride under the title with a Sort control.
- **The index budget.** Claude Code reads only the first 200 lines or 25 KB of
  `MEMORY.md` at the start of a session and drops the rest in silence. The pane
  now shows how close the index is to that limit, and names any row in it that
  points at a file that no longer exists.
- **Delete several memories at once,** with a confirm that names what it
  breaks: how many links in other memories will point at nothing, and how many
  of them the index still names.

### Fixed

- Codex's Memory pane no longer offers to copy `codex` as "the vendor's own
  command" — that command does nothing to a memory, and the pane now shows the
  note alone.
