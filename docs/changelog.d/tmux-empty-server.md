### Fixed
- **Creating a terminal right after closing the last one no longer fails.**
  When tmux's server had just emptied — the moment between the last session
  ending and the server exiting — its `has-session` answer, *"no current
  target"*, was read as a real failure, so PiCode refused to create a session it
  should simply have created. The answer now means what it is: not there.
