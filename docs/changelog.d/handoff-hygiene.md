### Changed

- **Six snippet debts were invisible on the handoff board.** The board renders
  the bullets under a topic file's `## Next` / `## Debts` headings, and
  `docs/handoff/open/snippets.md` had neither — so its debts never reached the
  sessions that read the board. It has the heading now, consolidated to the
  three that carry the meaning; the roadmap lives in the plan.
- Also pruned there and elsewhere: bullets that only restated a plan file, a
  debt already carried by its owning topic, and the 0.2.0 release step (cut;
  tag `v0.2.0` exists).
