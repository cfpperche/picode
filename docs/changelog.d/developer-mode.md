### Added
- **Developer mode — full CDP access** in Settings ▸ Browser, off by default
  and labeled Elevated risk: an agent at the Full tier can name any Chrome
  DevTools Protocol method, not only the curated ones, and every call it
  makes — allowed or refused — is listed with the method, the agent and the
  outcome. Everything keeps working with it off.

### Fixed
- **The JavaScript switch reaches the shell on load**, not only when it is
  clicked: after an app restart the shell's copy of the setting is now the
  saved one, not its default.
