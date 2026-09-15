# 2026-09-14 — feat/html-preview-unsaved: HTML preview v1.5 unsaved overlay

Shipped: the last v1.5 item — plan phase 2 is complete.
`PUT /preview/<token>/<document>` takes the pane's editor buffer as the
ticket's overlay (`preview.Store.SetOverlay`, in-memory, same ticket + session
gate, only the ticket's own document, UTF-8 ≤1 MiB); `GET` serves the overlay
under the same sandbox headers, assets still from disk. The hook PUTs while the
buffer is dirty and reloads the frame; a clean buffer re-mints so disk wins.
The "Unsaved changes aren't in the preview." notice and its Save-and-preview
button are gone (browser + mobile); the dirty dot and Save stay.
Docs: `docs/architecture/file-preview.md`, the plan, `docs-site/guide/files.md`.
Verified: `make ci-scoped` PASS; Go tests (overlay under sandbox headers, assets
unaffected, only-own-document, 400/413/404/405, store overlay/expiry); shared
PUT tests; frontend 786+357+338. Scratch `unsaved` (:8473): disk render → Raw
edit → Preview showed the unsaved text (read) → Save wrote it to disk and
cleared the dirty dot → a following on-disk change reloaded the frame (no
stale overlay); mobile pane renders; `__picodeOverlayAudit()` ok.
visual-review: PASS (card 5/5)
Not done: v2 only — a real second origin (storage/workers), its own ADR.
Merge: fast-forward ready, pending the owner's review.
