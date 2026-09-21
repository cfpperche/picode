### Fixed
- **The Inbox no longer promises a pickup nobody made.** Answering a
  question whose source has no reply channel records the answer on the
  item (so the human's Reply closes it), and the note said "the CLI that
  asked will pick it up here" — true for `picode inbox ask --wait`, false
  for a plain `ask` and for a pi launched outside the launcher, which
  read the durable queue instead. The note and the toast now name both
  paths: a waiting asker reads it here, a non-polling asker must be told
  another way.
