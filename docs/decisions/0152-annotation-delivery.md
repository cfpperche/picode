# ADR-0152: annotation-delivery

- **Status**: accepted | accepted | superseded by ADR-XXXX
- **Date**: 2026-09-18
- **Boundary**: protocol | persistence | security model | process — which one, and what crosses it

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
