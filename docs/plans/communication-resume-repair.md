# Native communication resume repair

Fix the enable → exact native resume → available path without model prompts,
credentials shared between conversations, or changes to native CLI homes.
Existing boundaries: ADR-0107 and ADR-0110; Codex applies the shared native client in ADR-0111.

## Decision table

| Conditions | Required outcome | Verification |
|---|---|---|
| OpenCode plugin initialization with an exact resume candidate | Return hooks without calling an instance-dependent API | Native plugin initialization test |
| OpenCode resume after initialization with verified root and native status | Report idle only for absent/explicit idle entry; busy/retry stay working | Native status decision test and real resume |
| OpenCode native activity before timer or while status request is in flight | Invalidate the startup snapshot; never erase a permission/busy state | Pre-timer permission and in-flight busy regressions |
| OpenCode malformed/unknown/failed status | Remain unobserved | Native status decision test |
| OpenCode event for resumed root | Verify metadata, then report its native state | Native plugin decision test |
| OpenCode child, other root or failed metadata lookup | Never authorize input or replace the selected root | Native plugin decision test |
| OpenCode new native user message | Select only the verified root; preserve ordered state | Native plugin decision test |
| Codex identified conversation, enrollment enabled | Discover private setup without restarting or adding a model turn | Native fixture; same PID/session before and after |
| Codex native tool ID/alias/owner mismatch or duplicate setup | Refuse before network access | Native CLI identity and compatibility tests |
| Existing Codex terminal without discovery marker | Match native tool thread and owner under the PiCode data root | Native compatibility tests |
| Grok/Hermes with ambient parent Codex variables | Keep their own native identity | Ambient identity regression test |
| Codex modern hook followed by auxiliary legacy notify | Preserve the selected session and activity | Same-run observation test |
| Codex legacy-only wrapper with inherited modern flag | Clear the flag and accept native notify | Actual wrapper selection test |
| Codex SessionStart / Stop / child event | Add discovery context only to native root startup | Actual hook script output test |
| Codex compaction/child/stale event | Keep working or ignore; never authorize idle input | Hook decision tests |
| Idle terminal needs connection setup | Preserve captured width and height through exact resume | Sized launch test and native fixture |
| Claude empty composer, narrow remote-control footer | Allow a short pointer with the same native identity gates | Composer decision test and native fixture |
| Claude draft, permission prompt, foreign footer or copy mode | Retain pending notification; never overwrite input | Composer decision test |
| Native send + correlated reply + both acknowledgments | Pass; setup alone never passes | Real CLI diagnostics |

## Native acceptance

Evidence is collected in `var/qa/communication-resume-repair/` on a separate
PiCode instance. Fixture IDs are recorded before operating any terminal.
Production test terminals remain owned by the user.

- Three fresh native conversations were enabled together at 80×24. Claude and
  OpenCode resumed the same IDs at the same dimensions; Codex retained the same
  PID and native ID. All became connected without an extra setup prompt.
- Codex 0.154.0 / gpt-5.6-luna ↔ OpenCode 1.18.30 / xAI Grok 4.6:
  `check_IGEFOFCT3RNNRTNZDO3ZZE4QYD` passed at 14:28:16Z on 2026-09-10.
  Codex's default sandbox blocked loopback initially; explicit native approvals
  were granted for the diagnostic commands, without disabling its sandbox.
- OpenCode ↔ Claude Code 2.1.267 / Sonnet 5:
  `check_IKGCCAGRLAZJ6PLYTEMSG5H5NS` passed at 14:28:39Z. Claude remained detached
  at 80×24 through delivery, reply and ACK. No browser resize was needed.
- All four durable message rows have acknowledgement timestamps. Native Codex
  history contains the SessionStart command discovery as a developer-context item.
- Z.AI refused the earlier scratch call for insufficient balance, so OpenCode's
  successful test used xAI, matching the owner's current test terminal.
- The final build also resumed OpenCode's same native conversation and returned
  to connected; `final-build-opencode-resume.json` records the binding.
- Scratch visual review passed for empty, blocked, overlay, verified and native
  OpenCode/Claude panes. The overlay audit passed; hover was not captured.

Limits: this is Linux/tmux acceptance for these versions, not a new six-provider
matrix. Very long wrapped OpenCode cwd/footer layouts remain conservatively
blocked by the existing composer guard. A fresh conversation still needs its
first native identity event; native tool approval and model capacity still apply.
