# Native communication acceptance — September 12

## Scope

Validate six real TUIs with current native credentials, then repair the observed
OpenCode composer refusal without changing mailbox, identity, consent or retry rules.
OpenCode uses `zai-coding-plan/glm-5.3-flash`; Pi uses `xai/grok-4.6`
with `pi-mcp-adapter` actually loaded. The preceding scratch copied Pi's adapter
installation but omitted its package configuration; that was not a product failure.

## Decision table

| Condition | Action / evidence |
| --- | --- |
| Empty OpenCode editor beside native sidebar and wrapped cwd | Accept captured native frame |
| Exact pasted pointer inside measured editor | Permit final native submission |
| Draft at, above or below cursor | Refuse |
| Copy mode, unknown footer, malformed escape or damaged border/gutter | Refuse |
| Pointer reaches or exceeds measured editor boundary | Refuse before claiming/pasting |
| Post-paste concurrent edit | Refuse submission; never retry automatically |
| Pi has installed but unloaded adapter | No transport acceptance; load native package configuration |
| Correct native providers and loaded adapters | Require message, reply and both ACKs |
| Daemon dies with native processes preserved | Recover exact identities; retain drafts |

Parser regression coverage: `TestPeerOpenCodeSidebarAndWrappedFooter`,
`TestNativeCodexAndOpenCodeComposers`. Native results are recorded at close.
