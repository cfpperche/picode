# Gmail connector

An optional MIT Pi package wrapping the community
[`@gongrzhe/server-gmail-autoauth-mcp`](https://github.com/GongRzhe/server-gmail-autoauth-mcp)
server (ISC). No Go handlers, frontend code or runtime dependencies ship here:
`pi.mcp` points to standard MCP configuration and the external npm package
implements the tools. Review that project before trusting it with a mailbox.

## Before first use — one-time Google sign-in

The server keeps credentials **outside** PiCode and outside this package, in
`~/.gmail-mcp/`. Prepare them once from any terminal:

1. In [Google Cloud Console](https://console.cloud.google.com/): create a
   project, enable the **Gmail API**, and create an **OAuth client ID**
   (application type *Desktop app*). Download the client secret JSON.
2. `mkdir -p ~/.gmail-mcp && mv <downloaded>.json ~/.gmail-mcp/gcp-oauth.keys.json`
3. `npx -y @gongrzhe/server-gmail-autoauth-mcp auth` — a browser opens for
   Google consent; tokens are stored in `~/.gmail-mcp/credentials.json`.

Skip this and the connector installs but every tool call fails until the
sign-in exists.

## Install

In PiCode, open **Packages** and install the absolute path to this directory.
It appears under **Agent CLIs → Pi → Connectors → Installed → Connector
packages** as `pi-connector-gmail`. Restart the agent to load its tools
(`send_email`, `draft_email`, `read_email`, `search_emails` and related
names belong to the external server; PiCode does not vouch for the exact
list).

Alternatively, add the same command through the **Gmail** card or the
**Custom server…** form. Do not combine paths unless you intentionally want
two separately named configurations.

## Removal and revocation

Remove through **Packages** (or delete the server entry) and restart affected
agents. That stops tool access but does **not** revoke Google tokens: also
remove `~/.gmail-mcp/` and revoke the application at your
[Google account permissions page](https://myaccount.google.com/permissions).

| Concern | Compatibility |
|---|---|
| Install/remove and scopes | Same as Pi packages |
| `pi.mcp` discovery/execution | Provided by pi-mcp-adapter, not native Pi |
| Credentials | `~/.gmail-mcp/` only; nothing copied into PiCode |
| Scope of trust | The agent can read, draft and send mail — grant deliberately |

See [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md)
and [pi-mcp-adapter](https://github.com/nicobailon/pi-mcp-adapter).
