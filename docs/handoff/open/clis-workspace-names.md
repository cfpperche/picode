# clis-workspace-names

Open items left by feat/cli-ws-name (2026-09-23).

## Debts

- [ ] Mobile panes share the renamed components but were verified only in the
  desktop DOM; capture a phone-width screenshot of Omp Settings and Packages
  bound to a workspace.
- [ ] Pi settings deep link (`#/clis/<cli>/settings?workspaceId=…`) drops
  `workspaceId` (pre-existing parser behavior in `cliSettingsLocation` — the
  context arrives from the sidebar selection), so a deep link without a
  selected agent still shows the generic label. Decide whether the parser
  should carry the workspace like the models pane does.
