### Fixed
- **Muse Code's plugin catalog is actionable.** A catalog row now carries the
  spec the CLI installs from (`name@marketplace`) and the path the marketplace
  resolved, so **Install** in the Marketplace tab runs the right command.
- **Marketplace rows say when a plugin is already installed.** Muse's own
  catalog keeps answering `available` after an install, so PiCode joins the
  catalog with the CLI's own list and shows **Installed** instead of an Install
  button for something that is there.
