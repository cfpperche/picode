### Removed

- **Old Pi-era Packages and Connectors links.** `#/packages`, mobile
  `#/more/packages`, `#/clis/packages[/<cli>]`, `#/mcps`, `#/more/mcps`,
  `#/integrations/connectors` and `#/clis/connectors` no longer open Pi's
  panes. A CLI's panes live at `#/clis/<cli>/packages` and
  `#/clis/<cli>/connectors`; `#/integrations` now opens Webhooks.

### Fixed

- **Connector sign-in returns to the right pane.** After an MCP server's
  browser sign-in, the success page's return link opens that CLI's
  Connectors pane with the workspace and agent you started from, instead of
  a retired address.
