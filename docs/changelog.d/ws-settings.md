### Added
- **Workspace settings.** A **Settings…** item on the workspace card's menu renames the workspace and sets how delivered work lands in the project: follow this machine's rules, or give the workspace its own (up to eight checks that must pass before an authorized branch lands as a fast-forward). With no rules, or fast-forward off, it says plainly that authorized branches stay blocked. It also links to the workspace's Communication page.

### Fixed
- Removing a workspace no longer leaves its integration rules behind.
- Two windows saving a workspace's integration rules over each other now get a conflict instead of silently losing one save.
- On a narrow window, pressing Escape in a dialog opened from the sidebar no longer also closes the navigation drawer.
