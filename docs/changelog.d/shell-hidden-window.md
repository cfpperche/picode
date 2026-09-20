### Fixed
- **The desktop app could not be opened from the tray.** With the resident
  started by the logon task, *Open PiCode* (and a second launch) did
  nothing: the window was alive but the app's own lookup missed it, so
  every open path quietly no-op'd. The shell now keeps its window handle
  and asks Windows directly to restore and focus it.
