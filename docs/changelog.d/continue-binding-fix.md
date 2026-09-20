### Fixed

- Bind newly created interactive Pi sessions as soon as their JSONL file appears,
  so the agent menu exposes `Continue in…` without a manual Sessions refresh.
- Allow a bounded Muse index-update window when pinning a terminal's latest
  session, reducing the shutdown race before the index is refreshed.
