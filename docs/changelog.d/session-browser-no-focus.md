### Fixed

- **Desktop browser: an agent driving its browser no longer pulls you to its
  tab.** Every click, type, snapshot or screenshot used to reselect the
  agent's tab, so you could not use another agent while one was browsing.
  Now only opening the browser brings that agent's tab forward; after that
  the agent keeps working in its own split while you stay where you are. If
  a screenshot cannot be taken while the split is off screen, PiCode shows
  the split once and tries again rather than failing the agent.
