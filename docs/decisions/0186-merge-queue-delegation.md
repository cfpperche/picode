# ADR-0186: Integration delegates to the project's merge queue; the local runner is the fallback

- **Status**: accepted (owner approved in session, 2026-09-22)
- **Date**: 2026-09-22
- **Boundary**: process — which engine performs an integration, and who owns the order when a project already has a queue; security model — that the authority to drive a hosting provider is the owner's own authentication, never an agent's; protocol — what PiCode writes to a provider (an enqueue, a removal) versus what it only reads (position, outcome)

## Context

The study for exactly this question was written one day before ADR-0182 and was
not cited by it: `docs/benchmarks/2026-09-21-delivery-governance.md`. Two of its
rows decide the engine. **t3code** documents "auto-merge pending checks, and
GitHub stack operations respecting branch rules and merge queues", and the
prescribed adaptation is **"reuse provider governance"**. **GitHub's merge
queue** documents ordered integration that validates a change against the current
target *and* the entries ahead of it, with removals that expose their reasons —
and the limit says "provider eligibility and merge-group CI configuration
constrain direct reuse". The study's own recommendation is to start with
trustworthy observations and blockers, *then* add explicit queue governance. The
execution baseline, `docs/plans/delivery-flow.md`, is blunter: its non-goals are
"no generic shell-command executor", "no mandatory GitHub/PR workflow", "no
replacement CI/CD platform", and its goal is to "allow other projects to use
different integration and publication providers".

ADR-0182 nevertheless made PiCode's own runner the engine, and D3 slices 1-3
built it: a durable queue, the owner's and the agent's doors, and a
declared-command executor with a local fast-forward. It works and it is tested —
but it is a second mechanism beside the one most projects already trust, and it
was decided without the evidence the repository already held. In owner review
(2026-09-22) the direction is: adopt the community's semantics instead of
inventing our own vocabulary for them, and delegate to the provider wherever a
project has one.

## Decision

An integration is performed by **the project's provider when it has one, through
that provider's own queue**: PiCode enqueues with the provider's tooling — the
repository's GitHub merge queue, or a reviewer's `r+` where the project runs
bors — and then **observes** that queue: the entry is present or not, its
position, and its ejection with the provider's reason. PiCode does not merge by
itself in this mode and does not run the project's checks; the provider's CI
does, on the merge result, exactly as a merge queue is meant to. The engine is
**declared, never detected**: the integration declaration names its mode
(`provider` or `local`) and follows the workspace → machine fallback ADR-0182
established, and a project that declares nothing is not executed at all — the
existing "the project declares no integration rules" blocker stands, because
absence is a named blocker and never a guess. `local` is the fallback for a
repository with no hosting provider: the runner landed in D3, bounded to the
declared single-line commands and a fast-forward-only move, becomes a *mode*
rather than the engine, and the `order` action belongs to it — under `provider`
the provider owns the order and PiCode refuses `order` with that reason. The
surface vocabulary is the community's: **merge queue**, entry, position,
*approved*, *integrating*, *integrated*, and **ejected** with a reason; internal
store field names may keep their own words. Provider state read by PiCode is
evidence with the provider named as its source, receipts stay the product's
format, and a provider that cannot be reached is unknown — never inferred.

## Consequences

Easier: PiCode stops being a shadow CI with its own executor to maintain; the
queue becomes legible to anyone who knows GitHub or bors; a project that already
governs integration keeps governing it; and the local mode still serves the plain
git repository the queue was originally for. Harder: a provider credential or CLI
enters PiCode's world and its authority must be **the owner's own** authentication
— never an agent's, and never a stored credential the product manages; the daemon
must be able to report "the provider refused" without inventing state; eligibility
now depends on state readable only from the provider, so an outage is a named
blocker rather than a stale answer; and two modes must be kept honest and tested,
with the provider mode needing fixtures against real vendor behavior. If we are
wrong about the provider path, the cost is a stalled queue that names its blocker
— loud and recoverable, not a wrong merge. If we retired the local mode too
early, a repository without a provider would lose integration entirely; that is
why it stays as a declared mode rather than being deleted.

## Alternatives considered

- **Keep the local runner as the engine.** This is what ADR-0182 said, and it
  contradicts the study's "reuse provider governance" row and the plan's
  non-goals; it also puts PiCode in the CI business, where a project's own
  provider is better informed. Rejected, though the code it produced is what
  becomes the fallback mode.
- **Provider only.** Aligns perfectly with the study and breaks the case the
  queue exists for — a plain git repository with `make ci` and no hosted queue.
  Rejected.
- **Detect the provider automatically instead of declaring the mode.** A guess
  about authority and about which queue is authoritative; ADR-0182's own rule —
  absence is a named blocker, not a guess — is better. Rejected.
- **Copy the merge-group CI configuration into PiCode.** The study's GitHub row
  already names the limit: provider eligibility and merge-group configuration
  constrain direct reuse. Rejected.
- **Rename everything, change no engine.** Cheaper, and it would leave the
  product with a second queue mechanism beside the provider's — the owner's
  review named both the semantics and the engine. Rejected.
