# Optional connectors

Connector implementations live outside PiCode. These files are portable,
standard MCP configuration, not Go plugins or another package manager.

| Definition | External implementation | Access |
|---|---|---|
| [deepwiki.json](deepwiki.json) | `https://mcp.deepwiki.com/mcp` | Public GitHub repository documentation, no account required |
| [gmail.json](gmail.json) | `@gongrzhe/server-gmail-autoauth-mcp` (community, ISC) | Read, draft and send Gmail; needs a one-time Google sign-in first (see the [Gmail cookbook](https://cfpperche.github.io/picode/guide/mcp-gmail)) |

Open **Agent CLIs → Connectors**, install the MCP adapter through Packages
if needed, then add the service: pick its card in the Marketplace search, or
open **Custom server…** and enter the same fields a definition holds — a
`url`, or a `command` plus an `args` array. The existing adapter owns
execution; native MCP files own configuration.
No binary rebuild or PiCode source edit is needed for another connector.

A definition may carry optional `headers`, `auth`, `bearerToken` for remote
servers and `env` for commands; the custom form takes the same fields. Keep
definitions free of personal credentials; enter a token locally through the
custom form. OAuth works only with servers supported by the installed
adapter's authentication flow.

Install local command dependencies separately, using their maintainer's
instructions. Adding a connector is not a dependency installer or a sandbox:
local commands have the agent's system permissions. Only add connectors you
trust. Removing configuration is not equivalent to revoking credentials at
the service; use Sign out when available and revoke provider tokens
separately. Running agents may need a restart to reload configuration.

The initial connectors validate the existing MCP path. The Gmail definition
shows a command-based local server whose credentials stay outside PiCode; it
is still third-party software — review it before granting mailbox access.
Vendor-specific non-MCP adapters beyond the installed one can be developed
separately when an actual integration needs capabilities beyond this
contract.
