### Fixed
- **Attach now verifies the terminal sent the message.** The paste plus
  Enter raced the TUI render, so the text sat in the composer while the
  bar cleared. The paste path polls the live pane until the composer
  reads empty, retries Enter once, and answers 502 `staged` (bar stays
  open with the reason) instead of a blind success. TUIs without a reader
  keep the previous behavior.
- **The attach bar closes after sending** and hands the pane its keyboard
  back. Failures keep it open with the text staged, as before.
