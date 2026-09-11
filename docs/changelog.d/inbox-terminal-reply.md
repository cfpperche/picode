### Fixed
- **Inbox: ignoring a question from a pi in an Agent CLI terminal now
  closes it.** Ignore sends nothing, so it no longer needs that terminal to
  be live and on the same session — it never claims "no reply" and then
  refuses to be dismissed.
- **Inbox: a reply to a terminal question fails with the truth.** A pi in
  that terminal that had not opened a conversation (a nested `pi -p`, a
  print-mode run) could take the reply file and answer "the terminal is
  showing a different session". The daemon now refuses before parking when
  the receiver names no session, and each reply file is addressed to the
  process whose hello was accepted, so another pi in the same terminal can
  no longer consume it.
