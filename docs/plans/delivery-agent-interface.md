# Common delivery interface — D1a

Status: implemented under accepted ADR-0171, following the owner's 2026-09-21
approval. Public usage: [delivery requests](../../docs-site/guide/delivery.md).
Architecture: [delivery](../architecture/delivery.md).

## How a delivery is created

A branch is an observed candidate, not automatically a registered delivery.
An agent invokes `picode delivery register` with title, source branch, full source
revision and target. PiCode derives the repository/launch identity, checks local
Git and returns a durable ID/version. The agent may ask for review; subsequent
updates clear that request. CLI and MCP share the same server/store contract.

The design adapts the benchmark's change-centered model and explicit state/evidence
separation. Registration is portable across shell-capable clients; verification
remains project/provider-specific. The source comparison is currently the only
observed evidence. No agent declaration can mark checks, integration or production
healthy. The UI and queue requests are future slices in the parent plan.

## Compatibility evidence

| Path | Implemented / tested | Limit |
|---|---|---|
| Pi, Claude Code, Codex, Grok, Hermes, OpenCode, Muse, Antigravity, Omp | Server test exercises nine registered catalog principals through the same action contract | No vendor executable, credentials or native conversation lifecycle in the fixture |
| Generic PiCode terminal | Inherited terminal identity and launch repository | Outside launches without identity are refused |
| Shell command | CLI → HTTP handler → SQLite → events, registration/retry/review/show/list/conflict | Requires the updated binary/daemon |
| MCP delivery family | Common adapter and server family; identity replacement, errors, no automatic retry | Existing automatic launch injection supports only Claude Code/Codex/OpenCode; explicit tool selections stay explicit; no new picker option |
| Observation and execution | Source revision comparison; unknown checks/integration/publication; queue capabilities false | ADR-0170 observation and D3/D4 authority are not implemented |

Launch attribution is not native-session attestation. A resume or new conversation
inside the same launch retains the principal. No messages connection is created.

## Decision table and automated evidence

| Conditions | Result | Test |
|---|---|---|
| Known launch + local branch/revision/target | Register version 1, one event | NativePrincipals; CLIThroughDaemon |
| Bound agent+terminal / bound terminal alone | Same agent principal, mutation ownership and retry receipt | BoundPrincipal |
| No identity / removed or unknown identity / disagreeing agent+terminal | Refuse | RefusalTable; ScopeAndSourceChanges; CLIUsageAndIdentity |
| No Git repository / missing target / invalid source ref / wrong revision | Refuse without a receipt | RefusalTable; ScopeAndSourceChanges |
| Same repository, other principal | Read allowed; mutation refused | ScopeAndSourceChanges; MutationDecisionTable |
| Other repository | Record unavailable | ScopeAndSourceChanges; MutationDecisionTable |
| Same request key and identical content | Original receipt, no extra event; no state rollback | MutationDecisionTable; NativePrincipals; CLIThroughDaemon |
| Same key, changed content | Conflict | MutationDecisionTable |
| Stale version / concurrent writers | Conflict; exactly one writer advances | MutationDecisionTable; ConcurrencyAndRollback |
| Review requested, then update | Review clears, version advances | MutationDecisionTable |
| Source moves before review / after registration | Refuse new review / show changed | ScopeAndSourceChanges; NativePrincipals |
| Event insert fails | Row and receipt roll back | ConcurrencyAndRollback |
| Store reopens | Record and original retry survive; no second event | Restart |
| More than 100 declarations | Stable sequence pagination | RegistrationRetryAndPagination |
| Record/receipt cap reached | Refuse new mutation; preserve exact retry | Capacity |
| Caller supplies forged tool identity | Replace with inherited identity | MCPContract |
| Transport failure or invalid response | Error; no automatic replay | MCPContract |
| Integration/deploy request | Explicit unavailable response; no execution | RefusalTable; CLIThroughDaemon |

Test names use the `TestDelivery` prefix (see Go test files). Git checks and the
store commit are not a joint Git transaction. A successful request describes the
recorded revision; it never guarantees that a concurrently edited branch still
points there. UI consumers must show current evidence separately.

## Next slices

D1b connects authoritative checks/integration evidence and the desktop/mobile
screen. D2 observes publication. D3/D4 introduce separately approved queue and
execution policies using the same interface vocabulary. They must recheck exact
revision, evidence and authorization before acting. This first slice reserves no
execution capability and does not change land/deploy scripts.
