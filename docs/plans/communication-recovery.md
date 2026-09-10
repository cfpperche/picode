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
| No current native conversation | Visible, unavailable; open conversation action | Shared state tests + browser QA |
| Reconnecting participant | Retain selector choice; test disabled | Shared selection tests + browser QA |
| Connected conversation with past passed test | Show activity, connection and proof separately | Shared state tests + browser QA |

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

Scratch `localhost:8473`, isolated HOME/project, six actual native TUIs. A final
abrupt daemon stop restored all six identities and connections in **6.052 seconds
after healthy** (startup 0.205 seconds); every wrapper PID, process start token,
run and native session stayed unchanged. A Codex draft remained byte-for-byte
unchanged and its pending message stayed unsubmitted/unacknowledged until the
draft was cleared. No native terminal was restarted during this recovery.

Native evidence lives in `var/qa/communication-recovery-20260910/` in the root
checkout: `restart-1789064925.json`, `draft-before.txt`, `draft-after.txt`, and the
paired draft history snapshots. Earlier exercises captured an event written while
the daemon was down (`offline-before/after.json`, `offline-recovery.json`) and real
Codex/Claude approvals (`native-codex-approval.json`, `claude-native-approval.json`).
An intermediate recorder-format upgrade deliberately remained unobserved until
each QA conversation emitted its next native event; it is not a recovery pass.

Regression coverage: `native_observation_test.go` covers recovery, exact process/
pane, missing/invalid files and fences, expiry, failure plus delayed events, source
precedence, obsolete invalidation and Pi receiver recovery. Existing
`TestPeerParticipationDecisionTable`, `TestPeerIsolationAndSessionBinding`,
`TestPeerAttentionClaimsDoNotConsumeOrRepeat` and the input decision tables cover
consent, moved owners, durable attempts and drafts. Shared participant tests cover
status/selection conditions. The adversarial review independently reproduced the
failure cases and passed after the fixes.

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
with the native CLI. Two checks held at approvals past five minutes expired
honestly; later replies did not turn those expired checks into passes.
