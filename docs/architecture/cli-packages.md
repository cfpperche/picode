# Native CLI packages (ADR-0102)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Packages is a pane of the selected CLI (`#/clis/<cli>/packages`) in each app.
The shared `cliPackages` module parses canonical/legacy URLs, declares native
package capabilities (Pi initially), and validates workspace/agent identities
from the fleet APIs. Canonical URLs carry `workspaceId`, `agentId` and install
`scope`; an unscoped URL means machine packages. Opening the pane from Agent
CLIs while a sidebar agent is selected writes that identity onto the hash so
the workspace/agent radios appear; an unscoped URL with nobody selected stays
machine-only. Legacy `#/packages*`,
`#/clis/packages*` and mobile `#/more/packages*` rewrite onto the pane. Missing or mismatched targets block editing instead of falling back.
Context refresh failures retain the mounted package view and its draft, with
writes blocked until retry succeeds. Each mutation revalidates its URL target,
including after confirmation. A changed workspace path or agent work path
retains the draft but requires a confirmed reload before editing. Roles saves
retain both layers and preserve an unsaved draft in the other layer.
Package reads show failures with retry rather
than reporting an empty installation. This view does not load terminal inventory
or CLI lifecycle jobs. Existing Pi package APIs, commands and persistence stay
unchanged; the desktop roles editor remains native-file-backed. Mobile config
links offer the desktop layout with the same URL until a mobile editor exists.

Configuration (ADR-0099, ADR-0119): a package is configurable in this view
when a config descriptor resolves for it. Descriptors resolve in order — the
user's own description (`DataDir/package-configs/<id>.json`), then
`picode.config` in the extension's `package.json`, then PiCode's catalog
(`internal/pipkg/configdescriptor.go`). The generic engine writes the declared
file (agent or workspace scope, per descriptor) with the descriptor's typed
fields — enum, boolean (tri-state: untouched = unset), number with min/max,
string, secret — preserving unknown keys and refusing to replace an
unparseable file without explicit force. The config page names the source of
the descriptor (user, catalog, manifest). A user description is created with
**Describe config…** on an undescribed card and edited or deleted from the
config page; deleting it never touches the described file. pi-roles keeps its
bespoke two-layer editor. No descriptor resolves, no Configure button.
