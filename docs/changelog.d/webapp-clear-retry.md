### Fixed
- **Apps: Clear data now waits its turn.** Clearing right after using an app could fail silently ("Could not save the web app") because the app's files were still being released. The clear now waits up to ten seconds and, if anything still blocks it, says exactly what.
