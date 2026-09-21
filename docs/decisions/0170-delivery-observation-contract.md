# ADR-0170: Observe delivery through revision-bound evidence

- **Status**: proposed — owner decision required before implementation
- **Date**: 2026-09-21
- **Boundary**: protocol and persistence — owner-scoped delivery observations,
  durable producer receipts and explicit local environment association; process —
  records execution facts without granting new execution authority
- **Related**: ADR-0022/0073 (repository identity and reads), ADR-0048 (events),
  ADR-0072/0095 (independent surfaces), ADR-0085 (deploy forensics),
  ADR-0105/0124 (owner deploy and validation reuse)

## Context

The [D0 inventory](../plans/delivery-flow-evidence.md) found usable Git and
runtime observations, but no complete delivery record. Scoped stamps retain only
success and vanish with worktree metadata; full CI overwrites a text log; deploy
records carry neither failed attempts nor full repository/artifact identity.
Current occupancy does not prove authorship. A queue-like UI built from these
alone would invent readiness, health or wait times.

The owner approved D0 research and design, not this boundary implementation.
The [surface contract](../plans/delivery-flow-design.md) separates Integration and
Deployment, with explicit unknown states and read-only actions in D1/D2.

## Decision proposed

Add a versioned, authenticated owner-scoped delivery read contract. Resolve the
repository through the existing owner and expected-root check. Resolve explicit
source/target OIDs, report completeness and source timestamps, and distinguish
historical evidence from current facts. Retain the existing Git routes unchanged.
No path-valued repository lookup or repository-script execution is introduced.

For the PiCode pilot, existing command producers write versioned observation
receipts, not queue requests. Scoped checks, full CI and land receipts live under
the repository common Git directory in `picode-delivery/`; deployment receipts
live under the configured PiCode data directory in `var/delivery/`. One uniquely
identified operation gets an atomic JSON record, initially started and later
replaced with its result. Readers tolerate missing/malformed/version-unknown
records. Use owner-only directory/file permissions; bound file sizes and refuse
symlink escape from the managed evidence directories. These files are data, never
commands. Git working files and agent messages cannot directly assert delivery.

Receipt envelope: schema version, operation ID, kind, repository key, actor when
known, start/end time, source and target-before/target-after full OIDs, outcome
(started/passed/failed/unknown), coverage/fingerprint where relevant and a bounded
error summary. Deployment additionally records requested revision, built revision,
artifact cleanliness and observed revision/boot identity when available. A start
without a finish after restart becomes outcome unknown; it is not retried by this
observer. Wall-clock anomalies invalidate durations. Attempts started before
capture are not reconstructed from commit timestamps.

Preserve the existing scoped-stamp reuse rule and deploy log; receipts supplement
them. Receipt-writing failure warns but never turns a failing command into success,
changes its exit code, or prevents owner-authorized recovery. No automatic retention
or deletion is added in D1/D2; paginate reads, report coverage limits, and measure
storage before proposing a retention policy. Removing a worktree does not remove
its common-directory receipt. Removing the repository/data directory still does.

Configure the D2 local observer explicitly on a registered workspace through the
store and a same-transaction event (with mutation-table coverage). Bind it to the
resolved repository key; a folder move or repository mismatch disconnects it.
Other owners may consume this repository's configured observation only after the
same repository match. Conflicting workspace bindings report a configuration
conflict, never pick an arbitrary environment. Initial observer vocabulary is
none or `picode-self`; no remote URL, credential, arbitrary command or network
probe supplied by repository files is accepted. A build/deploy receipt must bind
the running artifact to that repository before reporting included changes.

Expose full running revision and artifact provenance separately from display
semver, preserving existing version fields. Sample health/version/health under a
stable boot ID; health means responding only. Unknown provenance, shortened hash
ambiguity, dirty build or missing history cannot become verified publication.
The server can read its own identity directly; client reconnect must still
invalidate observations across boots. Never expose deploy-log cwd, raw logs,
credentials or terminal content through the summary.

Keep configuration in the store, command facts in producer-owned receipt files,
and derived observations in memory. No writes bypass the store for store-owned
state. Bounded visible refresh is allowed because existing `git.updated` events
omit clean commits and receipt changes; thresholds, timeout and coverage are in
the surface contract. Store configuration emits durable events; external receipt
reads are observations, not synthetic durable operation-history events.

## Unchanged authority

This proposal does not approve a change, order a queue, merge a branch or deploy.
It records current `make close`, `make land`, `make ci` and owner deploy behavior,
including failed CI after a successful fast-forward. ADR-0105's owner decision,
active-turn guard and shared mutation lock stay intact. D3/D4 require their own
approval and execution/recovery boundaries. Existing logs remain compatible.

## Consequences

The UI can show evidence tied to an exact change and retain it after worktree
cleanup. Additional producer instrumentation, a versioned parser, configuration
validation and parity tests become maintenance costs. Local evidence is not a
cryptographic audit trail: anyone able to modify the repository or data directory
can modify its receipts. It therefore grants no authorization by itself.

Missing receipts degrade visibility, not command safety. The pilot cannot infer
unrecorded failures, queue waits, historic authorship, semantic squash inclusion,
external-provider deployment, or application-level health. Legacy history remains
clearly labeled rather than backfilled with fabricated facts.

If wrong, disable the Delivery observer and stop emitting new receipts; existing
Git, command, deploy and scoped-stamp behavior continues unchanged. No database
migration deletes existing state, and no observer automatically retries work.

## Alternatives considered

- Parse terminal prose or ci-last.log: loses identity, overwrites history and
  cannot distinguish failed publication from a still-running previous version.
- Derive Ready from idle/clean/passed: none proves human review or queue intent.
- Send every producer event to the live daemon: the daemon is restarting during
  its own deploy; recording must work while it is absent. A future importer can
  be proposed if consumers need durable push history.
- Adopt GitHub-only queues now: excludes local fast-forward workflows and does
  not observe the running PiCode instance.
- Add a generic deployment URL or command field: expands security and execution
  scope before the local evidence model is validated.

## Approval and validation

Owner decision requested: accept the read protocol, producer-receipt persistence
and explicit local-observer association described above for D1/D2. Status stays
proposed until that decision is recorded; D0 completion does not imply acceptance.

Implementation must cover the surface contract's O01–O26 rows, receipt atomicity,
malformed/untrusted files, symlink confinement, permissions, duplicate IDs,
write-failure exit-code preservation, version compatibility, moved bindings and
stable-boot sampling. No test or UI verification is claimed in this ADR draft.
