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
`TestNativeCodexAndOpenCodeComposers`. Native results follow.


## Native acceptance

Scratch Linux instance `localhost:8472`, six real tmux TUIs. OpenCode used
`zai-coding-plan/glm-5.3-flash` with its sidebar visible; Pi used `xai/grok-4.6`
with the native adapter package loaded. No Pi product-code repair was required.

| Exchange | Diagnostic | Result |
| --- | --- | --- |
| Pi → OpenCode | `check_WAO4RDWD7374MGXULWIPCB2KKN` | Message, reply and both ACKs passed |
| Claude Code → Hermes | `check_DFRS6JQOC5PFXJGUG2TSWICE6Q` | Message, reply and both ACKs passed |
| Grok → Codex | `check_VMOPYXSHT3OFEF5VCPBHAAHMNB` | Message, reply and both ACKs passed |
| OpenCode → Pi after restart | `check_6Y4VFSQM7FB57AY37RFMCPTQSS` | Message, reply and both ACKs passed |

All four diagnostics passed without expired or uncertain attempts. Read-only
receipt verification checked sender/recipient, `reply_to` and both `acked_at`
values independently of the displayed diagnostic status. These paired exchanges
cover all six CLIs; they are not an exhaustive every-pair matrix.

Abrupt scratch-daemon death preserved all six native process/session identities.
Startup took 0.204 seconds; all six were connected, confirmed and Idle another
5.645 seconds later. OpenCode and Codex unsent drafts were byte-identical after
recovery. The pending diagnostic did not overwrite or submit either draft; the
final exchange completed after the operator cleared these test-owned drafts.

Evidence: `var/qa/communication-native-finish-20260912/native-validation.json`,
`restart-1789232903.json` and `draft-proof.json`. Screenshots were read by a
visual-review subagent: desktop/mobile connected, activity and empty states,
native selector and overlay audit passed. No frontend code changed; simulated
failure/repair states and physical devices were not revalidated.

Existing non-Linux recovery, onboarding partials and the PTY check-to-write race
remain tracked in `docs/handoff/open/communication.md`. Correct provider/package
configuration and this parser fix close the prior Pi/OpenCode transport blockers.
