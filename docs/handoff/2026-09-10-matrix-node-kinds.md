# 2026-09-10 — feat/matrix-node-kinds (Matrix v2 phase C3)

Base 99b63173 · three commits, one kind each: `note`, then `file`, then `diff`.

## What shipped

`kind` was one validator line and stays one: `agent, terminal, note, file, diff`, no migration. A `note`'s ref is a pin id; a `file`'s and a `diff`'s is `<owner>:<id>:<path>` with the desktop's own owner letters (ADR-0030), refused on both sides with the same sentence and taken apart or put together **only** by `parseRef` / `buildRef` in `web/shared/domain/matrix.js`. The store stays ignorant of all of it (ADR-0108): a pin that does not exist, an owner that is gone and a file that was never written are all stored, and `bindingState` — not the schema — says so. The unique index is per (kind, ref), so one path is a `file` panel and a `diff` panel at once.

Bodies: `NotePanel` (one `GET /api/pins/{id}`, `react-markdown` + `remarkGfm` over the app's `.md`, refetched on that pin's `pin.updated`), `FilePanel` (`FilePane` embedded), `DiffPanel` (`WorkingDiff` under a nonce that `git.updated` bumps through `gitTouches`). No timer anywhere.

Two rules moved into the pure module. `hasPane` is now `PANE_STATES` in `matrix.js`, and `loadPolicy` takes a per-entry `pane`: a body with no `term.buffer` never goes still — it reads normally from 0.4 up and is a name-plate below it, one test per row. And unsaved work is never unmounted silently: `FilePane` reports `onDirty`, the surface `keep`s that panel (a pin that survives a hidden matrix), and `docKey` moves the document into `lib/fileDocs.js` so maximize and the mode switch keep the text.

## Verdicts

`uiux-review: PASS` — read before the first JSX/CSS edit. `visual-review: PASS` — captures in `var/screenshots/` (never committed), dark and light, each read: the picker with its five groups, the picker with no file tab open, the grid with all four kinds, the canvas name-plates at 35 %, the gone rows. Cards 5/5. `__picodeOverlayAudit()` `ok` with the picker open. Four defects the reads found were fixed in-session, none shipped: a dead **Close** in the embedded editor (it only renders where there is an `onClose`), a lone pin title thrown to the right of its row by `.palette-item`'s `space-between`, the kind chip truncating to *No…* in a narrow header, and the editor's bar wrapping onto a second row. A fifth was a real bug the pixels found: maximizing a dirty editor dropped the *Unsaved* chip and its pin, because the old body's unmount cleared the flag the new one had just set.

The browser table is in `docs/architecture/matrix.md` (*Accepted in a browser (2026-09-10, C3)*).

## Debts

The picker is the only entry point: an *Add to matrix* row on an Inspector change, or in a file tab's menu, needs the desktop to know which matrix is open and where a panel would land — state that lives inside the Matrix app (ADR-0109), so it is a decision, not a line. The Files group's empty line has no button of its own (its next action is a file tab, outside the dialog; the dialog's Close is right below it). A `note` and a `diff` body subscribe to the feed directly rather than through `host.feed`, as the Inspector does. A dirty document held after its body unmounts is freed by removing the panel, not by an LRU. No `docs-shots` capture (still needs a `desktop-matrix` profile).
