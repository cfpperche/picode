---
description: Connect DeepWiki so an agent can read public GitHub documentation. No account.
---

# DeepWiki

Public GitHub documentation tools through `https://mcp.deepwiki.com/mcp`. No
GitHub account, no PiCode credentials.

- **Where:** **Agent CLIs → Pi → Connectors**, then import `connectors/deepwiki.json`, or install `packages/pi-connector-deepwiki` through [Packages](/guide/packages).
- **Not this:** not private repos and not a place for secrets. `ask_question` sends the question to the external service. How to add any connector: [MCP](/guide/mcp).

The package is a local example, not published to npm. The adapter names the server `pi-connector-deepwiki__docs`. Installation is not a claim of live access; the agent's first tool call verifies the remote service.

## Add it

1. Install `npm:pi-mcp-adapter` if Connectors still says **Open packages** — [MCP](/guide/mcp).
2. Pick **one** path: **Packages** → absolute path to `packages/pi-connector-deepwiki`, or **Import a file…** with `connectors/deepwiki.json`.
3. Restart the agent.

Do not use both paths unless you want two separately named configurations.

## Use it

`read_wiki_structure` and `read_wiki_contents` inspect public GitHub repositories. `ask_question` asks the external service. There is no token to revoke: remove the package or the connector and restart affected agents.
