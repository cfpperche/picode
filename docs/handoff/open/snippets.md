# Snippets — open debts

- **Visual pixel review (partly paid, 2026-09-13).** The v2 authoring
  surfaces have a pixel PASS from a harness that reads screenshots:
  placeholder table + Try-it, capture sheet, import dialog (empty and
  filled), the context menu and the mobile import sheet + mobile editor.
  Still owed: `#/snippets` empty/list/delete states on the phone and the
  run sheet (agent + shell confirm).
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
- **Adversarial round (2026-09-13) fixed:** studio archive/star now
  refetch (archived rows used to linger until reload); `preview:true`
  no longer logs `snip.ran` (previews are not runs) and is 400 on
  prompt kind; `{{branch}}` resolves from the live folder on `/run`
  (agent and terminal) via `currentBranch` — it used to expand empty.

## Next

- **Snippets v2, PRs 5–8** (`docs/plans/snippets-v2.md`): starters +
  duplicate (F6), slug availability check + drafts in `localStorage`
  (F4/F8), the mobile editor v2 (placeholder table, enums, live
  validation — the phone still has the v1 form), then the docs-site
  guide.
- **Capture parity for `Command` kind.** The capture sheet and import
  always create a `prompt`; a shell snippet still needs the Kind select
  in the editor.
