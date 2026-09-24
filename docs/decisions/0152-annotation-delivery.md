# ADR-0152: annotation-delivery — one staged file, its path in the prompt door

- **Status**: accepted (amended 2026-09-19: one note per Send, no cap on pins)
- **Date**: 2026-09-18
- **Boundary**: persistence (a new table and a staging-folder convention) and the user→agent input contract — what reaches an agent's terminal, and by whose act

## Boundary

Persistence (a new table and a staging-folder convention) and the user→agent
input contract (what reaches an agent's terminal, and by whose act).

## Context

v2c's step 4 was named as needing an ADR: how an annotation reaches the agent.
The field was studied ([2026-09-18](../benchmarks/2026-09-18-annotation-design-mode.md)):
Orca, Lovable and bolt.diy all deliver the capture as **one attachment in the
conversation**; our own attach study (2026-09-06) fixed the transport —
**paths, never bytes** — with the prompt door staging files under
`<cwd>/.picode/drop/`. The owner approved three choices (2026-09-18): a staged
file plus its path in a message; a single element first, multi later; and an
`Ask` asked at creation time rather than holding a turn.

## Decision

1. An annotation is a row in `browser_annotations` holding the pointer: page,
   title, host, selector, comment, the DOM and computed CSS (clipped at 64 KB
   with an honest marker) and the **file names** staged for it.
2. The bytes live in two files under `<terminal cwd>/.picode/drop/`: a small
   markdown note (the human's sentence first, then page, element and captures)
   and the crop, ≤ 4 MB.
3. Delivery is the **existing prompt door** with those paths and the human's
   caption. No new input path, and nothing is auto-sent — ADR-0078 stands.
4. The terminal's `cwd` is resolved from the terminal id in the store, never
   taken from the request, so a caller cannot stage a file outside a workspace.
5. Deleting a row deletes only the row: the files stay (the agent may still be
   reading them) and the drop folder's own sweep removes old ones.

## Consequences

- The picker and the capture are the remaining v2c work. The endpoints exist
  before their caller on purpose: the daemon half is verifiable on Linux while
  the picker is Windows-only, so neither half waits on the other.
- The screenshot policy row (Always include / Ask / Never) lands with the
  picker. An unanswered `Ask` proceeds **without the image** — the inverse of
  the permission watchdog, which must deny.
- Annotations are announced on the feed (`browserannotation.updated`), so a
  future list surface has one source of truth.

## Amendment (2026-09-19): one note per Send, and no cap on how many pins it carries

The owner exercised v2c live and asked for **no limit on the number of
annotations** (2026-09-19). The decision above staged one note per pin, so a
Send of N pins produced 2N files, and the prompt door's own cap of four files
per paste (ADR-0089, decision 1 — unchanged, and unchanged *for this reason*)
stopped a three-pin Send from delivering at all: it failed visibly with
"at most 4 files" and left the rows in the store.

What changes, and why it is the smallest honest change:

1. **The note is per Send, not per pin.** `POST /api/browser/annotations`
   takes the whole set (`items[]`), stages one crop per pin and writes **one**
   markdown document: the count and the page, then one section per pin (its
   sentence as a heading, its element, its HTML, its styles, its crop's file
   name). This is the reference's "N annotations" as a single context, and it
   is what makes N unbounded: the paste carries the note first.
2. **The paste carries the note plus the crops the door has room for.**
   `pastePaths` fills the door's four files, note first; the ones that do not
   fit stay staged beside the note, which names every one of them, and the
   message says how many stayed ("3 more screenshots staged next to the
   note"). ADR-0089's cap keeps its meaning — a bound on *attachments per
   paste*, not on how many things a human may annotate.
3. **A Send is staged whole or not at all.** Every image is decoded and
   checked before anything is written, so a bad item cannot leave half a
   package on disk and half in the store.
4. Rows stay one per pin (the store's shape is unchanged); they share the
   note name, which is the package they arrived in.

Consequences: the number of annotations is bounded only by the human and the
disk; the per-file cap (4 MB) still applies per crop; a very large Send costs
the agent one document plus the pictures it chooses to open.

### Decision table (every row tested)

| Pins | Camera | Files staged | Pasted | Message adds |
|---|---|---|---|---|
| 1 | on | note + 1 crop | both (2 ≤ 4) | — |
| 3 | on | note + 3 crops | both (4 ≤ 4) | — |
| 4 | on | note + 4 crops | note + 3 crops | "1 more screenshot staged next to the note." |
| N | off | note only | note | — (the strip says the camera was off) |
| N | on, one image bad | nothing | nothing | the Send fails, the store keeps no row |
| 0 | — | nothing | nothing | "an annotation needs at least one item" |
