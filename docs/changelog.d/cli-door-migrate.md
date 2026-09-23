### Changed

- A CLI that was still running in a terminal with no agent becomes an agent on
  upgrade, keeping its browser and computer permissions. Permissions now belong
  only to agents: a CLI typed into a plain shell gets no grant (the computer
  tool refuses it; the browser keeps only its own split beside the session)
  and cannot register deliveries until you make it an agent. Settings ▸ Browser and Settings ▸
  Computer list agents only.
