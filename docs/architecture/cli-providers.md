# Native CLI providers (ADR-0103)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Providers uses each app's `AgentClisFrame` at `#/clis/providers/pi`; `/new`
opens Add provider. A shared route/capability helper names Pi explicitly and
redirects legacy desktop/mobile links. Unsupported identities and explicit
agent/workspace scopes block editing; accounts still belong to the machine.
The editor loads its catalog independently of terminal inventory, retains
successful rows and drafts during refresh failures and offers retry. Successful
refreshes also update the app's model catalog. OAuth returns to the same app's
canonical list; closed/unmounted editors ignore late login completions.
Native provider APIs, the active Pi auth slot, extra-account vault and quota
semantics remain unchanged. The llama.cpp manager keeps its separate route.
