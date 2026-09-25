### Changed

- **Agents are told when their PiCode tools are out of date.** After an
  update that changes how PiCode's tools talk to it, a tool call from an
  agent that was started before the update now answers "Restart the agent
  to load the new tools" instead of failing with an obscure error.
