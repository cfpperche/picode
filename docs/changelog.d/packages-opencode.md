### Fixed
- **OpenCode: removing a plugin works from the packages pane.** Every removal
  used to be reserved for the CLI's job lane, which runs a command OpenCode does
  not have, so the pane answered *"this CLI does not expose that operation:
  opencode remove"* under a live Remove button. The transport declaration is one
  fact per verb now: OpenCode's install stays its own `plugin` command while its
  removal is PiCode's own write of the config file that names the plugin
  (comments and every other byte preserved), answered with the CLI's fresh list
  and shown as the same removal transcript Pi's own mutations show. A plugin the
  CLI's own config does not name is still refused, by name.
