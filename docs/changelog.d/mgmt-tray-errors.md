### Fixed
- Opening **Management** from the tray no longer freezes the tray while WSL is slow. The window opens right away at the address PiCode is already answering on.
- The desktop app no longer waits a full 30 seconds before showing its main window at startup or after **Open PiCode** rebuilds it. The check that waits for PiCode to be ready was probing the distro's name instead of PiCode's address, so it could never succeed.
- When **Give back held space** fails before anything stops, the Management window no longer says the distro was restarted. When it fails after the stop, the window now says that the distro's sessions ended.

### Changed
- Management errors now say what happened and what to do, such as "PiCode inside the distro is older than the desktop app. Update PiCode, then scan again." The tool's original message stays underneath in smaller text.
