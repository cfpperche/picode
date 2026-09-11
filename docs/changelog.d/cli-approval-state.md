### Fixed
- **Approving a permission prompt no longer leaves the row saying "Needs
  you" while the agent works.** Grok, Claude Code and Codex report a waiting
  prompt as `needs-you`, but none of them emits a "permission resolved"
  event, and none of them registered a tool-activity hook to contradict it —
  so the row stayed `Needs you` for the rest of the turn. A Grok plan-mode
  approval showed it for 10+ minutes after the plan had been approved. The
  tool lifecycle is now the resume signal (`PostToolUse`, plus
  `PostToolUseFailure` where the CLI has it), so the first tool that runs
  after the answer returns the terminal to `Working`.
- A Grok notification that is not a permission prompt (`task_complete`, …)
  no longer turns the row into a false `Needs you`; only a permission UI
  that is actually waiting does.
