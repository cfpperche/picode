# Communication restart recovery

Owner-approved 2026-09-10; ADR-0112. Adaptation: the existing
[Paseo-style native side channel](../benchmarks/2026-09-03-guest-tui-agent-state.md)
continues observing the native TUI. The mailbox and attention worker stay shared.

## Decision table

| Condition | Action | Required evidence |
|---|---|---|
| Valid native Idle, same boot/process/run/pane | Recover identity + Idle | Six real CLIs before/after daemon restart |
| Working or approval event during downtime | Recover that state, no automatic input | Ordered hook/recovery tests + native exercise |
| New native conversation during downtime | Bind new identity, old credential unusable | Six-CLI observation contract fixtures |
| Delayed older event | Ignore; preserve newer identity/activity | Ordering regression |
| Codex modern hook followed by legacy notify | Keep modern identity/activity | Source precedence regression |
| Different PID incarnation or OS boot | Refuse recovery | PID-reuse and reboot fixtures |
| Process is outside exact pane | Refuse recovery | Real tmux ancestry fixture |
| Ended wrapper or replacement | Clear/refuse prior observation | Incarnation/end regression |
| Missing/corrupt/unreadable/public record | Unknown; no automatic input | Recovery and attention regression |
| Record write fails | Invalidate old candidate; no stale Idle | Recorder failure regression |
| Newer write fails, then an older Idle arrives | Retain ordering/source fence; refuse delayed event | Recorder failure + ordering regression |
| Ordering fence cannot be saved | Block incarnation; only a fresh wrapper can reset | Lock/fsync/permission failure regression |
| Working older than TTL | Preserve identity, activity unknown | Expired Working regression |
| Consent revoked or owner moved | No restored authorization | Existing store/attention consent tests |
| Draft or native approval | Preserve text; retain pending mail | Existing input guards + native/browser QA |
| Submission attempted before crash | Never repeat automatically | Existing durable attention tests |
| Pi hello after presence fallback | Validate exact wrapper, renew receiver | Native receiver regression |
| Hermes helper changes process environment | Use native root session/turn for messages | Contaminated-environment regression + native rerun |
| Hermes helper shares session but has another turn | Ignore its activity/tool context | Root/helper turn regression |
| Hermes completion subprocess arrives late | Retain the event's original order | Concurrent delayed-completion regression |
| Hermes message command has quoted body / compound shape | Preserve plain command bytes / leave unsupported shape unchanged | Parser and native approval contract regressions |
| Native Grok unaccepted dim-and-italic suggestion, matching empty cursor/frame/footer | Treat composer as empty; retain native identity, Idle and permission guards | `TestPeerGrokNativeSuggestion` + native rerun |
| Grok draft, partially accepted suggestion, changed style/cursor/frame/footer or copy mode | Refuse automatic input | Nine negative cases in `TestPeerGrokNativeSuggestion` |
| No current native conversation | Visible, unavailable; open conversation action | Shared state tests + browser QA |
| Reconnecting participant | Retain selector choice; test disabled | Shared selection tests + browser QA |
| Connected conversation with past passed test | Show activity, connection and proof separately | Shared state tests + browser QA |

## Follow-up decision rows — 2026-09-12

| Condition | Action | Regression |
|---|---|---|
| Host without Linux process metadata, including stale files | Keep live hooks; no recovery files or invalidation | Unsupported recorder and legacy-state tests |
| Fence is locked by a writer | Nonblocking neutral wait; no automatic input | Busy-fence test |
| Unlocked corrupt, blocked, or unsupported fence | Show connection failure and terminal controls; preserve refusal | API corrupt-fence table + shared presentation test |
| Fence is a FIFO or symlink | Refuse without blocking or reading its content | Nonregular-fence test |
| Post-paste validation refuses | Record bounded stage/reason; never retry automatically | Attention reason table and existing durable attempts |

## Acceptance

Scratch only: Pi, Grok, OpenCode, Hermes, Codex and Claude Code use their real
TUIs and configured models. Measure restoration within ten seconds after the
server becomes healthy with no native PID/session change, hidden prompt, terminal
restart or manual input. Exercise a native exchange with sender/recipient ACKs
before and after restart. Distinguish model/tool-permission failures from transport.
Record native evidence and all untested rows honestly before closing.

Browser: desktop/mobile empty, reconnecting, real approval, connection error,
connected and selector/confirmation states; screenshots read in visual review,
overlay audit clean. Scoped gates, close, fast-forward and full main CI follow.
Deployment is a subsequent owner action.

## Recorded validation — 2026-09-10

Scratch `localhost:8473`, isolated HOME/project, six actual native TUIs. An abrupt
daemon stop with all six Idle restored their identities and Idle/Connected states in
**6.045 seconds after healthy** (startup 0.307 seconds); every wrapper PID,
process start token, run and native session stayed unchanged. Hermes loaded its
updated adapter through an explicit same-session resume before this baseline.
No native terminal restart or manual input was needed during this recovery.

An earlier restart recovered all six in 6.052 seconds after healthy (startup
0.205 seconds). A Codex draft remained byte-for-byte unchanged and its pending
message stayed unsubmitted/unacknowledged until the draft was cleared.

After the Grok suggestion compatibility fix, a later backend restart restored all
six confirmed identities and connections in **0.809 seconds after healthy**
(startup 0.609 seconds), with every native PID/run/session unchanged. Grok was
Working because its pending reply was delivered immediately; it read and
acknowledged that reply automatically, without manual terminal input.

Native evidence lives in `var/qa/communication-recovery-20260910/` in the root
checkout: `restart-1789066923.json`, `restart-1789066279.json`,
`restart-1789064925.json`, `draft-before.txt`, `draft-after.txt`, and the paired
draft history snapshots. Earlier exercises
captured an event written while the daemon was down (`offline-before/after.json`,
`offline-recovery.json`) and real
Codex/Claude approvals (`native-codex-approval.json`, `claude-native-approval.json`).
An intermediate recorder-format upgrade deliberately remained unobserved until
each QA conversation emitted its next native event; it is not a recovery pass.

`native-validation.json` cross-checks eight passed diagnostics against actual
message/reply rows and both ACK timestamps in `final-history.json` and
`final-workspaces.json`. The final fresh check completed automatically at
19:04:39.797 UTC, without manual terminal input or a daemon restart during it.

| Native exchange | Stage | Passed check |
|---|---|---|
| Pi → Grok | Initial exchange | `check_U3F47REQZU5AGMMR3ZNGXZESCX` |
| Codex → Hermes | Initial exchange | `check_Y3A26BZMZQ27M6QKZQ3NLW3RUG` |
| Claude Code → OpenCode | Initial exchange | `check_L65ZUNRYSAO6L7U72ILDLJCVJK` |
| Grok → Pi | Earlier recovery run | `check_Z7KF6VN6UQITPHGD5WIVDH65WR` |
| Hermes → Codex | After the all-Idle restart baseline | `check_YURD53KLGQX3HKWFGEKFIKK6HW` |
| OpenCode → Claude Code | After the all-Idle restart baseline | `check_AIWOOMSNUQAOWQJD36VTUSKRTI` |
| Grok → Pi | Pending reply resumed after suggestion fix | `check_PHUFOWXMI5HPVBTUCCBY7M2UC6` |
| Grok → Pi | Fresh final automatic exchange | `check_OEO7C43CUOKD6M5GNBUCSHRLPE` |

Regression coverage: `native_observation_test.go` covers recovery, exact process/
pane, missing/invalid files and fences, expiry, failure plus delayed events, source
precedence, obsolete invalidation and Pi receiver recovery. Existing
`TestPeerParticipationDecisionTable`, `TestPeerIsolationAndSessionBinding`,
`TestPeerAttentionClaimsDoNotConsumeOrRepeat` and the input decision tables cover
consent, moved owners, durable attempts and drafts. The Hermes integration test
covers root/helper context, ordering and command parsing;
`TestPeerGrokNativeSuggestion` covers its captured empty suggestion and nine
refusal cases. Shared participant tests cover status/selection conditions. The
26 decision-table rows have regression coverage; physical-mobile and non-Linux
acceptance remain external. The adversarial review independently reproduced the
failure cases and passed after the fixes. `make ci-scoped` passed after both
adapter fixes; final branch close and main integration gates follow.

Browser review passed at desktop 1440×1000 (light/dark) and mobile 390×844:
empty/unknown/approval/error/connected states, disabled visible participants,
selection retained across replacement connections, readable selectors and clean
confirmation overlay audits. Screenshots are under
`var/screenshots/communication-recovery/`; the read verdict is `visual-review.md`
alongside the native evidence.

Limits: OpenCode's Z.AI GLM-5.3-Flash returned insufficient balance; its transport
was exercised with xAI Grok 4.6 instead. Its long wrapped sidebar/footer still
blocks automatic input; QA hid the native sidebar, without weakening the guard.
That layout, physical mobile and non-Linux process recovery remain open. Already
loaded old Pi receivers retain their old heartbeat until reloaded; absent native
observations cannot be reconstructed retroactively. Actual tool approvals remain
with the native CLI. Four expired diagnostics remain recorded, including native
approvals, pre-fix Hermes context drift and the interrupted paste. Checks held
at approvals past five minutes expired; later
replies did not turn those expired checks into passes. A separate Hermes check
expired after background review changed its environment; the adapter repair is
covered by the root/helper-turn, delayed-completion and command-parser regression.
One daemon stop after paste but before Enter preserved the attempted state and
the pointer in the native draft (`uncertain-paste-preserved.txt`); completion
required manual input, and the worker did not resubmit the attempt.

A separate Grok check stopped at the unchanged post-paste guard without a daemon
restart. Its pointer remained in the composer and the check stayed uncertain
(`grok-uncertain-after-final-restart.json`, `grok-uncertain-composer.txt`). QA
cleared only that task-owned pointer before starting a fresh check. This records
intermittent guarded delivery; the failing guard was not captured, and the
uncertain check is not a successful exchange. The later eight passed diagnostics
do not change its result or the four expired results.
