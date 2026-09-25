### Changed

- **Agent CLIs: the tmux guard and browser hand-off switches have their own
  Settings tab.** They apply to every PiCode terminal, not to the CLI you
  pick, so they left the top of the CLI catalog for a third tab beside CLIs
  and Messages (`#/clis/settings`), on desktop and mobile.

### Removed

- **Old Pi-era settings links.** `#/settings`, mobile `#/more/settings`,
  `#/clis/settings/<cli>` and the `?tab=keys` sub-tab link no longer redirect
  to Pi's settings. A CLI's settings live at `#/clis/<cli>/settings`.
