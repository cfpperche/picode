# DeepWiki connector

An optional MIT Pi package. No Go handlers, frontend code, credentials or
runtime dependencies: `pi.mcp` points to standard MCP configuration and the
external DeepWiki service implements the tools.

Requires an installed `pi-mcp-adapter` with package-manifest support (verified
with 2.32.1). This package is a local example, not published to npm.

In PiCode, open **Packages** and install the absolute path to this directory.
It then appears under **Agent CLIs → Pi → Connectors → Installed → Connector
packages**. Restart the agent to load it. The adapter names the server
`pi-connector-deepwiki__docs`. Installation is not a claim of live access;
the agent's first tool call verifies the remote service.

Use its `read_wiki_structure` or `read_wiki_contents` tools to inspect public
GitHub repositories. `ask_question` sends a question to the external service;
do not include private code or secrets. No GitHub account is needed.

After installing both packages in user scope, `node verify.mjs` is an opt-in
smoke check: it uses the adapter's public configuration loader and MCP client
to read public `golang/go` documentation. It makes no model turn and does not
use owner credentials; it does not certify OAuth or model-driven behavior.

Remove through **Packages** and restart affected agents to stop loading it.
There is no token to revoke. Do not also add the DeepWiki card or a Custom
server… entry for the same service unless you intentionally want two
separately named configurations.

| Concern | Compatibility |
|---|---|
| Install/remove and scopes | Same as Pi packages |
| `pi.mcp` discovery/execution | Provided by pi-mcp-adapter, not native Pi |
| Configuration and credentials | No PiCode database copy |
| Managed runtimes | Pi only; no implied Claude Code/Codex/Grok support |

See [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md)
and [pi-mcp-adapter](https://github.com/nicobailon/pi-mcp-adapter).
