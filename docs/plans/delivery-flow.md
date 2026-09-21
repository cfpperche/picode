# Delivery flow — execution baseline

Status: scope and sequence fixed for planning on 2026-09-21 at the owner's
request after the benchmark study. D0 design is complete; product implementation has not started. This records
an execution baseline, not authorization to change accepted ADRs or deploy.

Evidence: [delivery-governance study](../benchmarks/2026-09-21-delivery-governance.md).

## Outcome

A person supervising a project can identify what awaits review or integration,
why it is blocked, what has landed but is not deployed, and which revision is
actually serving each observed environment, without opening agent terminals.

## Fixed scope

- A project delivery view, centered on changes rather than agent activity.
- Links from each change to the relevant branch/revision and available sessions.
- Separate integration and publication lifecycles; many changes may share one
  deployment. Cancellation and abandonment are not successful delivery.
- Evidence, observation time, blocker and next action for each relevant state.
- Read-only observation first; explicit governance follows in separate slices.
- PiCode's own repository is the first measured workflow. The product model must
  allow other projects to use different integration and publication providers.
- Desktop and mobile share headless semantics; each retains its own presentation.

## Outside this baseline

No autonomous deploy, scheduled release train, automatic force/restart, automatic
conflict resolution, mandatory GitHub/PR workflow, or replacement CI/CD platform.
No generic shell-command executor, broad provider rollout, dependency-aware
scheduler or automatic rollback in the initial delivery. Existing provider links
may be exposed; deep remote integrations follow evidence from the pilot.

## Facts and state contract

These are required meanings, not a finalized database schema or endpoint design.

| Object | Required meaning |
|---|---|
| Change | Repository identity, source revision, target branch, optional PR and session links; attribution may be unknown. |
| Validation | Result, exact tested revision/base or relevant input fingerprint, check identity, time and evidence link. |
| Integration | Observed inclusion in target plus validation status; landed with failed or unknown CI is distinct from healthy. |
| Deployment | Environment, requested and observed revision, included changes when ancestry is known, operation result and health evidence. |
| Operation, governance slice only | Explicit intent, actor, target revision, lifecycle, blocker, evidence and terminal result. |

Agent status and delivery status are independent. A historical success does not
become current evidence after a revision changes. Git ancestry determines inclusion
where applicable; rebases/squashes or missing history require provider evidence or
an unknown mapping. Failure to read a source is unknown, not an empty queue.
An environment with no configured observer says unconfigured, not undeployed.

## Execution slices and exit criteria

Each implementation slice uses a new branch/session and the normal repository
rite. Check off a slice only with linked evidence. D0 receipts: [inventory](delivery-flow-evidence.md),
[surface contract](delivery-flow-design.md), and [proposed ADR-0170](../decisions/0170-delivery-observation-contract.md).

### D0 — Inventory and design contract

- [x] Trace existing Git/worktree readers, gate stamps, session attribution,
  deploy readiness, deploy records, version/health evidence and event feed.
- [x] Produce a source-to-state matrix with ownership, freshness, failure modes
  and unsupported cases. Seven source lookups were timed; actual queue wait has no
  recorded start and remains unavailable (see inventory).
- [x] Choose the host surface using existing Git/workspace conventions, inspect
  desktop/mobile empty/blocked/error states, and specify navigation before UI work.
- [x] Identify required protocol, persistence, security and process boundaries;
  draft applicable ADRs. ADR-0170 is proposed; obtain the owner decision before
  crossing its boundaries in D1/D2. D0 crosses none of them.

Exit: every displayed fact has a source and unknown behavior; an implementation
contract and decision table exist. No new execution authority is introduced.

### D1 — Observe integration

- [ ] Implement read-only project changes with revision, branch/base relation,
  known validation, attribution links and explicit blockers.
- [ ] Distinguish ready for review, validated candidate, stale validation,
  integrated awaiting validation, integrated healthy, and integrated failing.
- [ ] Refresh from existing events where available; justify any external Git
  observation/polling separately, with visible observation timestamps.

Exit: seeded concurrent-worktree cases and a scratch-instance walkthrough answer
what can be reviewed, what requires revalidation and what actually landed.
No claim that an observed candidate is enrolled in an execution queue.

### D2 — Observe publication

- [ ] Add per-environment current revision, last attempt, health observation and
  integrated-but-unpublished changes where evidence supports the comparison.
- [ ] Observe PiCode's existing readiness and deployment evidence for the pilot;
  expose unavailable, failed, interrupted and unconfigured cases honestly.
- [ ] Show a deployment containing multiple changes without inventing a separate
  deploy requirement for every branch. Preserve previous known evidence on errors.

Exit: distinguish main ahead of production, success with verified revision/health,
failure with previous production still live, and unknown current state. D1+D2 are
the first usable milestone; they require no new deployment action.

### D3 — Govern integration

- [ ] After the relevant owner-approved ADR, persist explicit queue intent and
  bind eligibility/authorization to the reviewed revision and applicable policy.
- [ ] Support enqueue, withdraw and explicit ordering, with one integration
  operation per repository and a named blocker for every waiting entry.
- [ ] Recheck base and evidence before execution. A changed revision invalidates
  prior eligibility; do not silently broaden an approved operation.
- [ ] Handle retries, duplicate requests, external merges and restart recovery
  without repeating an operation whose result is uncertain.

Exit: decision-table tests cover concurrent requests, base movement, stale
approval, partial failure and recovery. Existing land/gate semantics remain
visible, including a successful fast-forward followed by failing CI.

### D4 — Govern publication

- [ ] After owner approval of authority/security/process boundaries, provide an
  explicit publication request for a named environment and revision/change set.
- [ ] Integrate existing serialization and active-turn guards; show the current
  operation and the reason a request cannot run. Never bypass mutation locks.
- [ ] Reconcile actual deployed revision and health after reconnect/restart;
  requested revision and observed revision remain distinct if preparation changes
  the candidate. No silent substitution of a newer main.

Exit: owner-approved scratch scenarios cover contention, active agents, changed
candidate, failed build, lost connection, duplicate intent and health failure.
Force deploy and automated rollback stay outside this baseline.

### D5 — Pilot and generality decision

- [ ] Compare the same supervision tasks before and after D1+D2; evaluate again
  after any approved governance slices. Record terminal lookups, time to identify
  a blocker, integration/publication wait, and discrepancies in displayed state.
- [ ] Exercise a single-agent project, concurrent agents on one repository, and
  a project using external CI/deploy; simulated cases are labeled as such.
- [ ] Report which capabilities are common, project-specific or still unknown;
  owner selects follow-on providers and any scope changes from that evidence.

Exit: the owner can answer the outcome questions without inspecting transcripts;
all tested revision/health claims agree with their sources. Hypothetical workflows
alone do not establish demand across the user population.

## Required acceptance scenarios

| Conditions | Required observation/action | First slice |
|---|---|---|
| Agent ends turn; no validation evidence | Finished activity, delivery readiness unknown | D1 |
| Validation passed; source or relevant base changes | Mark evidence stale; require applicable revalidation | D1/D3 |
| Another branch lands before queued candidate | Recompute eligibility; expose waiting/revalidation | D3 |
| Fast-forward succeeds; main CI fails | Integrated, validation failed; never claim branch unmerged | D1/D3 |
| Worktrees disappear or attribution is missing | Reconcile from Git; preserve unknown attribution | D1 |
| Several changes integrated; production still older | Show unpublished set for that environment | D2 |
| No observer, unreachable source or divergent history | Unconfigured/unknown; no invented zero or delivery mapping | D2 |
| Deploy record succeeds; live revision/health unavailable | Historical success; current production unverified | D2 |
| Duplicate intent, lost response or process restart | Reconcile before retry; never infer safe repetition | D3/D4 |
| Deploy blocked by active agents or mutation lock | Waiting with reason; no force or parallel restart | D4 |
| Candidate changes after approval | Require renewed eligibility/authorization for changed scope | D3/D4 |
| Deploy fails while previous version remains healthy | Failed attempt and previous live revision shown separately | D2/D4 |

These are planned test obligations, not tests already run. Each implementation
must cover its rows or record an explicitly accepted gap in the owning open topic.

## Decision checkpoints and delivery discipline

The sequence and initial observation milestone are fixed. D0 resolves the exact
surface, evidence adapters and data/API design. Durable queue storage and events,
execution authority, authorization expiry and restart recovery require explicit
boundary decisions before D3/D4; this plan does not supersede ADR-0105 or other
accepted decisions. D1/D2 also need an ADR if their chosen design crosses a boundary.

Read the UI and visual-review skills for implementation. Validate desktop/mobile
states on scratch, read screenshots in a subagent, run the overlay audit where
applicable, and require the five-question visual card. Store mutations append feed
events. Run scoped gates and close per branch, then full CI on main; ship behavior,
architecture docs, public help, changelog and handoff together. Deployment always
remains a separately authorized owner operation.

Any expansion or reordering is recorded here with its reason and owner decision.
D0 is documented in the linked inventory and surface contract. The next task is
D1, after the owner decision on proposed ADR-0170, in a fresh session. D1–D5 are
not implemented; D0 changed documentation only and performed no deployment.
