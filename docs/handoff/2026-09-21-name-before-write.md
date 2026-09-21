# 2026-09-21 — name-before-write: the naming prompt moved before the write

Shipped: `POST /api/credentials/import` accepts `preview: true` — Detect plus
the identity resolution plus a `Key()` lookup answer what the import would do
(no write; `created`, `existing`, `who`, `stamp`). The pane's Check now runs
the preview first and asks the name BEFORE writing when the write would
replace an unnamed row; the typed name is also what the row shows
(`isDefaultLabel` renames the counter label). The same-login stamp guard
stays.

Verified: `TestCredentialImportPreviewDoesNotWrite` (no write, existing row
named out, named import separates), the unnamed/named server tests, and a
scratch walkthrough — prompt open with the vault file mtime unchanged, name
typed with the native setter, row labeled. Blind spot: three QA runs shared
one scratch, so the final roster shows extra synced rows; the per-step
assertions were made against fresh imports.

visual-review: PASS (nbw-prompt.png: "Name this login" open before any write)
Not done: when the person cancels the name, the import still replaces the
unnamed row (honest toast, but the old tokens are gone — the vendor's
single-file model leaves no way back).

## Next up

- Consider offering "keep the old login" in the same prompt: it would require
  naming the OLD row from the preview's `existing` — the pieces are all in
  place.
