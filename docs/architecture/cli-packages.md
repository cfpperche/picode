# Native CLI packages (ADR-0102)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Packages is an independent Agent CLIs view in each app. The shared
`cliPackages` module parses canonical/legacy URLs, declares native package
capabilities (Pi initially), and validates workspace/agent identities from the
fleet APIs. Canonical URLs carry `workspaceId`, `agentId` and install `scope`;
an unscoped URL means machine packages. Legacy `#/packages*` and mobile
`#/more/packages*` replace themselves after resolving the available pane
context. Missing or mismatched targets block editing instead of falling back.
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
