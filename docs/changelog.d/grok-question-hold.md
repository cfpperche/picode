### Fixed
- **Grok terminals show Needs you while a question card waits.** The
  `ask_user_question` card notified an `elicitation_dialog` notification that
  the hook map ignored, so the row stayed on Working for the whole wait. The
  question is now reported as a held attention, and it survives the sibling
  tool completions Grok runs in the same parallel batch — the hold ends only
  when the question tool itself completes.
