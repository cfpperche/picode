# ADR-0116: A Matrix edge grants a mailbox contact, never a transcript

- **Status**: proposed — the owner approves or rejects before any code (plan `docs/plans/matrix-canvas.md` §6.1)
- **Date**: 2026-09-10
- **Boundary**: security model — who may reach whom between sessions, what an
  edge grants, who may draw one, and what is refused permanently. Extends
  ADR-0104's contact rule; nothing here changes persistence or the wire beyond
  one table and two events.

## Context

The Matrix canvas (ADR-0113, phases C0–C3) draws agents, terminals, notes,
files and diffs as panels on a plane. The benchmark that inspired it
([nodeterm](https://nodeterm.dev), studied in
`docs/benchmarks/2026-09-10-node-canvas.md`) draws **edges** between agent
nodes and describes them as "context links" that let agents "read each other's
transcripts on demand". The owner asked for the same spatial idea. The plan
parked it behind this ADR precisely because that sentence hides a security
decision, not a drawing feature.

**What PiCode already has.** ADR-0104 gives every enrolled session a mailbox:
`list_contacts`, `send_message`, `read_messages`, `ack_messages` over MCP, with
a per-conversation bearer stored only as a hash and revalidated inside each
store transaction. Two facts decide this ADR:

1. **Contacts are scoped to one workspace.** `PeerContacts` reads
   `WHERE workspace_id = me.workspace_id AND revoked_at IS NULL`, then keeps
   only connections whose recorded session is still current. Two enrolled
   sessions in the same workspace can already message each other with no edge
   at all. Two sessions in **different** workspaces cannot see each other, at
   any price short of an owner moving one.
2. **Enrolling and revoking is an owner action**, done in the Messages view
   (`#/clis/messages`). No agent enrolls itself.

**What a transcript is.** A session transcript is everything that was pasted,
read or printed into that conversation: file contents, command output,
customer data, and — routinely — credentials. PiCode's own operating rules
refuse sharing credentials across conversations, and ADR-0104 deliberately
keeps message bodies out of the event feed for the same reason. "Let agent A
read agent B's transcript" is not a convenience feature; it is a lateral
disclosure channel that no revocation can walk back, because reading is
copying.

**What the owner actually wants**, read from the request rather than from the
benchmark's wording: to see that two sessions work together, to make them work
together without leaving the canvas, and to do that even when they live in
different project folders.

## Decision

**An edge is the owner's recorded intent that two sessions may exchange
messages. It grants exactly ADR-0104's contact, and nothing else.**

1. **No transcript access, permanently.** An edge never lets one session read
   another's session file, scrollback, buffer or history. The supported way to
   get context out of another session is to **ask it**: send a message and let
   it answer under its own judgment, with its own redactions — the same shape
   as the Inspector's existing "Ask ⟨agent⟩" (ADR-0078), which enqueues a
   prompt instead of reading the agent's state.
2. **Only the owner draws an edge**, in the browser, through the authenticated
   owner API. The MCP surface gains no verb: an agent can neither create an
   edge nor discover that one could exist. A capability whose beneficiary can
   grant it to itself is not a capability.
3. **Cross-workspace pairing is the one new power**, and it exists only per
   pair. `PeerContacts` becomes the union of *same workspace* (unchanged) and
   *linked by a live edge*. The workspace default is not loosened, and there is
   no global "let everything talk" switch — the blast radius of an edge is two
   named sessions, drawn by hand, visible on the plane.
4. **Removing the edge revokes the grant.** The grant is *derived* from live
   edges on every contact read, never cached, never copied into
   `peer_connections`. Deleting the matrix, either panel, or the edge itself
   removes it. There is no state that outlives the line you can see.
5. **An edge never enrols silently.** Drawing between two panels whose targets
   are not enrolled offers the existing owner enrolment, names what it means in
   one line, and does nothing until confirmed. Drawing across workspaces
   confirms separately, naming both folders.
6. **An edge is undirected.** The mailbox models a symmetric contact; a drawn
   arrow would promise a one-way restriction the store cannot keep.
7. **A broken end does not grant.** An edge whose panel is gone, whose session
   changed, or whose connection was revoked reads as broken on the plane and
   contributes no contact — ADR-0104 already invalidates a connection whose
   recorded session moved, and this ADR does not soften that.

**Data.** One table, cascading from the matrix:

```
matrix_edges(id, matrix_id → matrices ON DELETE CASCADE,
             a_panel, b_panel, created_at,
             UNIQUE(matrix_id, a_panel, b_panel))
```

The pair is stored ordered (`a_panel < b_panel`) so the unique index refuses
the same edge drawn twice in either direction. Edges reference **panels**, not
sessions: a panel already carries `(kind, ref)`, and re-pointing a panel is a
new binding that must be drawn again. An edge between panels that are not
`agent` or `terminal` is refused — a note has no mailbox. Events
`matrix.edge.added` / `matrix.edge.removed` carry ids only, like every other
event here (ADR-0048).

## Consequences

- **Easier**: seeing which sessions may reach each other, and pairing two of
  them across project folders without an owner-wide rule change.
- **Harder**: "may talk" now has two sources — the workspace and the edges —
  so `PeerContacts` is a union that must stay correct under revocation,
  session change and matrix deletion. Its decision table is the test suite.
- **A security control drawn with a mouse must look like one.** A confirm on
  the cross-workspace case, a non-spatial list of every live edge in the
  Messages view (the canvas is not the only place a grant may be audited), and
  a broken edge that reads broken rather than dotted-and-ignorable.
- **The benchmark's headline feature is deliberately not matched.** Someone
  comparing PiCode to nodeterm will find no transcript sharing. That is the
  decision, not an omission, and the guide says so in plain words.
- **If we are wrong**, edges are one table and one union clause: dropping both
  returns the mailbox to workspace scope with no migration of anything else.
  Nothing was copied while an edge existed, because nothing could be.

## Alternatives considered

- **Edges read transcripts (the benchmark's model).** Rejected: reading is
  copying, revocation cannot undo it, transcripts routinely carry credentials,
  and the useful part — "what did you find?" — is answered better by the
  session itself than by its raw log.
- **Edges are decoration.** Rejected: a line between two agents that suggests
  a channel which does not exist is worse than no line. Every benchmark that
  draws a wire makes it mean something.
- **No edges; keep the Messages view as the only pairing surface.** Honest and
  cheap, and it stays the fallback if this is rejected. It loses the spatial
  reading the owner asked for, and it cannot express a cross-workspace pair at
  all.
- **A per-matrix mailbox scope** (an edge grants contact only "inside this
  matrix"). Rejected: the mailbox has no per-matrix dimension, and inventing a
  scope the transport cannot enforce would be a lie told in the UI.
- **Directed edges with one-way contact.** Rejected for v1: `peer_connections`
  is symmetric, so the arrow would be decoration on top of a grant that is not.
- **Agents may propose an edge for the owner to accept.** Deferred, not
  refused: it needs an inbox item kind and a way to name the two sessions
  without leaking either one's content into the other's request. Worth its own
  decision if the need appears.
