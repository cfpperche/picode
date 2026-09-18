### Fixed
- **Desktop: the shell no longer bricks on a daemon restart.** Back-to-back deploys could leave the desktop app parked on a 404 body forever. The shell now waits for the app to really answer before first paint, and an automatic reload waits for the page to serve again (it never reloads into a dead body).
