# llama.cpp manager work plan

- Date: 2026-09-06
- Owner approved the four-delivery roadmap and starting implementation.
- Status: delivery 1 merged and deployed as `ac4ff1dd` (`0.1.0+ac4ff1d`).
  Deliveries 2–4 planned, not shipped.

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

Implementation and acceptance are pending. A job records the normalized server
identity, model, operation, request key, observed state and timestamps; secrets
stay outside the job. Replacing other models reserves the whole server while
ordinary operations reserve their model. A connection edit cannot retarget an
existing job. Store transitions append feed events in the same transaction.

| Conditions | Required action | Acceptance |
|---|---|---|
| Same request key and identical operation | Return existing job; never repeat POST | Pending |
| Same key with different input | Conflict; no remote mutation | Pending |
| Active operation on the same model | Conflict until reconciled | Pending |
| Replace requested while any model job is active | Conflict before unloading anything | Pending |
| SSE responds with the expected content type | Observe events, reconcile final catalog | Pending |
| SSE unavailable or disconnected | Bounded server-side catalog polling; preserve progress | Pending |
| Progress has multiple files | Keep done/total per file; unknown totals remain indeterminate | Pending |
| Browser leaves or reconnects | Read durable job and feed; never replay mutation | Pending |
| PiCode restarts after accepting a job | Reconcile without replay; ambiguous result is interrupted/unknown | Pending |
| Server URL or credential changes during work | Keep the original target; no silent retargeting | Pending |
| Download cancel supported and download still active | Request unload; confirm observed terminal state | Pending |
| Cancel races with successful completion | Reconcile; never label completed download as canceled | Pending |
| Cancel unsupported, load/unload active, or remote state uncertain | Explain limitation; no false cancellation success | Pending |
| Server unreachable after accepting a mutation | Unknown outcome; retain evidence and reconcile | Pending |

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
  `TestWaitOutcomes`, `TestLlamaOperationFailures` and
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
