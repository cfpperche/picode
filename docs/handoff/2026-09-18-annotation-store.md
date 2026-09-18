# 2026-09-18 — annotation-store (v2c step 3)

The daemon half of browser annotations: the store, the endpoints and the two
staged files. ADR-0152 records the delivery contract.

## Landed

- Migration **055** `browser_annotations` + store (create/list/search/delete,
  `browserannotation.updated` on the feed, row in the invariant test).
- `POST/GET/DELETE /api/browser/annotations`: stages `<note>.md` (the human's
  sentence first, then page/element/HTML/CSS) and the crop into
  `<terminal cwd>/.picode/drop/` — the folder the **existing prompt door**
  already reads. The cwd comes from the terminal id, never the request.
- Caps: image 4 MB (413), DOM/CSS clipped at 64 KB with a `…[truncated]` marker.
- Study: `docs/benchmarks/2026-09-18-annotation-design-mode.md` + README row.

## Verified

- Store table test (five rows: url, something to look at, host normalization,
  search by comment and by host, the 200 clamp, double delete).
- **Live on a scratch**: terminal created with a known cwd → POST 200, host
  normalized, the `.md` (291 B) and a real PNG on disk (`file` says 1×1 RGB),
  the note's first line the human's sentence, search/list/delete (204), and the
  refusals — unknown terminal 400, nothing to look at 400, an oversized image
  413.
- `make ci` green; deployed.

## Debt

- The endpoint has no Go test of its own (no server-test harness was touched):
  the live scratch run above is the evidence. Worth a harness test with the
  picker slice, together with the wire rows of the grants endpoint.

## Not here (deliberate)

- The **picker** (step 1) and the **capture** (step 2) — Windows-only, next
  slice with the shell's `Agent Browser` eval/capture path.
- No changelog fragment: nothing user-visible changed yet. The row
  (Always include / Ask / Never) lands with the picker — a switch that controls
  nothing is a dead control (desktop-v2's own rule).
