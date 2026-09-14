### Changed

- **The handoff board is an index now, not a ledger.** Next steps still render
  inline (they are bounded and they are what a session acts on), but debts are
  one line per topic — the count, the topic file and its plan. Rendering every
  debt made the board grow with the backlog, which is what filled it; it is
  about half the size and no longer trips its budget on one honest entry.
  `make handoff` now **fails, naming the file**, when a topic file lists bullets
  outside its `## Next` / `## Debts` headings — six snippet debts were invisible
  that way for a day.
