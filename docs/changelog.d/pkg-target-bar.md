### Changed

- **Agent CLIs → Packages: the install target sits with the action it
  modifies.** The "This machine / workspace / agent" radios left the row they
  had under the source input — where they read as if they belonged to the
  Installed list below — and now sit in the install bar, labelled **Install
  to**, immediately before Install. Every control in that bar shares the 36 px
  control height, and the bar wraps as a unit (source, then target plus
  Install) when the pane is narrow.
- The redundant workspace line under the Packages tab bar is gone: it repeated
  the workspace the target pill already names, and it sat hard against the tab
  rule. The pane owns a 12 px gap under the tabs instead, on Packages and on
  the pi-roles settings sub-page alike.
