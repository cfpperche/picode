### Fixed

- **The tray reopens the shell again.** Closing the shell window destroyed it
  instead of hiding it, so tray click and **Open PiCode** silently did
  nothing. Close now hides the window; reopening returns to the same page.
- **No more console window beside the shell.** The shell built as a console
  app, so every start opened a terminal window titled with the exe path that
  lived as long as the tray icon. It now builds as a GUI app (debug builds
  keep the console).
