# ADR-0177: Delivery's publication lens is removed from the product

- **Status**: accepted
- **Date**: 2026-09-21
- **Boundary**: protocol and persistence — the delivery read no longer returns
  `environments[]` or a per-change `publication`, and the workspace observer
  binding leaves the product; process — the deploy producer stops recording
  receipts for a product surface. Supersedes the D2 half of ADR-0170; the D1
  declaration and observation boundaries are unchanged.

## Context

ADR-0170 approved observing publication through revision-bound evidence and
left D2 — associate the local environment, record deployment attempts, verify
the running revision — as a planned slice. That slice was implemented the same
day (`feat/d2-deployment`, commits `88a54f8c`…`6d1e982e`): a per-workspace
binding (migration 065), deployment receipts under `<data>/var/delivery/`
written by `picode deploy`, a full-revision runtime identity, `environments[]`
plus a per-change `publication` in the owner-scoped read, and a second lens in
the Delivery view on desktop and phone.

The owner reviewed it live, on the instance it was built for, and rejected the
shape: the only environment the vocabulary could name was `picode-self`, so the
lens answered "is the PiCode binary serving me behind the repository it was
built from?" — a question that exists only when the project **is** PiCode's own
repository. For any other project the same lane reads `unconfigured`, or, once
connected, `revision-unmapped` forever, because the running revision is not a
revision of *that* repository. The owner's rule is explicit: a surface that only
serves the development environment does not belong in the product — if the
capability is not for any software project with a repository, it is not for
PiCode. ADR-0170's fixed scope already said the pilot's own repository is the
first measured workflow; measured, it turned out to be the *only* one.

## Decision

Remove the publication lens from the product. The Delivery view returns to a
single Integration lens on both surfaces; the owner-scoped read keeps D1's
fields and drops `environments[]` and `publication`; the workspace observer
binding, its route, its store methods and its event leave the code; the deploy
receipt producer leaves `internal/install`; the runtime identity additions
(`version.Revision`, `/api/version`'s `revision`, the discovery file's
`revision`/`boot`, `server.BootID`) and the guard's coverage variant leave with
their only consumer.

If environments return, they must be **project-owned markers**: a release tag or
branch of the project's own repository, or receipts the project's own tooling
writes into that repository's evidence directory (`<common-git-dir>/picode-delivery/`,
where D1's receipts already live). That direction is a recording, not a
commitment — it needs its own boundary decision, because reading a configured
marker and comparing it against the integration branch is protocol work, and
because "not released in v2.3.0" is a weaker, honest claim than "not in
production".

## Consequences

Easier: the Delivery view carries no surface that cannot do anything; the
delivery code is one lens, one observer, one receipt reader; nothing in the
product knows the difference between PiCode's checkout and an application's.

Harder: the question "is what is running older than `main`?" has no answer
again for PiCode's own deployment; that information is now only in
`~/.picode/var/deploy-log.jsonl` and the operator's own reading.

Accepted costs: migration 065 keeps its number and its table with no reader
(deleting the file would risk a future migration 065 being silently skipped on
every database that already applied it); the two receipts the deploy wrote while
the lens existed stay on disk in `<data>/var/delivery/` — no evidence is pruned
— with nothing to read them. If this is wrong, the honest recovery is to
re-land `88a54f8c` and then fix the vocabulary, not to leave a permanently
unknown lane in the product.

## Alternatives considered

- **Keep the lens for PiCode's own repository only.** Rejected by the owner as
  contamination: the product would carry a control that means nothing to any
  customer, and "this instance" would remain the vocabulary.
- **Generalize the environment now** (release tag or branch, plus a receipt
  directory the project's tooling writes). Recorded above as the direction; not
  taken now because it is a slice of its own — new selector, a producer that
  writes into the repository, a public receipt envelope for third-party
  tooling, and a read contract that names a project-owned marker. Building it on
  the same day the data said the owner does not want the surface would be
  guessing.
- **Leave it inert** (visible, `unconfigured`) until a later decision. Rejected:
  a permanently unknown lane in every project is exactly the surface the owner
  refused.
