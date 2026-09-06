# llama.cpp manager work plan

- Date: 2026-09-06
- Owner approved the four-delivery roadmap and starting implementation.
- Status: delivery 1 merged and deployed as `ac4ff1dd` (`0.1.0+ac4ff1d`).
  Delivery 2 is implemented and visually validated in `feat/llama-jobs`; integration
  and deployment remain pending. Deliveries 3 and 4 remain planned.
  Deliveries 3–4 remain planned.

## Outcome and navigation

A dedicated llama.cpp surface replaces the embedded Providers manager.
Desktop opens a dedicated page with section tabs; mobile opens a full screen. Providers
keeps a Manage entry. `/llama` and old links reach the new surface.
Canonical routes: `#/llama/models`, `#/llama/catalog`, `#/llama/server`,
`#/llama/activity`. Only implemented sections are exposed. The base route
shows Models, with a clear connection action when unavailable.

Adapt Cursor's explicit provider verification (the existing
[Providers benchmark](../benchmarks/2026-09-03-providers-view-v2.md)) into
Test connection and actionable results. Adapt the independent settings
surfaces in the [Cursor/t3code/Paseo study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
into a dedicated, compact manager rather than a long Providers footer.

## Delivery 1 — Page and reliable connection

- Dedicated desktop/mobile surface, old-link compatibility and Providers entry.
- Server URL and credential editing; distinguish configured from reachable.
- Connection errors distinguish authentication, timeout, unreachable and
  unsupported router responses; preserve useful errors without exposing secrets.
- Explicit cancellation before load; unload errors and completion timeouts
  must not be reported as success; failed downloads must not become successes.
- Existing search/download/load/unload remain accessible.
- Validate routes, credential preservation, operation outcomes, and screenshots
  of empty, blocked, error and dialog states on both apps.

## Delivery 2 — Model operations

- Detect server capabilities and supported build; use `/models/sse` where
  supported, with bounded server-side polling fallback and reconciliation.
- Jobs with IDs, durable Store transitions and feed events; per-model operation
  exclusion, reconnect recovery and explicit interrupted/unknown outcomes.
- Preserve download progress per file; show real progress or indeterminate
  stages. Support cancellation only when the connected server supports it.
- Activity history survives navigation; no replay of mutations on reconnect.
- Acceptance: navigate away/reconnect during each operation and verify its
  actual final state, concurrency and cancellation behavior.

### Delivery 2 execution matrix

Delivery 2 implements the following contract. A job records the normalized server
identity, model, operation, request key, observed state and timestamps; secrets
stay outside the job. Replacing other models reserves the whole server while
ordinary operations reserve their model. A connection edit cannot retarget an
existing job. Store transitions append feed events in the same transaction.

| Conditions | Required action | Acceptance |
|---|---|---|
| Same request key and identical operation | Return existing job; never repeat POST | `TestLlamaReservationsAndIdentity`, `TestLoadDedupAndFallback` |
| Same key with different input | Conflict; no remote mutation | `TestLlamaReservationsAndIdentity`, `TestLlamaJobHTTP` |
| Active operation on the same model | Conflict until reconciled | `TestLlamaConcurrentReservation`, `TestLlamaReservationsAndIdentity` |
| Replace requested while any model job is active | Conflict before unloading anything | `TestLlamaReservationsAndIdentity` |
| SSE responds with the expected content type | Observe events, reconcile final catalog | `TestCapabilityEvidence`, real `TestLiveLlamaJobs` |
| SSE unavailable or disconnected | Bounded server-side catalog polling; preserve progress | `TestLoadDedupAndFallback`, `TestEventsAndContextCancellation` |
| Progress has multiple files | Keep done/total per file; unknown totals remain indeterminate | `TestProgressRedactsURLsAndPreservesUnknownTotal`, real download + UI matrix |
| Browser leaves or reconnects | Read durable job and feed; never replay mutation | `qa-llama-jobs.mjs`, real `qa-llama-jobs-live.mjs` |
| PiCode restarts after accepting a job | Reconcile without replay; ambiguous result is interrupted/unknown | `TestRestartReconcilesWithoutReplay`, `TestTimeoutAndQueuedRecovery`, real restart |
| Server URL or credential changes during work | Keep the original target; no silent retargeting | `TestRestartChangedConnectionAndCancellationNeverReplays` |
| Download cancel supported and download still active | Request unload; confirm observed terminal state | `TestDownloadCancelAndCompletionRace`, real cancellation |
| Cancel races with successful completion | Reconcile; never label completed download as canceled | `TestDownloadCancelAndCompletionRace` |
| Cancel unsupported, load/unload active, or remote state uncertain | Explain limitation; no false cancellation success | `TestCancelUnsupportedAndFailedDownload`, `TestRestartChangedConnectionAndCancellationNeverReplays` |
| Server unreachable after accepting a mutation | Unknown outcome; retain evidence and reconcile | `TestTimeoutAndQueuedRecovery` |

Use `scripts/qa-llama-runtime.py` against an explicitly selected disposable,
unloaded model for real router load, streaming, tool-protocol and unload checks.
The optional `--pi` executable adds an isolated Pi read-tool round trip. It does
not replace the job matrix or certify model quality, GPU performance, or the
browser workflow. It changes neither production Pi configuration nor services.

## Delivery 3 — Choose and use a model

- Catalog: model source, size, quantization, license/gated access and search states.
- Replace unconditional Q4_K_M recommendation with contextual guidance.
- Distinguish file size, estimated memory and measured memory. Missing data
  stays unknown; context and execution settings affect estimates.
- Explicit short streaming and tool-call tests before recommending agent use.
- Use in agent action and affected-agent visibility; configuration references
  are not evidence that an inference is currently in flight.
- Acceptance: download → load → test → use in a real Pi agent, including a
  failed tool-call test. Other coding CLIs remain terminal integrations.

### Delivery 3 acceptance record

The pinned b10809 CPU fixture passed router discovery, SSE, load/unload,
streaming, forced tool protocol and a real Pi 0.85.1 `read` round trip with
Qwen3-4B-Q4_K_M. The earlier Qwen3-0.6B fixture intentionally failed the
agent read round trip while still passing protocol checks; it remains recorded
as a negative compatibility example. Both runs used four CPU threads and
8192-token context, with disposable servers stopped afterwards. This does not
certify GPU performance or general coding quality.

## Delivery 4 — Execution settings and managed service

Included in the approved roadmap. Before implementation, record the concrete
ownership and lifecycle design in a separate ADR, including how it changes
the existing connect-only/no-file-deletion guide. The roadmap authorizes design
work; it does not make an unspecified destructive operation safe to execute.

- Execution presets: context, GPU offload and supported runtime flags, with
  effective configuration preview and visible restart requirements.
- Installation and start/stop/restart for an explicitly PiCode-owned service;
  inspect externally managed services without taking ownership implicitly.
- Version-pinned updates with source verification, recovery and rollback.
- Model cache cleanup: exact file ownership, references, affected agents,
  reclaimable-size evidence and explicit selection/confirmation/revalidation.
- Linux/WSL first; identify the server host separately from the browser host.
  Remote servers require their own management mechanism, not local process probes.
- Export redacted diagnostics. No arbitrary shell command field or silent restart.
- Acceptance: disposable-service installation, failed update rollback, restart
  with active consumers, cleanup refusal for referenced/unowned files and
  recovery after interruption. Record untested platforms as debt.

## Decision table and verification

| Conditions | Action/outcome | Required coverage |
|---|---|---|
| Connection succeeds, no models | Empty list + Download action | API fixture + desktop/mobile screenshot |
| HTTP 401/403 | Authentication result + edit connection | Client test + error screenshot |
| Timeout / transport failure | Distinct result + Retry | Client test + blocked screenshot |
| Non-router/malformed response | Unsupported result + setup link | Client test |
| Load with other models, user cancels | No mutation | `scripts/qa-llama.mjs` |
| Load with other models, keep selected | Load without unloading | `scripts/qa-llama.mjs` |
| Load with other models, unload selected | Explicit unloadOthers request | `scripts/qa-llama.mjs` |
| Unload-others fails | Stop load, report failure | Handler test |
| Unload completion fails | Report failure, never success | Client/handler test |
| Download reaches failed state | Report failure, never success | Client test |
| URL edited without changing secret | Existing credential preserved | Provider contract test |
| Job reconnect/cancel/concurrency | Reconcile actual state | Delivery 2 matrix |
| Model tooling incompatible | Do not label agent-ready | Delivery 3 matrix |
| Managed/external ownership or referenced cache | Mutate only reviewed eligible target | Delivery 4 matrix |

Every delivery runs make ci and visual review for UI changes, updates
architecture/public guide/changelog/handoff and records remaining coverage.
No delivery is called shipped before its required acceptance passes.

## Research sources

Reviewed 2026-09-05; upstream capabilities require installed-version checks:

- [llama-server API](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md):
  router, SSE, load/unload, download and cache management.
- [Tool calling](https://github.com/ggml-org/llama.cpp/blob/master/docs/function-calling.md):
  model/template compatibility must be tested.
- [LM Studio model loading](https://lmstudio.ai/docs/cli/local-models/load):
  context, GPU, TTL and memory-estimation UX reference, not llama.cpp API parity.

## Delivery 1 validation record

- make ci passed: formatting, vet, Go tests, 893 JS/package tests, both apps,
  embedded binary, generated-doc parity/build and Vale. Public captures were
  regenerated through docs-shots; docs gates passed again after guide edits.
- `TestConnectionResults`, `TestConnectionTimeoutAndTransport`,
  `TestWaitOutcomes`, `TestReplaceFailureNeverLoads` and
  `TestPutLlamaPreservesKeyOnURLChange` cover the backend matrix.
- `scripts/qa-llama.mjs`: desktop/mobile legacy routes, empty catalog,
  download dialog/search-empty, blocked/auth error, invalid/saved connection,
  cancel with no mutation, keep/unload others, unload and Providers entry.
- 16 screenshots and overlay audits: `docs/screenshots/llama-qa.json`.
  Captures read; visual-review PASS. Tested against a disposable fixture and
  synthetic router responses, not an installed GPU/model runtime.
- Delivery 1 intentionally retains synchronous operations. Installed-version
  acceptance, job recovery/concurrency and model readiness remain subsequent
  delivery work; no real service installation, restart or deletion occurred.

Integration validation (2026-09-06): combined make ci passed with 902
JS/package tests; the 16-image browser matrix passed again. Public captures
were regenerated and read. Owner authorized merge and deployment.

Deployment (2026-09-06): `ac4ff1dd` published with serialized make deploy;
health ok, deployed desktop bundle verified, 9/9 pre-deploy terminal IDs
preserved. Live desktop/mobile legacy links, Models/Server pages and Test
connection passed; screenshots read and audits ok. The pre-existing llama
endpoint timeout remains, so real-model acceptance is still pending.

## Real runtime investigation (2026-09-06)

No listener on WSL port 8080 and no matching Windows llama.cpp/Ollama/LM
Studio process were found. Windows networking is mirrored. No standalone
llama-server or completed GGUF was found in the searched user folders;
`~/ollama` exists, with only partial model blobs. This does not identify the
network component dropping connections to the unused production endpoint.

A disposable upstream Linux CPU router was run on loopback ports 18080/18081,
without changing production configuration. Build `b10809-5266f24da` was
obtained from the official v0.4.0 release's nightly pointer. SHA-256 verified:

- Server archive: `5e34434ddc6d03cd1584f403201aff0d4bd1a5793a72ff7e286532dfd1e4b941`.
- Qwen3-0.6B-Q8_0 GGUF: `361cc68159042c36ebff7715dc5a2e4612153e88f3e9c9c234820849d6dc9e1d`.
- Model repository revision: `b5f37287796e5be0ea3dab2e7430873fb3f73e49`.

The runtime test passed router detection, SSE subscription and load events,
load/unload, four streaming content chunks and a forced function call with
validated arguments. The isolated real Pi read-tool round trip **FAILED**:
Pi exited zero but the model invented file content without executing `read`.
The test correctly returns failure despite that exit code. This model is not
certified for coding-agent use. The nonexistent-model refusal also passed
without a mutation. Evidence: [runtime report](llama-runtime-qa.json).

`make ci` passed (902 JS/package tests and Go/build/docs gates). No application
code or UI changed. Delivery 2 remains unimplemented; the matrix above is the
next implementation contract, not acceptance evidence. Real router download,
cancellation, GPU execution, durable jobs and a successful Pi model remain
pending. The disposable servers were stopped; downloaded test artifacts remain
under `/tmp/picode-llama-runtime`, outside the repository and production cache.

Pinned protocol reference:
[llama-server b10809 API](https://github.com/ggml-org/llama.cpp/blob/5266f24da/tools/server/README.md).

## Larger-model validation (2026-09-06)

Qwen3-4B-Q4_K_M from the official Qwen repository passed the same runtime
script on build b10809, CPU-only, four threads, 8192-token context and thinking
disabled. The 2,497,280,256-byte model was pinned to revision
`bc640142c66e1fdd12af0bd68f40445458f3869b`; its SHA-256 matched the upstream
LFS metadata. See [the complete report](llama-runtime-4b-qa.json).

Router detection, SSE subscription/load events, model load/unload, streaming,
forced function arguments and the real Pi 0.85.1 read-tool round trip all
passed. Pi emitted tool execution start/end events and returned the marker
from the temporary file. The 0.6B failure remains recorded for comparison.
This is one basic integration test, not a coding benchmark or a reliability
rate. No GPU, production browser, download/cancel or durable-job acceptance
is implied. The server was stopped after unloading; production configuration
was not changed. The verified model remains in the isolated temporary cache.

`make ci` passed again (902 JS/package tests plus Go/build/docs gates).
Delivery 2 can now use this model/build pair as a real acceptance fixture;
job implementation and its decision matrix remain outstanding.

## Delivery 2 validation and limits

Validation uses synthetic condition matrices and the real b10809 router with
an isolated cache. Qwen3-4B runs on CPU with four threads and 8192 context;
models are loaded sequentially and unloaded after the test. The real download
cancellation uses the smaller Qwen3-0.6B Q4_0 artifact and stops after progress
is observed. No benchmark, GPU load or production service change is required.

The real catalog did not expose download byte counts in the acceptance build;
`download_progress` SSE events do, nested under `progress` in b10809. Both
the nested and documented direct payloads are accepted. The implementation merges these per-file
observations, strips URL credentials/query strings and persists them. When SSE
is unavailable, catalog polling reconciles state and retains the last known
bytes; no new byte counts or percentages are invented. SSE reconnect retries
are bounded by the job's 30-minute watch deadline.

Ordinary CI skips only the opt-in `TestLiveLlamaJobs`, whose router/model/cache
must be provided explicitly. Run it with `PICODE_LLAMA_LIVE=1 go test
./internal/llamajob -run '^TestLiveLlamaJobs$' -count=1 -v`; the router must be
isolated on `127.0.0.1:18081`, with Qwen3-4B-Q4_K_M initially unloaded. Set
`PICODE_LLAMA_REPORT` to save the result. Browser matrices use a separate
`picode-docs-fixture` at port 18763. `qa-llama-jobs-live.mjs` changes only that
fixture's connection, loads/unloads the real model and checks shared history.

Unknown download results can retain a reservation when the model is absent:
absence alone cannot prove that a failed/canceled download completed. No
forced unlock, automatic retry, service ownership or cache deletion is added.
Cancellation is verified only for b10809; other builds retain observation and
show the limitation. Older job records remain in SQLite; retention/pruning
beyond the displayed most-recent-50 history is not introduced.

The final full gate uses `GOMAXPROCS=4 GOFLAGS='-p=2 -count=1' make ci`.
Cached runs stalled in Go's `computeTestInputsID` / `EvalSymlinks` / `Lstat`
before another test process started; disabling result reuse avoids that local
cache-path lookup and runs the tests afresh. No application workaround is added.
