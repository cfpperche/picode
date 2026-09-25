# 2026-09-25 — mobile-fork: Fork agent… on the phone

Paid the agent-fork debt "the phone hides Fork agent…". `web/mobile/src/components/
ForkAgentSheet.jsx` (MobileSheet) mirrors the desktop dialog: name, where (new worktree
through the git door `lib/gitDelivery.js`, or same folder), task with photo/file/folder
attachments (no sketch on the phone yet), the shared `forkRequest`. A worktree command
that waits for Enter closes the sheet, opens that terminal and the fork follows in the
background (`onForkPrepared`). Row menu no longer filters "fork"; `IconFork` on mobile.
Found in QA: a phone attached to the new agent's terminal narrows its pane to 44
columns, and the prompt door refused every pane under 70 columns and needed Pi's footer
"%/" (cut at that width) — the fork's task was never delivered (unattended door =
"unrecognized"), and the phone's attach bar only ever typed blind. `peerPiInput` now
accepts Pi's empty editor down to 30 columns by its full-width rules and folder line
(footer only has to be present below 70); other CLIs keep the 70-column gate. Test:
TestNativePiComposerOnANarrowPane with the real 44-column capture.
Verified live on a scratch at 390 px: menu → sheet → fork (same folder) → task
"responda apenas MOBILE5\ne nada mais" in the copy at ~9 s, line break kept.
