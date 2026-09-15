### Fixed

- **One PiCode can no longer remove another PiCode's terminals.** Every
  session PiCode creates now carries `PICODE_INSTANCE`, the data directory of
  the instance that created it, and the tmux inspector reads it: a session
  from a different PiCode on the same machine is labelled *another PiCode
  instance's work* — with the text saying which one — instead of being
  offered as a removable leftover. The removal itself refuses it, so the rule
  holds even if the request does not come from the screen. Sessions created
  before this version are told apart the same way, by the loopback port their
  session environment names.
