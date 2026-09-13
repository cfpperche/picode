# Snippets — open debts

- **Visual pixel review.** The branch was verified on scratch instances
  with `__picodeOverlayAudit()` (geometry) and DOM snapshots; the
  authoring harness could not read PNGs, so no pixel-level
  visual-review PASS exists. Re-run `/skill:visual-review` on
  `#/snippets` (empty/list/edit/delete), the run sheet (agent + shell
  confirm) and the mobile flows from a harness that reads images.
- **Palette shell rows.** The design's E2–E4 wanted **Run command: …**
  palette rows for shell snippets; v1 ships the term-menu / actions
  path only. Revisit if palette invocation of commands is wanted.
- **Enum editing.** `placeholders[].enum` round-trips and is enforced
  server-side and by the run sheet `<select>`, but no studio UI edits
  enums yet (API/JSON only).
