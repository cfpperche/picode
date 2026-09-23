### Fixed
- Opening the desktop Management window no longer pops up a terminal window: every measurement and cleanup now runs in the background.
- The steps of **Give back held space** now show in the window while it runs. Before, the window never received them.

### Changed
- The Management window now shows its scan as it happens. Windows and the distro each get a row with a spinner and a clock, and each card fills as soon as its half is read. **Scan again** keeps the previous numbers on screen until the new ones arrive.
- The Clean tab uses the same scan as the Disk tab, so opening the window walks the home directory once instead of twice.
