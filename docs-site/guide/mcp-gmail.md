---
description: Connect Gmail to a Pi agent. Credentials stay outside PiCode.
---

# Gmail

Read, draft and send Gmail from a Pi agent. The catalog uses the community
server `@gongrzhe/server-gmail-autoauth-mcp`. There is no official Gmail MCP
from Google.

- **Where:** **Agent CLIs → Pi → Connectors**, then the **Gmail** card — or import `connectors/gmail.json`, or install `packages/pi-connector-gmail` through [Packages](/guide/packages).
- **Not this:** credentials are **not** stored in PiCode. They live in `~/.gmail-mcp/`. Review the community server before you grant a mailbox. How to add any connector: [MCP](/guide/mcp).

## 1. One-time Google sign-in

1. In [Google Cloud Console](https://console.cloud.google.com/), create a project, enable the **Gmail API**, and create an **OAuth client ID** (application type *Desktop app*). Download the client secret JSON.
2. From a terminal:

```bash
mkdir -p ~/.gmail-mcp
mv <downloaded-client-secret>.json ~/.gmail-mcp/gcp-oauth.keys.json
npx -y @gongrzhe/server-gmail-autoauth-mcp auth
```

A browser opens for Google consent. Tokens land in `~/.gmail-mcp/`. Skip this and the connector installs, but every tool call fails until the sign-in exists.

## 2. Add the connector

Pick **one** path:

- the **Gmail** card in the catalog
- **Import a file…** with `connectors/gmail.json`
- **Packages** → install the absolute path to `packages/pi-connector-gmail`

Do not combine paths unless you want two separately named configurations. Restart the agent so the adapter loads the tools.

## 3. Revoke

Remove the connector (or the package) and restart affected agents. That stops tools. It does **not** revoke Google tokens. Also delete `~/.gmail-mcp/` and revoke the application at your [Google account permissions](https://myaccount.google.com/permissions).
