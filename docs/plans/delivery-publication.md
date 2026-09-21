# D2 — publication observation (implementation contract)

Status: implemented in `feat/d2-deployment`, 2026-09-21 (owner: "pode comecar
d2"). D2 shipped; D3–D5 are not implemented and keep their own approval gates.
Parent: [execution baseline](delivery-flow.md) § D2.
Boundary: [ADR-0170](../decisions/0170-delivery-observation-contract.md)
(accepted for D1; D2 association, receipts and runtime provenance are this
slice). Surface contract: [delivery-flow-design.md](delivery-flow-design.md)
(O16–O21, O25–O26). No execution authority: no deploy button, no queue, no
retry, no rollback.

## What D2 adds

| Piece | Where | What it answers |
|---|---|---|
| Local observer binding | store (`delivery_observers`), per workspace | Is this project connected to the PiCode instance observing it? |
| Deployment receipts | producer writes `<data>/var/delivery/<uuid>.json` | What was deployed, from which revision, was it clean, what answered after |
| Runtime identity | `/api/version` (+`revision`), `/api/health` (`bootId` unchanged), discovery file (+`revision`, `boot`) | Which revision is actually serving, under which boot |
| Observation | `internal/delivery` (`deployment.go`) + the delivery read route (`environments[]`, per-change `publication`) | Which integrated changes are not published, and why the answer is unknown when it is |
| Deployment lane | browser + mobile Delivery view | The facts above, read-only, with the design's copy |

## Contracts (fixed before code)

### Deployment receipt — `<data>/var/delivery/<uuid>.json`, dir 0700, file 0600

```json
{ "schemaVersion": 1, "id": "<uuid>", "kind": "deploy",
  "repositoryKey": "/abs/path/.git", "actor": "term id or empty",
  "startedAt": "RFC3339Nano", "finishedAt": "RFC3339Nano (absent while started)",
  "outcome": "started|passed|failed",
  "requestedRevision": "<full oid>", "builtRevision": "<full oid>",
  "revisionBefore": "<full oid>", "observedRevision": "<full oid>",
  "bootBefore": "", "bootAfter": "", "clean": true, "error": "" }
```

Rules: `id` UUID and file name `id + ".json"`; `repositoryKey` non-empty;
`startedAt` parses and is not in the future; `finishedAt` required unless
`outcome == "started"`; every non-empty revision matches the 40/64-hex rule;
`clean` a real bool; `error` ≤ 512 bytes, one line; unknown/extra fields are
not accepted; a `started` record is observed as **unknown** (never running,
never failed). Writes are atomic (temp + rename) and best-effort: a receipt
that cannot be written never changes the deploy's exit code.

### Store — migration 065

```sql
CREATE TABLE delivery_observers (
  workspace_id TEXT PRIMARY KEY,
  repo TEXT NOT NULL,
  observer TEXT NOT NULL,           -- 'picode-self'
  updated_at TEXT NOT NULL
);
```

`SetDeliveryObserver(workspaceID, repo, observer)` upserts (`observer: "none"`
deletes) and appends `delivery.observer.changed` in the same transaction;
`GetDeliveryObserver`, `ListDeliveryObservers(repo)` read. A row is a row in
`TestEveryMutationAppendsAnEvent`.

### Read route — `GET /api/{workspaces|agents|terminals}/{id}/delivery`

Adds `environments[]` (one entry for the `picode-self` observer) and
`publication` per change:

```
environment: { id, kind, status: known|unknown|unconfigured|conflict,
  reasonCode, repositoryKey, displayVersion, revision, boot, startedAt,
  responding: true|false|null, artifact: clean|dirty|unknown, observedAt,
  unpublished: { count, changes: [change id] },
  lastAttempt: { id, outcome, at, requestedRevision, builtRevision,
                 revisionBefore, observedRevision, error } | null,
  busy: { status: known|unknown, owners } , coverage: { complete, issues } }
change.publication: published | not-published | unknown
```

The environment is resolved from the **viewed owner's own workspace binding**;
another workspace's binding is never substituted (no arbitrary pick). Four
outcomes: bound to this repository → observed; bound to a *different*
repository while another workspace claims this one → `conflict`
(+`binding-conflict`); bound to a different repository (the folder moved or the
project was replaced) → `unknown` (+`repository-mismatch`); no binding →
`unconfigured`.

### Config route — `POST /api/workspaces/{id}/delivery/observer`

Body `{"observer": "picode-self" | "none"}`. Resolves the workspace's
repository key (409 when the folder is not a repository), writes the binding,
answers the stored record. This is the environment selector's only write.

## Decision table (every row tested)

| # | Conditions | Observation/UI | Row |
|---|---|---|---|
| 1 | No binding for this project | `unconfigured`; "Deployment is not connected for this project." + View setup | O16 |
| 2 | Binding, running revision unknown or not in this repository, or artifact not clean | `unknown`; "Running version found; included changes are unconfirmed." | O17 |
| 3 | Bound, full revision present, target ahead of it | `known`; exact unpublished count + change list, responding | O18 |
| 4 | Boot changes between the two identity samples | retry once, then `unknown` (+`boot-changed`) | O19 |
| 5 | Newest attempt failed, an older revision answers | failed attempt and the live revision, separately | O20 |
| 6 | Newest attempt `started`, no finish | `unknown` outcome, not running, not failed | O21 |
| 7 | Busy guard errors or coverage missing | `busy.status: unknown`, no safety claim anywhere | O25 |
| 8 | Several changes integrated, one publication set | list changes, dedupe by source OID, no per-branch deploy claim | O26 |
| 9 | Receipt unreadable/malformed/foreign repository/symlink/oversized | rejected, `coverage.issues` names it, no green | ADR |
| 10 | No deployment receipt, running revision mapped | revision known, `lastAttempt: null` | O18 |
| 11 | The binding names another repository (folder moved, project replaced) | `unknown` (+`repository-mismatch`), the instance's identity still shown | ADR-0170 |
| 12 | This project is bound elsewhere while another workspace claims this repository | `conflict` (+`binding-conflict`); neither side is picked, and the binding is never substituted | ADR-0170 |

## Evidence

Automated: `internal/delivery` (`TestEnvironmentDecisionTable` — O16–O21 and
O25/O26, `TestReadDeploymentsConfinement`,
`TestReadDeploymentsIgnoresASymlinkedDirectory`,
`TestUnpublishedDeduplicatesOneCommit`,
`TestPublishedWhenTheRunningRevisionContainsTheChange`,
`TestArtifactStateNeedsAPassingRecordForTheRunningRevision`,
`TestLatestAttemptIsTheNewestForThisRepository`,
`TestBusyObservationNeverClaimsSafety`, `TestRevisionIdentitySurvivesAStableBoot`);
`internal/server` (`TestDeliveryObserverRoute`, `TestDeliveryEnvironmentRead`);
`internal/store` (the observer binding, its event and its mutation rows);
`internal/install` (`delivery_receipt_test.go`: round trip, atomic replace,
symlink refusal, a started-only record, and `DeployForce` preserving its own
error and exit code); `web/shared/domain/delivery.test.js` (the lane's copy and
the publication labels); `web/mobile/src/lib/mobileRoutes.test.js` (the lane in
the phone's route).

Scratch (`qa-scratch d2` on :8474, fixtures in `/tmp/d2-qa`): the Deployment
lens was read in every state — not connected, known with one unpublished change
(target ahead of the running revision), last attempt passed, failed (previous
version still responding), unknown outcome, no attempt recorded, a running
revision this repository does not have, a binding that names another
repository, and a claimed-by-another-workspace conflict — plus the Integration
lens's `Not published` marker and its detail tile. Captures live in
`var/screenshots/d2-*.png` and were read in a subagent; the receipts they show
are fixtures in the scratch's data dir, because the producer's real path runs
only on the owner's `picode deploy`.

## Copy (canonical, from the design contract)

- Unconfigured: **Deployment is not connected for this project.** · View setup
- Unknown identity: **Running version found; included changes are unconfirmed.** · View evidence
- Healthy: **Running <version>** · Responding · checked N seconds ago
- Unpublished: **Integrated, not published: N changes** · View changes
- Failed attempt: **Last attempt: Failed · previous version is still responding** · View evidence
- Unknown attempt: **Last attempt: Unknown outcome** (never "running")
- Busy: **Agents are still working.** · View activity

Nothing in this lane says safe, ready, approved, queued or deployed-by-PiCode.

## Verification

- Go: `internal/delivery` deployment decision table + confinement; `internal/store`
  binding + event + migration; `internal/server` read shape, config route,
  error/conflict cases; `internal/install` receipt atomicity, exit-code
  preservation, discovery identity.
- JS: domain labels and the lane's derivations (`web/shared/domain/delivery.test.js`),
  route `lane` handling.
- Scratch: unconfigured, healthy, unknown revision, failed attempt with the old
  version answering, unknown attempt, busy, and mobile; screenshots read in a
  subagent; overlay audit; five-question card.
- Docs: `docs-site/guide/delivery.md` gains the Deployment section (the View
  setup destination), `docs/architecture/delivery.md` the D2 paragraph.
