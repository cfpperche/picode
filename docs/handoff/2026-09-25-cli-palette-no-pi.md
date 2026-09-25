# 2026-09-25 — feat/cli-palette-no-pi: the palette no longer defaults to Pi
Shipped (cdc7f480e): fourth of the Pi-default cleanup series, requested by the owner.
Decision table (routes.test.js): no agent → the CLI pane commands (settings,
packages, skills, mcps, connectors, providers, providers-new, providers-custom)
open the catalog #/clis; agent with an empty cli → Pi (the agentIsPi data rule,
not a fallback); agent with a cli → that CLI. Desktop go() in
web/browser/src/lib/routes.js (CLI_PANE_COMMANDS); Pi entries dropped from
ROUTES. App.jsx composer slash go-providers/new and onAgentPage pass the
agent's cli. With no last agent, mobile More search shortcuts open #/clis;
the Pi-only mobile go() is deleted. Both AddProviderDialogs navigate with
cliProvidersHash(cli, …) directly instead of go("providers-custom").
Verified: `make ci-scoped` and `make close` green. Scratch instance with no
agents: palette "CLI settings", "Packages", "Connectors" → #/clis; Add
provider → "+ Custom provider" on Pi → #/clis/pi/providers/custom, dialog closed.
Blind spot: the "agent of another CLI selected" row is covered by unit tests
only, not driven live.
visual-review: PASS (clipal-catalog.png, clipal-custom.png)
Not done (left on purpose): sessionsHash(ws, cli="pi") still defaults to Pi for
session rows without a cli field (a data default); the catalog's first row is
Pi by list order.
Merge: fast-forward ready.
