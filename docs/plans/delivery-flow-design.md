# Delivery flow D0 — surface and implementation contract

Status: D0 design complete, 2026-09-21. No product UI or delivery executor is
implemented. Parent: [execution baseline](delivery-flow.md). Evidence and measured
limits: [inventory](delivery-flow-evidence.md). Boundary proposal:
[ADR-0170](../decisions/0170-delivery-observation-contract.md), awaiting owner decision.

## Product decisions

Use the existing project Git surface, adding **Delivery** alongside its history
view on the browser and alongside Changes / History / Pull request on mobile.
Inside Delivery use **Integration** and **Deployment**. No new global sidebar,
app/plugin, dashboard grid or Inspector tab. This is host project tooling; it does
not open a new guest-app door under ADR-0109. Text below is canonical English copy.

The unit is a change identified by repository, full source ref and observed
revision. Multiple associated agents may appear. A branch is the initial
candidate, not a proven task or an enrolled queue item. Deleted/rebased changes
require historical evidence; observations alone cannot reconstruct task identity.

Take t3code's links between change and conversation, Paseo's review visibility,
Linear's integrated-versus-delivered distinction, and GitHub's explicit blockers.
Sources and limits remain in the [benchmark study](../benchmarks/2026-09-21-delivery-governance.md).

## Desktop wireframe — illustrative data, not live status

```text
[PiCode project v]   Git: [History] [Delivery]             [Refresh]
Delivery             [Integration] [Deployment]
Target [main v]       [Needs attention v]      Checked 12 seconds ago

Change                 Associated agents    Integration       Checks
Fix sign-in            Codex                Not integrated    Needs recheck
  The target changed.                                        [View details]
Improve keyboard       Claude, Pi           Not integrated    Passed
  Review has not been recorded.                              [Review changes]
Update help            Not recorded          Integrated        Failed
  The project checks failed after integration.                [View evidence]

Showing 3 observed changes · Review and queue status are not recorded
```

Desktop detail uses an in-pane region, not a blocking modal: title, revision,
explicit target, blocker, files/history links, associated agents and evidence.
The first line answers the problem; hashes, paths and check scopes are expanded
details. At narrow widths rows stack and detail replaces the list with Back.

```text
Delivery             [Integration] [Deployment]
Environment [This PiCode instance v]                   [Refresh]
Running version 0.3.1+abc1234       Responding · checked 12 seconds ago
Integrated, not published: 2 changes                   [View changes]
Last attempt: Failed · previous version is still responding
  Build failed.                                       [View evidence]
```

Only show Last attempt with a structured attempt receipt. Legacy data says
Last recorded deployment, never Last attempt. Failed attempt and live environment
are separate facts. No deploy button, queue order, ETA or approval control in D1/D2.

## Mobile wireframe — same facts, independent presentation

```text
< Back                    Git · PiCode
[Changes] [History] [Pull request] [Delivery]
[Integration] [Deployment]
Target [main v]                  [Refresh]

Fix sign-in
Not integrated · Needs recheck
The target changed.
Codex                           [View details]

Improve keyboard
Not integrated · Checks passed
Review has not been recorded.
Claude, Pi                      [Review changes]
```

Allow horizontal scrolling of the Git tab strip with visible selected tab; never
shrink labels to unreadability. A row opens a pushed detail screen; Back restores
filter and scroll. Project/target selectors use mobile-owned sheets/native controls,
not desktop imports. Deployment uses the same compact vertical fact list. No
horizontal data table on a phone.

## Navigation and interactions

- Existing `#/git/<owner-kind>/<id>` authorization/deduplication stays. Add optional
  view query `?view=delivery&lane=integration` (deployment for the other lane),
  interpreted by each app's route helpers. Existing links default to their current
  view. The view parameter conveys no path, permissions or execution intent.
- Browser keeps the `g:<repository-key>` tab; mobile keeps its own navigation.
  Explicit project picks clear change detail and evidence before loading the new
  owner. Root mismatch offers Follow folder; never silently changes the project.
- Target selection is an explicit local branch, defaulting to main only when it
  exists in the PiCode pilot. Otherwise Select target; never guess main/master or
  treat upstream as the target. Changing target recomputes every state.
- Default order: known failed/blocked, needs verification, other not integrated,
  then integrated; stable title/ref tie-break. This is attention sorting, not an
  integration execution order. All filter removes no candidates silently.
- Primary actions are read-only: Review changes, View evidence, Open agent,
  Refresh, Back. Review opens existing scoped diff/history; opening an agent does
  not send a prompt. If there is no explicit session association, link the agent
  only and say Session not recorded in details.
- Refresh keeps last successful rows, adds Updating and prevents overlapping
  requests. First load uses rows shaped like the final list; no fictional names.
- Controls use existing tokens, Radix/native patterns and `--ctl-h`. Keyboard
  focus remains on the selected row after refresh; announce status updates once.
  Color is supplemental to words. Reduced-motion preference disables transitions.

## Empty, blocked and error copy

| Condition | One line | One action |
|---|---|---|
| No repository | This folder is not a Git repository. | Back |
| No chosen target | Choose the branch changes will join. | Select target |
| No unintegrated observed changes | No changes waiting to join this branch. | View history |
| Filter hides every row | No changes match this filter. | Clear filter |
| No deployment observer | Deployment is not connected for this project. | View setup (public help) |
| No validation record | Checks have not been recorded. | Review changes |
| Invalid/stale validation | These checks do not cover the current change. | View evidence |
| Known check failed after integration | Integrated; project checks failed. | View evidence |
| Incomplete scan | Some changes could not be checked. | Retry |
| Source changed during collection | The project changed while being checked. | Refresh |
| Initial read error | Could not read delivery status. | Retry |
| Refresh error | Could not update; showing the last check. | Retry |
| Connection lost | Offline; showing the last check. | Retry |
| Owner folder moved | This project's folder changed. | Follow folder |
| Production identity cannot map to Git | Running version found; included changes are unconfirmed. | View evidence |
| Busy guard reports owners | Agents are still working. | View activity |

Setup help is required in D2; no dead setup link ships. Detail fallback for missing
receipts is a short explanation with Back, not a broken View evidence link.
Busy status is a reported guard observation, not a universal deployment safety claim.

## State model and precedence

Keep separate axes: `integration`, `validation`, `review`, `publication`, and
`observation`. Do not compress them to a single progress bar.

| Conditions | State / action | Planned test |
|---|---|---|
| Repository unreadable, unsupported or collection incomplete | unknown/partial, never empty/all clear | O01 |
| No target or target disappeared | target-required; select again | O02 |
| Clean branch, agent idle, no review intent | not integrated; review unknown; no Ready badge | O03 |
| Dirty checkout | changes in progress; no eligible candidate | O04 |
| Source is ancestor of resolved target | integrated regardless of later check failure | O05 |
| Source not included; target ancestor of source | fast-forward possible; review/validation independent | O06 |
| Neither revision contains the other | update needed, not an asserted textual conflict | O07 |
| Missing/corrupt scoped stamp | validation unknown; keep Git facts | O08 |
| Clean identical tree and compatible successful evidence | scoped checks passed, not full CI or approved | O09 |
| Covered blobs match but main validation unknown | historical reusable scope; overall readiness unconfirmed | O10 |
| Covered blobs differ or checkout dirty | needs recheck | O11 |
| Included source, matching full CI failure | integrated / checks failed | O12 |
| Cwd association only or several agents in checkout | Associated agents; authorship/session unknown | O13 |
| Ref deleted; retained receipt source OID is reachable | retain historical change; membership from OID | O14 |
| Ref deleted without receipt or objects unavailable | incomplete history; do not claim all tasks accounted for | O15 |
| No configured environment | publication unconfigured | O16 |
| Semver-only, ambiguous short hash or dirty/unproven artifact | included changes unknown | O17 |
| Stable boot, bound full revision and target ahead | responding; show exact included/unpublished OIDs | O18 |
| Health/version collection crosses boot change | retry once; then observation unknown | O19 |
| Failed deployment, previous revision responds | failed attempt + previous live revision, separately | O20 |
| Start receipt has no terminal result | outcome unknown, not forever running or automatic failure | O21 |
| Fresh Git event but gate evidence changed without event | bounded refresh reconciles; timestamp remains visible | O22 |
| Target/source refs move during collection | retry once; mark unstable, no readiness claim | O23 |
| Owner/root changes or old request resolves late | discard response; retain original context until explicit follow | O24 |
| Busy guard read errors or missing coverage | unknown coverage, never Safe to deploy | O25 |
| Same source appears in multiple changes in one publication | list changes; deduplicate commits in publication set | O26 |

O01–O15, O22–O24 are D1 requirements. O16–O21, O25–O26 are D2 requirements.
Tests do not exist yet; this table is the contract for implementation. Semantic
squash/cherry-pick equivalence is unsupported; show unconfirmed inclusion rather
than asserting that identical-looking text is the same delivered change.

## Data and refresh contract proposed for implementation

`GET /api/{workspaces|agents|terminals}/{id}/delivery?target=<local-ref>&root=<expected-root>`
returns `schemaVersion`, `repositoryKey`, `target {ref, oid}`, `observedAt`,
`coverage {complete, issues}`, `changes[]`, `environments[]` and source-specific
observations with `status: known|unknown|stale|unsupported` and `reasonCode`.
Zero/false are real values, not substitutes for unknown. Source and target OIDs
are read before/after; one retry then unstable. This route does not exist today.

A change contains an observation ID (repository/ref/source OID), source OID,
worktree presence/dirty status, integration and validation facts, associated
agent IDs, optional explicit session refs, and sanitized evidence references.
New revision means new evidence scope. ADR-0171 now supplies a stable ID for explicitly registered declarations;
this does not reconstruct historical tasks from Git. Native session association
and execution history remain separate work. Repository commit history remains
navigable if no receipt tracks a branch.

Evidence references return bounded structured summaries, never arbitrary filesystem
paths for the browser to fetch. D1 adds parity tests for ADR-0124 and explicit
subprocess errors/timeouts; it must not parse human `worktree-status` output or
execute repository-supplied scripts when reading.

Refresh on entry, focus, feed reconnect/reset, and relevant Git/agent changes.
While Delivery is visible, a shared client refresh loop requests at most once per
15 seconds; server deduplicates in-flight reads per repo/target and caches up to
5 seconds. Explicit Refresh bypasses cache with the same concurrency bound.
Stop when hidden; on return, refresh immediately. Mark observations older than
30 seconds stale (thresholds are initial implementation values, tune from evidence).
The exception to feed-only updates is justified: current Git events omit clean
commits and gate/deploy receipt changes. No new fleet-wide polling is proposed.
Initial scan cap is 32 checkout statuses per response, with explicit omitted count;
all local refs remain discoverable and additional statuses load on demand. Bound
whole collection to 10 seconds and return partial coverage, not zeroes, on timeout.

D2 pilot observes only the explicitly connected local PiCode instance. Configuration
binds workspace, canonical repository key and observer `picode-self`; a changed
binding invalidates old evidence. Remote URLs, credentials and provider execution
are not accepted. See ADR-0170 for receipt and configuration ownership.

## Ready-to-build package and acceptance

D1 touches owner-scoped server reads, validation/land receipt producers, pure shared
delivery helpers/client, browser Git content and mobile Git screens. D2 adds explicit
local observer configuration, deployment receipts and runtime identity observations.
Shared code has no React UI;
no changes belong in desktop shell window chrome.

Required receipts and protocol/configuration changes are a proposed boundary;
owner approval of ADR-0170 is needed before those changes, not to finish this D0
specification. D3/D4 remain separate authority decisions. No accepted ADR is amended
by this document.

D1 visual acceptance: populated, empty, first-load failure, stale/refetch failure,
partial scan, root moved and target selection on desktop and phone; keyboard,
narrow layout, reduced motion and overlay geometry. D2 adds unconfigured, healthy,
unknown revision, failed attempt with old production and busy observation. Use
scratch, screenshots read in a subagent and the standard five-question visual card.
No D0 screenshot verdict is claimed: these are wireframes and code-path inspection.

First pilot task: identify one blocked change, its reason and evidence, then list
integrated changes absent from the connected runtime. Count terminal lookups and
elapsed human task time; compare with the D0 source-lookup baseline without
presenting machine-query latency as human time. Real queue wait remains unavailable
until explicit operation events exist in D3/D4.

## D1a implementation clarification

The common [agent interface](delivery-agent-interface.md) records explicit
declarations independently of discovered Git candidates. A future observation
view must distinguish these, correlate by repository/ref/revision and retain the
stable declaration ID across updates. `review: requested` is attributed intent
for the recorded revision, never human approval or queue enrollment. This
addition does not implement the read endpoint or screen specified above.
