# Native CLI attention matrix

Validate Pi, Grok, Hermes, Claude Code, Codex and OpenCode in isolated real TUIs.
Keep initial launch separate from a conversation initialized by a user prompt.
A successful exchange requires a native request, reply and both acknowledgements.
Do not infer delivery from connection state or mailbox acceptance.

| Condition | Required action |
| --- | --- |
| Fresh launch after native trust, no model turn | Connect only with native identity; deliver if available, otherwise report waiting |
| Claude session ID observed but transcript not saved | Keep the current process; never resume a nonexistent conversation |
| Initialized conversation, empty input and Idle | Deliver and acknowledge native request and reply |
| Unsent draft | Retain pending attention and byte-identical draft; deliver only after owner clears it |
| Native turn in progress | Retain pending attention without sending or interrupting |
| Idle after resize | Recognize supported layout; unknown/too-narrow layout stays pending |
| Daemon restart with native terminals intact | Recover same identities, preserve drafts and resume safe delivery |

Results distinguish native transport acceptance from automatic identity availability.
Credentials, scratch state and captures remain uncommitted under var/qa/cli-attention-matrix.

## Validation evidence

Evidence lives under the root checkout's `var/qa/cli-attention-matrix/` and is not committed.

| Scenario | Verified result |
| --- | --- |
| Fresh Pi and Grok, no model prompt | Native request, reply and acknowledgements passed |
| Fresh Codex, Hermes and OpenCode | Native identity unavailable until a first prompt; this is not cold-start delivery acceptance |
| Fresh Claude Code | Enabling before a saved transcript now preserves the process and waits; after a first message, exact-ID resume and OpenCode → Claude communication passed |
| All six with unsent drafts, daemon restart | Before/after pane snapshots identical, including drafts and pane identity |
| Pi → Grok, Hermes → Codex, Claude Code → OpenCode | Sender diagnostic waits behind draft; recipient attention remains pending with draft intact; clearing drafts after resize to 120×40 yields request, reply and both acknowledgements |
| Native busy delivery, all three pairs | Attention waited during native turns; after completion, request/reply and both acknowledgements passed |
| Native CLI stop/resume | Pi, Grok, Claude Code and OpenCode identities returned automatically; Codex and Hermes required an explicit first `RESUMED` prompt |
| Grok → Pi, Codex → Hermes, OpenCode → Claude Code after resume | All three native exchanges passed with both acknowledgements after the initialization noted above |

The draft/resize, busy and post-resume exchanges have matching request/reply acknowledgement receipts in `*-proof.json`, `*-busy-proof.json` and `*-resume-proof.json`. `claude-guard-proof.json` confirms waiting-to-connected retained the exact native session ID. `resume-initialization.json` records which resumed CLIs required a first prompt.

The initial Codex-sender diagnostic expired: the model ignored the diagnostic and later used an incorrect request ID after a manual nudge. Its eventual mailbox acknowledgement does not count as a passing diagnostic. The pointer now explicitly asks the agent to execute `connection_check`; the final initialized Codex-sender test passed without a manual nudge.

`make close` passed on merged main content, including the final diagnostic wording. The merged browser build passed. Scratch browser/mobile happy, waiting, empty and overlay visual review passed, including fresh Claude displaying Idle, Waiting to connect, a first-message instruction and Open with successful overlay audits; the merged `/browser` and `/mobile` entry points were visually rechecked.

Final read-only receipt validation confirmed 11 passed diagnostics with matching replies and both acknowledgements. Two earlier failed checks (Pi draft attention uncertain and initial Codex diagnostic expired) remain recorded as failures. Seven owned terminals were deleted, the scratch server stopped and no processes with the exact scratch HOME remained (`cleanup.json`). Main full CI remains the integration gate. No production deployment is part of this validation.
