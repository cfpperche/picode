### Fixed
- **Clicking “PiCode” opens the dashboard again in the desktop app.** The
  wordmark in the window's top row was a window-drag region as well as a
  button, so pressing it dragged the window and the click never arrived;
  the sidebar wordmark also did nothing while a non-workspace page (Agent
  CLIs, Browser, Preferences, Devices) was open. Both now open the
  dashboard, and the top row stays draggable around the button.
