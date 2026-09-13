# Snippets — open debts

- **Visual pixel review (partly paid, 2026-09-13).** Pixel PASS now on
  the v2 authoring surfaces (table + Try-it, capture sheet, import dialog
  empty and filled, context menu, mobile import sheet and editor). Still
  owed: the phone's list/empty/delete states and the run sheet (agent +
  shell confirm).
- **Palette shell rows.** The design's E2–E4 wanted **Run command: …**
  palette rows for shell snippets; v1 ships the term-menu / actions
  path only. Revisit if palette invocation of commands is wanted.
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
