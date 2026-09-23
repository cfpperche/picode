# Missions: implementation and verification

The owner approved M0–M3 on 2026-09-23. ADRs [0199](../decisions/0199-mission-persistence-contract.md)
and [0200](../decisions/0200-mission-execution-transfer.md) settle the boundaries.
M4 (separate reviewer) and M5 (unattended execution) remain future work.

## Adapter inventory (M0)

This is the measured implementation boundary, not a claim that every vendor
version was exercised. Source: `internal/server/cli_tools.go`,
`internal/server/term_prompt.go`, managed Pi runtime and session bindings.

| CLI | Mission submission | Reporting | Live mission proof here |
|---|---|---|---|
| Codex | Existing recognized-composer door | Launch-injected MCP; CLI subject to native sandbox | 0.156.1: receive, acknowledge, block, transfer |
| Claude Code | Existing recognized-composer door | Launch-injected MCP; CLI when current binary is on PATH | 2.1.280: receive transfer, recover decision, evidence, review, new-session recovery |
| OpenCode | Existing recognized-composer door | Launch-injected MCP or CLI | Not run against a provider |
| Pi | Managed prompt or interactive prompt door | CLI, or configured MCP | Existing adapters and fixtures; no provider run |
| Grok, Hermes | Existing recognized-composer door | CLI or explicitly configured connector | Not run against providers |
| Muse Code, Antigravity, Omp | Manual context continuation; Missions refuses automatic paste without a reader | CLI or configured connector when available; known session required for agent writes | Not run against providers |

Missions uses existing agents. Starting a stopped CLI remains its normal
agent/terminal action. Acknowledgement is distinct from submission. Child
quiescence requires owner confirmation; an activity badge is not proof.
The native pilot exposed two environment limits: Codex shell networking was
blocked by its sandbox, and Claude's login shell found the older installed
PiCode binary. Both completed through the launch-injected mission MCP tool.
The context/help now prefer that tool when available. No global network or
provider configuration was relaxed.

## Native pilot (M2/M3)

An isolated scratch daemon served the implementation from the feature worktree
on port 8471. Executable, process cwd, daemon metadata and `/api/version` were
checked. Only newly created QA agents and a disposable Git repository were
used. Authentication was copied into the scratch home without conversation or
peer-connection files. Production agents were not used.

1. Created a mission through the browser: preserve objective/decisions/evidence.
2. Codex received the packet, acknowledged generation 1, and filed a mission
   question through MCP. Mission reached blocked at version 6.
3. Answered that question through the mobile Inbox at 390×844: use a short
   Markdown handoff. The answer updated mission context at version 7.
4. Stopped the Codex writer and transferred to Claude Code, generation 2.
   Claude read the decision, did not repeat the question, attached passing
   evidence and requested review at version 12.
5. Advanced the disposable candidate commit. Acceptance of the old review was
   refused. Restarted Claude into a new native session; a report under its
   previous assignment was refused with HTTP 403.
6. Recorded changes and rebound Claude explicitly, generation 3. It attached
   fresh evidence for the new commit and requested review at version 18.
7. Stopped the writer and accepted through the mobile UI. Mission completed
   at version 19, with a retained acceptance snapshot and no reservation.
8. Restarted the scratch daemon. Completed state and its history survived.

Evidence is retained locally under `var/qa/missions-mvp/native-pilot-evidence.json`
and `var/screenshots/missions-*.png` (not committed). The native pilot is one
controlled task, not the five-mission owner workflow pilot in the product plan.
It does not establish behavior in the Windows native shell or on a physical phone.

## Decision-table coverage

Test names refer to `internal/store/missions_test.go`,
`internal/server/missions_test.go`, and the existing prompt/Delivery suites.

| Row | Evidence / boundary |
|---|---|
| D01–D02 | `TestMissionRetryVersionAndHistory`; persisted browser request keys; editable conflict review/rebase |
| D03 | Mission state changes only through `ApplyMission`; idle/native exits do not infer acceptance; native pilot stops preserved review |
| D04 | `TestMissionAssignmentDecisionTable`, `TestMissionHTTPRoundTripAndIdentity`, MCP inherited-identity test |
| D05 | Assignment table requires explicit source confirmation; server reuses `agentBusy`; arbitrary child-process quiescence is owner-reported |
| D06 | Reservation/transfer tests and real Codex → Claude transfer, generations 1 → 2 |
| D07 | Dispatch cannot repeat; restart fixture retains unconfirmed reservation; live Claude submission reported unconfirmed before attributed acknowledgement |
| D08 | Existing `TestTerminalPromptDecisionTable`; missing target/source refusals and manual continuation for CLIs without a reader |
| D09 | `TestMissionSurvivesRestartAndAgentRemoval`; scratch daemon restart retained completed mission/history |
| D10 | Assignment decision table retains reservation on pause/cancel and refuses late agent updates |
| D11–D13 | Evidence/revision tables, dirty-candidate test, file confinement, native stale-candidate refusal and explicit re-review |
| D14 | `TestMissionInboxAnswerAfterTransfer`; stale answer retained without changing new assignment |
| D15 | Restart/removal test, `TestMissionRelinkKeepsGenerationAndInvalidatesEvidence`; unavailable sources remain readable |
| D16 | `TestMissionDeliveryReferenceDoesNotTransferAuthorship`; existing Delivery actor checks |
| D17 | `TestMissionTransferRequiresPreparedCleanWorkingFolder` |
| D18 | M4; no separate reviewer role exposed |
| D19 | M5; no unattended scheduler exposed |
| D20 | Concurrent reservation test; target-readiness confirmation and existing composer draft gate |
| D21 | `TestMissionConcurrentPauseAndDispatchOrAcknowledgement`; one version winner, reservation retained, late acknowledgement refused |
| D22 | Wrong-session fixture and live Claude native-session rollover/rebind, generations 2 → 3 |

Additional regressions cover capacity cleanup, unborn repositories, CLI owner
verb refusal, schema/route behavior, dirty evidence capture, first Git initialization, exact retry payloads and
bounded context packets. Full CI and
visual verdicts are recorded in the branch handoff. Hard-killing the daemon
at every individual native submission instruction is not covered by the pilot;
the durable pre-send receipt and no-retry contract are covered by store fixtures.

## Delivery limits

The five real owner-workflow missions, physical phone use and Windows shell
acceptance follow deployment at the owner's choice. Do not infer those results
from browser emulation. The user controls the executor, acceptance and deploy;
Missions does not enforce a process sandbox or turn this implementation into
an unattended scheduler.
