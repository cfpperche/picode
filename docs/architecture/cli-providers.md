# Native CLI providers (ADR-0103)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Providers is a pane of the selected CLI at `#/clis/<cli>/providers`; `/new`
opens Add provider. A shared route/capability helper names Pi explicitly and
redirects legacy desktop/mobile links (`#/clis/providers*`, `#/providers*`). Unsupported identities and explicit
agent/workspace scopes block editing; accounts still belong to the machine.
The editor loads its catalog independently of terminal inventory, retains
successful rows and drafts during refresh failures and offers retry. Successful
refreshes also update the app's model catalog. OAuth returns to the same app's
canonical list; closed/unmounted editors ignore late login completions.
Native provider APIs, the active Pi auth slot, extra-account vault and quota
semantics remain unchanged. The llama.cpp manager keeps its separate route.

The roster renders **one row per account**, grouped under the provider that
owns it, as a six-column grid (`Provider · Account · Identity · Usage ·
7d spend · actions`). Its geometry lives in
`web/shared/styles/providers.css`, imported last by both apps' `index.css`:
the two components stay per-app (ADR-0072), the roster's layout does not.
The column template drops the identity column below a 1000 px container and
folds the cells into a stacked card below 840 px, so one markup serves the
desktop window, a squeezed window and the phone.
