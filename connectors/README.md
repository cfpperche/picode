# Optional connectors

Connector implementations live outside PiCode. These files are portable,
standard MCP configuration, not Go plugins or another package manager.

| Definition | External implementation | Access |
|---|---|---|
| [deepwiki.json](deepwiki.json) | `https://mcp.deepwiki.com/mcp` | Public GitHub repository documentation, no account required |

Open **Integrations → Connectors**, install the MCP adapter through Packages
if needed, then **Import a connector definition**. Choose a JSON file, review
the destination or local command and scope, and confirm **Add connector**.
The existing adapter owns execution; native MCP files own configuration.
No binary rebuild or PiCode source edit is needed for another connector.

Author one `mcpServers` entry with either `url` or `command` plus an `args`
array. Optional `headers`, `auth`, `bearerToken` apply to remote servers;
`env` applies to commands. The importer refuses unsupported options instead
of silently discarding them. Keep definitions free of personal credentials;
use the custom form to enter a token locally. OAuth works only with servers
supported by the installed adapter's authentication flow.

Install local command dependencies separately, using their maintainer's
instructions. Importing a definition is not a dependency installer or a
sandbox: local commands have the agent's system permissions. Only import
trusted definitions. Removing configuration is not equivalent to revoking
credentials at the service; use Sign out when available and revoke provider
tokens separately. Running agents may need a restart to reload configuration.

The initial connector validates the existing MCP path. Vendor-specific
non-MCP adapters and a marketplace are not required to add another MCP
service; they can be developed separately when an actual integration needs
capabilities beyond this contract.
