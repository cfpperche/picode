# ADR-0069: Configure and manage coding CLI terminals

- **Status:** accepted (owner approved the revised terminal-only plan, 2026-09-04)
- **Date:** 2026-09-04

## Context

The owner is moving PiCode toward a multi-CLI ADE. Managed agents remain Pi.
Other CLIs need controlled terminal launches and observation before a separate
project evaluates protocols, sessions, packages, permissions and orchestration.
The existing Terminal status preferences configure session-local wrappers but
do not manage launch settings or distinguish configuration from observed state.

## Decision

Extend ADRs 0017, 0056 and 0062 with an Agent CLIs surface at `#/clis`.
Preferences no longer owns Terminal status; its old address redirects here.
The surface has a CLI catalog and a terminal inventory. Desktop and mobile use
the same component, terminal records, tmux sessions and event feed.

Each CLI has launch defaults: executable, argument array, environment, PATH
entries and integration enabled. Each configured terminal has optional field
overrides. Resolve environment then CLI defaults then terminal overrides;
PiCode correlation variables are reserved. PATH entries precede the inherited
PATH. Arguments are individual values, never an evaluated shell expression.
The configured executable is resolved outside PiCode wrappers.

Launches use the installed CLI, existing injection adapters and a private,
immutable launch directory. Changes apply on the next start. The last applied
launch stores a configuration fingerprint and redacted diagnostics; it never
claims the CLI is still present. A CLI exit leaves an interactive shell.
Manual invocations in ordinary PiCode terminals retain wrapper instrumentation.
CLI defaults apply to launches made by the central manager.

SQLite owns defaults and terminal launch settings; migration imports the old
enabled flags once. Store writes and invalidation events commit together.
Presence and activity remain ephemeral, correlated to terminal/run IDs.
Restarting a terminal starts another process, not an automatic conversation
resume. Stop retains its record; Remove removes it and its launch settings.
Closing a browser tab only detaches. Active stop/restart/remove require a
confirmation naming the terminal. Mutations on one terminal are serialized.

The adapters keep user-owned configuration authoritative. PiCode writes its
own launch files and supplies invocation arguments/environment. It does not
implement other CLIs' authentication, package managers or agent protocols.

### Decision table and acceptance coverage

| Conditions | Action / observable result |
|---|---|
| Legacy flag exists, no stored CLI default | Import the flag once; preserve the legacy file |
| Stored default already exists | Stored value wins over legacy flags |
| Executable absent or directory invalid | Refuse start with a repair action; keep saved configuration |
| Invalid/reserved environment or malformed arguments | Refuse configuration before mutation |
| Inherited fields plus terminal overrides | Overrides win; environment merges; explicit empty arguments clear defaults |
| Live terminal, Open/Start | Attach to the existing process; no second launch |
| Stopped terminal, Start | Resolve current settings and launch once |
| Live terminal, settings saved | Keep process; show pending settings when effective configuration differs |
| Live terminal, unconfirmed Stop/Restart/Remove | Refuse; leave process and record intact |
| Confirmed Stop | End the terminal, retain record and launch configuration |
| Confirmed Restart | Validate next launch first, end old terminal, launch new process |
| Remove | End terminal and remove its saved launch configuration, retaining native CLI data |
| Integration enabled but no activity | Show configured integration and no observed signal separately |
| CLI exits or a different CLI starts | Show observed presence, never infer it from saved launch settings |
| Old run event arrives | Ignore when it does not match the active run |
| Daemon/browser reconnects | Reconcile terminal state; never automatically restart work |
| Launch settings contain environment values | Exclude values from feed events and mask effective diagnostics |

Coverage: `TestCLIConfigImportAndAtomicEvents` and
`TestTerminalLaunchPersistenceAndCascade` cover migration, ownership and store
events. `TestResolveLaunchDecisionTable`, `TestValidateLaunchDecisionTable`,
`TestLaunchDiagnosticsRedactValues` and `cliLaunch.test.js` cover resolution,
validation and diagnostics. `TestCLITerminalLifecycleDecisionTable` and
`TestCLITerminalRejectsInvalidRequests` exercise real tmux, concurrent Start,
pending settings, confirmation, invalid restart, stop/open restoration and
exact removal. Existing `term_runtime_test.go`, `term_wiring_test.go` and
`feedReducers.test.js` cover lease identity, stale activity, injection and
reconciliation without invented state. Visual and installed-CLI acceptance
evidence is recorded in the handoff.

## References and adaptation

- [Vibe Kanban agent configuration](https://www.vibekanban.com/docs/settings/agent-configurations):
  tool selection and dedicated configuration, adapted to terminal launches.
- [Zed agent settings](https://zed.dev/docs/ai/agent-settings): separate CLI
  integration from model providers.
- [Paseo provider contracts](https://github.com/getpaseo/paseo/blob/main/docs/providers.md):
  availability and explicit capabilities; diagnostics do not create conversations.
- [Cursor benchmark](../benchmark-cursor.md): compact controls, keyboard access,
  visible pending state, progressive disclosure.

## Consequences

The existing Pi Agent entity, RPC runtime, Inbox, packages and automations keep
their present scope. Adding another terminal CLI needs a catalog entry and a
launch adapter. Promoting it to a managed agent is a separate decision.
CLI version drift and service/shell environment differences require real
acceptance evidence. Diagnostics report what was checked, not a generic promise
that every lifecycle event is supported.
