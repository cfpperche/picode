### Added
- **Agent history access** (Settings ▸ Browser): an agent may read where you
  have been — off unless you allow it, its own permission rather than a side
  effect of a grant. The `browser` tool gained the `history` verb (a search
  over url and title, newest first, capped at 200 rows) and answers with url,
  title and time only: no page content, no sessions. ADR-0146.
