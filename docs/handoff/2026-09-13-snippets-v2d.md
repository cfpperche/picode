# 2026-09-13 — snippets-v2d: starters and duplicate (F6)

Shipped: `GET /api/snips/templates` serves six built-in starters from
`internal/snips/templates.go` (code, not rows — the `automate.Templates`
pattern; static path, so it cannot be eaten by `/{id}`). The studio's empty
page shows them as a grid; once the reader has snippets the grid becomes a
remembered `<details>` (Automations pattern). Picking one hands the body,
tags, kind and a derived slug to the editor as a draft with origin
`starter`. **Duplicate** on a list row and in the snippet: a row carries no
body, so it reads the snippet first; the copy gets `title + " copy"` and the
slug `<slug>-copy`, locked so editing the title does not re-derive it, and
the original is never touched.

Verified: `go test ./internal/snips ./internal/server` (starters parse and
expand, ids unique, kind valid, the door answers 200); live on scratch `pr5`
— empty page grid (6 tiles), starter → editor with slug `standup-update`,
Save → row with the derived slug, duplicate from a row (body fetched) and
from the detail (body already loaded), title edit does not move the locked
slug, both rows saved with distinct slugs. `__picodeOverlayAudit()` ok.
Pixel PASS: empty page (grid), list with two rows (4 aligned row actions),
starter editor.
Not done / debts: the phone's starter grid is still PR 7 scope; no shell
starter yet (all six are `prompt`).
Merge: fast-forward ready.
