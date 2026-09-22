### Changed

- A CLI that was still running in a terminal with no agent becomes an agent on
  upgrade, keeping its browser and computer permissions. Permissions now belong
  only to agents: a CLI typed into a plain shell gets none (the computer tool
  refuses it, the browser reads the tab on screen) and cannot register
  deliveries until you make it an agent. Settings ▸ Browser and Settings ▸
  Computer list agents only.
