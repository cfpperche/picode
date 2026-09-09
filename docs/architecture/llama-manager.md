# llama.cpp manager (ADR-0080)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`#/llama/models` and `#/llama/server` own model operations and connection
settings on desktop and mobile. Providers links to this surface; legacy
`#/providers/llama` and the desktop `/llama` command reach it. URL and
optional key still live in Pi auth.json; an empty key preserves the saved key.
`GET /api/llama` adds a safe `connection` code/message distinguishing ready,
authentication, timeout, unreachable, unsupported and server errors.
Model operations now return HTTP 202 with a durable job (ADR-0083). SQLite
migration 030 stores request identity, observations, per-file progress and
revision-guarded transitions. `internal/llamajob` reserves models/endpoints,
coalesces router SSE with bounded polling fallback, and reconciles restart
outcomes without replaying mutations. Unknown results retain reservations.
Connection fingerprints stay outside public JSON; credentials are not stored
in jobs. Cancel is available for downloads on the verified b10809 build family.
`#/llama/activity` follows `llama.job` feed events and refreshes on reconnect;
its history survives navigation. Service ownership remains delivery 4 in
[the plan](../plans/llama-manager.md).
Hugging Face GGUF metadata also exposes file size and a conservative runtime
memory estimate with contextual guidance; missing size data remains unknown.
Delivery 4's ownership boundary is ADR-0090: only an explicitly created local
Linux/WSL profile may receive lifecycle or cache mutations; external and remote
routers remain inspection-only for those controls.
`internal/llamaservice` implements one explicitly created local CPU profile.
Migration 034 stores its configuration, release manifests, ownership ledger and
bounded job history through a CAS store mutation plus `llama.service` events.
The desktop/mobile `#/llama/service` page has Light/Balanced presets, advanced
settings, effective argv, reviewed lifecycle actions, cache and diagnostics.
It adapts Cursor's progressive disclosure and t3code's reload-safe routes from
the [benchmark study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md).

First creation stages empty folders with a random ownership marker and saves
an intent before atomically publishing the fixed service path. Linux
`RENAME_NOREPLACE` preserves even a preexisting empty directory; the existing
`golang.org/x/sys/unix` dependency supplies this operation. Failure or restart
removes only the recorded attempt's empty folders. Unexpected content is
retained for inspection; unreadable ownership proof keeps the recovery intent
until access is restored. An unrecorded staging folder is never adopted.

The release ledger retains manifests beyond the current/rollback pair.
Cache inventory includes older installations with verified exact contents;
current, rollback, unknown, changed and symlinked installations are protected.
Cleanup binds selection and bytes to a review, checks every target again,
and removes only manifest-listed files, never a recursive directory tree.
An interrupted or partially failed cleanup is not replayed; remaining files
are retained and a mismatched manifest requires inspection.

The installer embeds official b10809/b10826 Linux CPU archive hashes, verifies
before extraction, materializes internal library links as pinned regular
files, and checks version/required flags. It does not accept arbitrary URLs,
binary paths or shell fragments. It revalidates installed files before launch.
Reviews expire after five minutes and bind revision, targets and consumers;
model reservations share the lifecycle lock. Updates retain the old release
and recover it if a running replacement fails readiness. The private same-binary
supervisor owns a router process group: parent pipe EOF kills the entire group,
including after a daemon crash. Startup marks unfinished jobs interrupted and
does not relaunch the service. No process name or external PID is adopted.

The initial CPU profiles use explicit GPU zero, Jinja/autoload switches and
1–4 generation/batch threads. Model cache ownership is recorded only after a
tracked download on the owned router identifies a new exact file, pinned by
SHA-256. The verified b10809/b10826 HF cache catalog omits file paths: completed
download filenames and sizes are matched to a new snapshot link and its
content-addressed blob. The blob hash must match its name; existing blobs or
snapshot links are never adopted. Cleanup removes the recorded snapshot link
with its blob and refuses shared blobs, changed links or agent references to
any quantization of the same repository. Cleanup requires a stopped service,
no active model jobs and no configured agent references, with file revalidation.
Unknown, changed and referenced files stay intact. Diagnostics use an allowlist
instead of exporting logs or credentials. Linux x64 has real CPU acceptance;
ARM64 and GPU execution remain unverified. Filesystem checks assume the owner's
private data directory is not concurrently modified by another local process.
