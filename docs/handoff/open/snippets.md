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
- **Body decode is uncapped** on the snips POST/PATCH handlers (the
  Pins pattern); newer endpoints (checklist, drop) use
  `MaxBytesReader`. Harden if abuse ever matters — paired clients only
  today.
- **`snip.ran` skips 404 attempts** (unknown snip id returns before the
  event is armed). The letter of "every attempt" would emit with the
  requested id.
- **The shell door does not consult `repoBusy`.** A confirmed command
  can run while an agent works the same repository — same blast radius
  as the human typing it, and confirm is the gate, but say it out loud
  if that ever changes.
- **Multiline shell bodies** execute line-by-line (bracketed paste +
  one Enter, bash parses the whole buffer). Authoring guidance only;
  the editor does not warn.
- **Adversarial round (2026-09-13) fixed:** studio archive/star now
  refetch (archived rows used to linger until reload); `preview:true`
  no longer logs `snip.ran` (previews are not runs) and is 400 on
  prompt kind; `{{branch}}` resolves from the live folder on `/run`
  (agent and terminal) via `currentBranch` — it used to expand empty.
