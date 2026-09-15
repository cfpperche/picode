### Added

- **Servers** in the inspector rail: what is listening on this machine, who owns it (the PiCode terminal or agent whose process holds the port) and what the page calls itself, with one **Open** that shows it in PiCode's own browser tab.
- A loopback URL printed in a terminal (Ctrl+click, or the pane menu's **Open …**) opens in PiCode's own browser instead of the system browser — unless Browser settings says local development sites should open externally, in which case nothing changes.
- Without the desktop app, that browser tab renders a page from this machine in a frame, with one line saying so, and its address survives a reload.

### Fixed

- The work-browser tab crashed the whole shell: an address-bar preference was read by an effect before its own state existed (`ReferenceError: Cannot access … before initialization`, blank window on every work-browser tab). The tab strip also never received the tabs' addresses, so every web tab read "New tab".
