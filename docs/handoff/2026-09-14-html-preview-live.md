# 2026-09-14 — feat/html-preview-live: HTML preview v1.5 (live reload + preview-only)

Shipped: the two v1.5 items in `docs/plans/html-preview.md`. **Live reload** —
`preview.Store.Touch` records every file a ticket serves (cap `MaxWatch` = 200,
document first), `GET /preview/<token>/__events` (SSE, same ticket + session
gate) stats them every second and emits one `change` frame when an mtime/size
moved (first sighting seeded silently; a vanished path reads `gone`). The
pane's `usePreviewTicket` (browser + mobile) holds one EventSource, bumps a
`v=` nonce on the iframe `src`, and the parent reloads the frame — no script
injection, sandbox untouched; five stream failures close it and leave the
manual Reload. **Preview-only** — a `.html` over the pane's ≤1 MiB text read
now mints and renders from disk with Reload/Open and no "too large" banner
(no Raw: there is nothing to edit). Docs in `docs/architecture/file-preview.md`.
Verified: Go tests (watch set/cap/expiry; serve records paths; SSE
hello/seed/change at a 10 ms poll; 404/405 refusals); `make ci-scoped` PASS;
frontend 784+357+337 tests. Scratch `live` (:8472): on-disk document and asset
changes reloaded the desktop frame with no click, a document change reloaded
the 390×844 mobile frame, `big.html` (1.1 MiB) rendered preview-only on both,
empty state unchanged; every screenshot read; `__picodeOverlayAudit()` ok.
visual-review: PASS (card 5/5)
Not done: unsaved-editor-text preview (v1.5 remainder), v2 (a real second
origin). QA note: the first agent-browser session wedged right after the first
live reload; `close --all` + a fresh session recovered it — server-side SSE
emitted 0 spurious events, so it was session state, not a reload loop.
Merge: fast-forward ready pending the owner's review.

## Next up

- Unsaved-editor-text preview (parent PUTs the editor text to the ticket route
  as a memory overlay) — the last v1.5 item; v2 needs its own ADR.
