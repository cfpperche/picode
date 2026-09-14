### Fixed

- **Inbox replies reach legacy terminal questions.** An `ask_human` item
  from pi in an Agent CLI terminal that predates per-question session
  stamping can now be answered from the Inbox when the terminal's pinned
  session and the conversation its receiver is showing agree; the item
  still stays open with an honest refusal when they disagree.
