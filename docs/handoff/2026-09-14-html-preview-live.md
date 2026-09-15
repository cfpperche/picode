# 2026-09-14 — feat/html-preview-live: HTML preview v1.5 (live reload + preview-only)

Shipped: the two v1.5 items in `docs/plans/html-preview.md`. **Live reload** —
`preview.Store.Touch` records each file a ticket serves (cap `MaxWatch` = 200,
document first); `GET /preview/<token>/__events` (SSE, same ticket + session
gate) stats them every second and emits one `change` frame when an mtime/size
moved (first sighting seeded silently; vanished reads `gone`). The pane's
`usePreviewTicket` (browser + mobile) holds one EventSource and bumps a `v=`
nonce on the iframe `src`, so the parent reloads the frame — no injection,
sandbox untouched; five stream failures close it and leave the manual Reload.
**Preview-only** — a `.html` over the pane's ≤1 MiB text read mints and renders
from disk with Reload/Open and no "too large" banner (no Raw to edit).
Docs: `docs/architecture/file-preview.md`, plan phase 2.
Verified: Go watch/SSE/refusal tests at a 10 ms poll; `make ci-scoped` PASS;
frontend 784+357+337. Scratch `live` (:8472): on-disk document and asset
changes reloaded the desktop frame with no click; a document change reloaded
the 390×844 mobile frame; `big.html` (1.1 MiB) preview-only on both; empty
state unchanged; every screenshot read; `__picodeOverlayAudit()` ok.
visual-review: PASS (card 5/5)
Not done: unsaved-editor-text preview (v1.5 remainder), v2 (its own ADR). QA:
the first agent-browser session wedged after the first live reload; `close
--all` + a fresh session recovered it — SSE emitted 0 spurious events.
Merge: fast-forward ready, pending the owner's review.

## Next up

- Unsaved-editor-text preview: parent PUTs the editor text to the ticket route.
