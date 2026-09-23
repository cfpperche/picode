# ADR-0083: Durable llama.cpp operation jobs

- **Status**: accepted
- **Date**: 2026-09-06

## Context

The owner approved delivery 2 of the [llama manager plan](../plans/llama-manager.md).
Loading and downloading can outlive a browser connection. A lost response
cannot tell us whether the external router accepted a command. The validated
b10809 router supports events and download cancellation through unload.

## Decision

Load, unload and download return HTTP 202 with a durable job. A caller supplies
a request key; repeating identical input returns the same job. Different input
with the same key conflicts. Store writes and `llama.job` feed events share a
transaction. Revision checks protect cancellation against stale observations.

Jobs reserve their server/model, and replacement reserves the whole endpoint.
At most four unresolved jobs per endpoint can run. Unknown outcomes retain
their reservations. The endpoint and connection fingerprint are pinned at
acceptance; secrets remain in Pi credential storage or the running client,
never job payloads. URL userinfo is refused. Connection edits affect new jobs.

SSE invalidates observations, coalesced to at most two catalog reads per second.
A two-second server-side fallback covers missing/dropped events and observes
external model state, which PiCode's feed cannot originate. Watching is bounded
to 30 minutes; uncertain outcomes remain unknown with a Check result action.
There is no new browser API polling timer. Per-file byte counts persist;
unknown totals stay indeterminate. Historical API reads include all unresolved
jobs and the most recent 50 jobs; older records remain in SQLite.

Restart recovery never replays load, unload, download or cancellation. Queued
jobs become interrupted. Dispatched jobs reconcile the catalog against their
original connection; changed credentials require restoring that connection.
Interrupted replacement never resumes its load step. An observed unloaded
model after a restarted load is interrupted; missing downloads remain unknown.

Cancellation is enabled only for the verified b10809 build family and only
while a download is observed running. The cancellation dispatch is persisted
before POST. A completion racing cancellation wins as completed; canceled is
reported only after acknowledged cancellation and an absent/failed download.
Losing that acknowledgement remains unknown. Load/unload cancellation and
unverified builds show a visible limitation. Cancel never means file deletion.

Desktop/mobile own an Activity page at `#/llama/activity`. It follows the
existing feed, refreshes after reconnect/visibility changes, preserves history
across navigation and exposes progress, unknown outcomes and cancellation.
This adapts t3code's explicit waiting states and reload-safe routes from the
[benchmark study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md).

## Consequences

Navigation no longer owns the operation lifetime. A conservative unknown
outcome can keep a model reserved until the remote state is reconciled.
Builds other than b10809 can still run jobs but do not advertise cancellation
as verified. External tools can change the router outside PiCode's locks.
No service installation, ownership, restart or cache deletion is introduced.

## Amendment (2026-09-23): abandoning, and jobs on a stopped owned service

Owner-approved after an adversarial review found unknown jobs that could never
resolve. An **unknown** job can be **abandoned** by the owner: a new terminal
state, `abandoned`, recorded on the job with its time. It releases the model
reservation and the endpoint slot, and PiCode never touches the server for it.
A job on the endpoint of PiCode's own service becomes `interrupted` as soon as
that service's process is not running (stopped, or the daemon restarted),
because nothing can be in flight on a server that is not running. After a
restart, an unload whose model is still loaded becomes `interrupted`, as a load
already did. A download whose model is missing stays `unknown`, as before, and
can now be abandoned.

## Alternatives considered

- Keeping the HTTP request open loses durable identity and reconnect history.
- Retrying POST after reconnect can load or unload a model twice.
- Browser polling ties progress to a single page and duplicates feed delivery.
