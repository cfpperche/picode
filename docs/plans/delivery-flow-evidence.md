# Delivery flow D0 — evidence inventory

Inspected 2026-09-21 against main `85f46a0a6eda617c1a8e309003a83bb1be505741`.
This is a source-code and read-only local inspection, not implementation or
visual acceptance. The working repository continued changing during measurement.
Design: [delivery-flow-design.md](delivery-flow-design.md).

## Source to state

| Fact | Existing source and ownership | Freshness / failure limit | D1/D2 treatment |
|---|---|---|---|
| Repository identity | `internal/gitgraph/gitgraph.go`: `Key`, common Git directory; owner-scoped `/api/{workspaces,agents,terminals}/{id}/git` in `internal/server/gitgraph.go` | Owner cwd may move; helpers can flatten subprocess failures to empty values | Preserve owner/root preconditions; explicit errors in new reader; no arbitrary path parameter |
| Branch and worktree inventory | `gitgraph.ListWorktrees`, graph refs/worktrees | Graph dirty scans capped at 32; gitstatus sibling scan capped at 8 and only returns dirty siblings; history is paged | Enumerate refs and worktrees, not only gitstatus or loaded history; show partial coverage and removed checkouts |
| Integration into a target | Local Git ancestry | `Ref.Merged` is relative to reader HEAD; ahead/behind in refs/status is upstream-relative, not necessarily main | Resolve explicit target OID; compute ancestry against that OID; do not reuse upstream distance as land readiness |
| Changed files | `internal/server/gitstatus.go`, `StatusWithStats`; existing diff/blob routes | File lists may be confined to subfolder; missing root can become `git:false`; a dirty worktree does not prove completed work | Keep scope visible; changed-file links use existing owner/worktree confinement |
| Agent association | `occupantsByWorktree` in `internal/server/gitgraph.go`; store agent cwd, pi terminal launch cwd | Current occupancy only; not authorship or historic session attribution; direct terminal scan is Pi-specific | Label Associated agents; never Owner/Author from cwd. Keep session link null unless explicitly recorded |
| Scoped validation | `scripts/ci-scoped.sh`, per-worktree git-dir `picode-ci-scoped.json` | Successful run only: base/tree/at/count/covered/roots; no failed or running attempts. File disappears with worktree metadata | Read as historical scoped evidence; do not claim full CI or completed review |
| Reusable validation | `scripts/ci-scope-reuse.mjs`, ADR-0124 | Identical tree or matching covered blobs on a clean checkout. Assumes green main for covered-only reuse | Share/port the pure decision with parity fixtures; show reuse basis and missing main evidence |
| Worktree summary | `scripts/worktree-status.mjs` | `greenNow` only compares tree, so can say stale where close allows covered-content reuse | Discovery aid only; not the delivery status authority |
| Full main CI | `scripts/land.mjs`, `scripts/ci.sh`, `var/ci-last.log` | Land fast-forwards first, then CI. Log is overwritten, has no durable structured run identity/result bound to revision | Integrated is Git fact; validation remains unknown without a matched receipt. New receipts needed for complete D1 |
| Git updates | `internal/server/git_watch.go`, started every 3 seconds in `cmd/picode/main.go` | Ephemeral `git.updated`, key is branch/worktree/dirty count, not HEAD; 8 siblings; relist each 10 ticks; plain terminals not watched | Event invalidation is useful but cannot cover clean commits, gate files or every checkout; bounded visible refresh justified |
| Stronger change token | owner-scoped `/git/head`, `gitgraph.Token` | Includes HEAD, refs, worktrees and own dirty count; not gate receipts or file contents at same dirty count | Useful Git invalidation only; no readiness conclusion from unchanged token |
| Deployment activity guard | `/api/deploy/readiness`, `internal/server/deploy_readiness.go` | Global daemon busy owners, not a project queue. List failures are skipped; ready=true does not prove every probe succeeded | Show reported busy owners; never translate ready=true into Safe to deploy. New observer needs error coverage |
| Last deployment record | `internal/install/install_unix.go`, data `var/deploy-log.jsonl` | Best-effort append after systemctl restart: at/version/termId/cwd. No started/failed attempts, requested SHA or health proof; cwd is not a repository binding | Historical receipt only; no complete last-attempt or queue-wait claim |
| Running version | `/api/version`, `internal/version/version.go` | Source builds expose seven-character revision; stamped releases/no VCS can expose semver only; dirty-build flag ignored | Unique local OID resolution or unknown; do not equate version with pristine contents or map to unrelated project |
| Server response | `/api/health`, `handleHealth` | Liveness and bootId only, not application correctness; separate version request may cross a restart | Bound observation with health/version/health; retry once if bootId changes, otherwise unknown |
| Serialization | Makefile `MUTATION_LOCK`, deploy CLI lock | Prevents concurrent restarts; contains no durable queue position or authoritative named holder | Do not invent queue position or waiting duration from a lock file |
| Feed | `internal/feed`, `web/shared/client/feed.js`, architecture/change-feed.md | Store events durable/replayed; ephemeral observations not replayed as history | Reconcile on open/reset/reconnect; configuration mutations go through store+event |

## Existing surface inspection

These are code-path inspections, not screenshot verdicts.

- Browser/desktop content: `web/browser/src/components/GitGraphSurface.jsx`
  owns the repository canvas. First load has skeleton rows; first-load error has
  Try again; refresh keeps last graph with warning; no commits has a message and
  Refresh. The workspace picker selects the read owner, while `g:<key>` identifies
  the repository. `web/desktop` is shell chrome, not the place to implement this.
- Mobile: `web/mobile/src/screens/Git.jsx` has Changes, History, Pull request;
  `GitReadState.jsx` supplies skeleton and Retry. A moved root offers Follow
  folder; a non-repository says so with Back. Detail screens are pushed, with
  the selected project/root retained. This is the entry point for Delivery.
- Inspector `Changes` is session/file-oriented. Its current association is useful
  for links but does not make it a project delivery queue. No additional
  Inspector tab is proposed.

## Measured baseline: one read-only lookup sequence

At 2026-09-21 17:07:57 UTC, seven separate source lookups were needed to assemble
branch, validation and running-instance facts. Executed sequentially from the
root; raw sanitized output is local-only `var/delivery-d0-baseline.json` in the
D0 worktree. The table below preserves the measurements after cleanup.

| Lookup | Method | Elapsed |
|---|---|---:|
| Worktrees | `git worktree list --porcelain` | 3.6 ms |
| Branch/gate overview | `node scripts/worktree-status.mjs` | 460.5 ms |
| Main CI evidence availability | stat `var/ci-last.log` (not a result assertion) | 0.1 ms |
| Deploy receipt | read latest JSONL row, return only time/version | 0.7 ms |
| Liveness | GET local `/api/health` | 10.2 ms |
| Running identity | GET local `/api/version` | 6.7 ms |
| Busy guard | GET local `/api/deploy/readiness` | 127.5 ms |

Total source-query time: 609.3 ms, one sample, excluding human interpretation.
This is neither a user-task timing nor a performance target. The local HTTPS
probe used an unverified TLS context for localhost only; this is not a proposed
product transport policy. No authentication material or message content was read.
The running instance reported `0.3.1+85f46a0`, health ok and six busy entries.
The last successful receipt had the same version at 16:56:06 UTC. These values
are historical observations, not current status or proof of application health.

The initial main read was `85f46a0a`; the later worktree overview already reflected
another session's branch being included. This is not an atomic project snapshot.
D1 must compare target/source OIDs before and after its collection and expose
unstable observations rather than combine them into eligibility.

Actual time spent waiting to land/deploy cannot be measured from these sources:
there is no durable ready/enqueued/requested timestamp. Commit age and time since
deploy are not waiting time. Record that baseline as unavailable, then collect
actual owner-task timing during the observation pilot and operation waits after
D3/D4. No historical wait estimate is invented to satisfy D0.

## Gaps that determine implementation

1. Error-aware, target-specific snapshot with honest partial coverage.
2. Versioned validation/land/deploy receipts, independent of surviving worktrees.
3. Explicit project-to-running-instance association and full build identity.
4. Reliable refresh for clean commits, evidence changes and lost events.
5. Human-reviewed readiness and queue intent have no source today; neither is
   inferred from agent idle, clean tree, commit message or gate success.

The proposed boundary is [ADR-0170](../decisions/0170-delivery-observation-contract.md).
It is not accepted by this inventory. External providers, semantic application
health and historical authorship remain explicitly unsupported in the pilot.
