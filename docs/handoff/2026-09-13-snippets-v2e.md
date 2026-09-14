# 2026-09-13 — snippets-v2e: the address check and durable drafts (F8, F4)

Shipped: `GET /api/snips/slug/{slug}` (204 free / 409 taken, `?except=<id>`
so a snippet's own address is never a clash; static path, so `/api/snips/{id}`
cannot eat it) and the editor's debounced check — a red line under the Slug
field, Save off while it is taken, a failed check never blocking a save.
Drafts moved `sessionStorage` → `localStorage` in both editors (same keys and
`draftToRestore` base rule). Hydration is now state, not a ref: with a ref the
draft write ran in the hydration commit with `f` still pristine, one transient
render in which storage held an empty draft.

Verified: store test (taken / own / normalized / archived rows / empty),
server test (204, 409, `?except=`, and that the slug route is not matched by
`/{id}`), 16 `snipDraft` tests; live on scratch `scratch` — taken state
(red line + Save off), free state (line normal, Save on), own slug on edit,
and F4 by typing into the editor and **reloading the page**: the text comes
back and storage keeps it. `__picodeOverlayAudit()` ok in the taken and the
invalid-body states (the reason badge beside Save is gone; a text span in a
`data-align-row` is the misalignment the audit exists to catch).
Pixel PASS: editor in the taken-slug state.
Honest note: I first read "draft wiped on the next visit" as a product bug —
it was my test (a closed agent-browser session gets a fresh profile, so
localStorage is legitimately empty). The reload test above is the real F4
check, and the ref version passed it too, so the ordering fix closes a
transient window rather than a reproduced loss.
Merge: fast-forward ready.
