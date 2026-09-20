### Fixed

- **Message delivery keeps up with vendor UI changes.** Grok 1.0.34
  restyled its composer footer and Claude Code predicts dim follow-up
  suggestions — both silently stalled automatic prompts. Both shapes are
  now recognized, with the same strict frame, cursor and footer checks.
