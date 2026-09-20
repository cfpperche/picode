### Changed
- **The computer tool acts only in the window the agent last saw.** A
  click, a keystroke or typed text is refused (`foreground_changed`) when
  the window in front is no longer the one the agent last captured or
  focused, so the agent looks again instead of typing into what you are
  using (ADR-0156, the first refinement of ADR-0148).
