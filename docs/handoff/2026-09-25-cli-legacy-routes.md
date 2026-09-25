# 2026-09-25 — feat/cli-legacy-routes: retire Pi-era Packages and Connectors addresses
Shipped (1d3fb1bf4): follow-up to feat/clis-settings-tab, owner asked. Retired
#/packages*, #/more/packages*, #/clis/packages[/<cli>], #/mcps, #/more/mcps,
#/integrations/connectors, #/clis/connectors. cliPackagesLocation and
cliConnectorsLocation parse only #/clis/<cli>/packages[/config/<pkg>] and
#/clis/<cli>/connectors; adoptPane, cliLocation's legacy argument, the dead
helpers packagesConfigRoute/packagesConfigHash/integrationSection and the
mobile More sections mcps/packages are gone. #/integrations* opens Webhooks
(desktop + mobile).
Server: the MCP OAuth returnTo was https://host/#/mcps; it is now
connectorsHash(cli, workspaceId, agentId) → #/clis/<cli>/connectors?…
(internal/server/mcp.go, table test TestConnectorsHashIsCanonical).
Verified: `make ci-scoped` and `make close` green. Scratch instance:
#/clis/pi/packages and #/clis/codex/connectors render their panes,
#/integrations renders Webhooks, #/packages and #/mcps land on home.
Blind spot: returnTo not exercised through a real OAuth sign-in.
visual-review: PASS (clileg-integrations, clileg-connectors; overlayAudit ok)
Not done (owner's call, still Pi-defaulting): #/clis/sessions and #/sessions
(ADR-0079 redirect to #/clis/pi/sessions); providers legacy (#/providers,
#/more/providers, #/clis/providers/<cli>); palette go() falls back to Pi with
no agent selected. AgentClis props legacyAgentId/legacyPackageContext now only
feed the setup context (naming only).
Merge: fast-forward ready.
