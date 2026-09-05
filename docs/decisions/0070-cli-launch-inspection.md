# ADR-0070: Inspect CLI launches and reuse terminal profiles

- **Status:** accepted (owner approved the v2 proposal, 2026-09-04)
- **Date:** 2026-09-04

## Context

ADR-0069 exposes editable overrides but hides automatic executable resolution
and injected launch settings. Users need to inspect the next launch, distinguish
setup checks from actual activity, and reuse settings without creating agents.

## Decision

Extend the terminal-only manager with a backend launch plan shared by preview
and execution. Preview is read-only: it neither runs a CLI nor writes files.
Generated paths use a visibly symbolic next-generation directory. Conditional
adapter branches (Codex hook capability and Pi maintenance commands) remain
explicit; a preview never invents their runtime outcome. Native CLI-owned model,
permission, authentication and session settings remain outside this plan.

Default resolution stays automatic until the owner saves an executable
override. Restore defaults clears launch customizations but preserves the
activity-reporting choice. The UI starts with a compact read-only summary;
editing has Save/Discard and protects dirty drafts during navigation.

Named profiles persist complete launch configurations in SQLite. Applying one
copies its settings into terminal overrides, including explicit empty arrays
and automatic executable selection. Editing or removing a profile never changes
an existing terminal. Profiles have no runtime, protocol, credentials, workspace
ownership or Agent record. Configuration and invalidation events commit together.

Setup checks record executable identity, version and reporter prerequisites.
Repair is a separate explicit action confined to PiCode-owned integration files.
Activity evidence is only the current daemon's accepted lifecycle reports, not
an installation claim. Binary/configuration drift invalidates a saved check.
Launch failures retain a bounded, redacted last-attempt record for repair/retry.

Restart prepares the next immutable generation before ending the old terminal.
Preparation failure leaves the running process intact. OS spawn failure after
stop remains possible and is reported, never disguised as rollback. Workspace
removal serializes its terminal operations and removes private launch artifacts;
native CLI data is untouched.

## Decision table

| Conditions | Action / expected evidence |
|---|---|
| Automatic executable, preview/read/save unrelated field | Resolve dynamically; never pin the discovered path |
| Restore defaults, reporting off | Clear overrides, keep reporting off |
| Preview with reporting off | No injected arguments or overlay files |
| Preview with reporting on | Shared adapter arguments/files; conditional branches labeled |
| Preview with secrets | Environment values omitted; known secret arguments masked |
| Missing executable / invalid folder | Preview names a missing executable; launch preflight refuses either problem safely |
| Dirty editor, navigation/cancel | Confirm discard; cancel retains the draft |
| Profile create/update/delete | Validate, persist and announce; no process changes |
| Profile copied, profile later changes/disappears | Terminal settings remain independent |
| Live terminal inherits changed defaults | Pending comparison; no automatic restart |
| Terminal explicitly overrides a changed field | Preserve override; no false pending change |
| Config/executable changes after check | Mark prior diagnostic stale; no implicit execution |
| Integration files missing | Offer explicit repair, do not edit native config |
| No lifecycle event / presence only | Activity remains unverified |
| Accepted event / stale run event | Record current evidence / reject stale event |
| Restart preparation fails | Retain exact old process and record failure |
| Start fails after record creation | Retain configuration and redacted failure for retry |
| Workspace removed | Stop and clean exact owned terminals, retain unrelated/native files |

Coverage: `TestCLIPreviewDecisionTable` and `TestCLIAdapterPreviewMatchesExecution`
cover resolution, masking, reporting branches and actual argument vectors.
`cliLaunch.test.js` covers restore, profile copying, pinned overrides and binary
comparison; `hashGuard.test.js` plus desktop/mobile browser checks cover dirty
navigation. `TestCLIProfilesCopyPersistAndRollback` and
`TestCLIProfileRoutesAndAffectedLaunches` cover persistence, event atomicity,
profile independence and affected/pending terminals. Setup/repair staleness is
covered by `TestCLISetupCheckAndRepairAreSeparate`; failure/retry and cleanup by
`TestCLICreateFailureRetainsLaunchForRetry` and
`TestCLIRestartPreparationFailureAndWorkspaceCleanup`. Existing
`TestCLITerminalLifecycleDecisionTable`, `TestCLITerminalRejectsInvalidRequests`
and the runtime run-ID/projection tests cover unsafe restarts, invalid folders,
presence-only and stale-event conditions. Vendor working/approval/settled
acceptance still requires separate real-version evidence.

## References and adaptation

- [VS Code settings](https://code.visualstudio.com/docs/configure/settings):
  distinguish defaults from explicit changes; reset without freezing detection.
- [VS Code terminal profiles](https://code.visualstudio.com/docs/terminal/profiles):
  reusable executable/argument/environment presets, adapted as copied settings.
- [Cursor benchmark](../benchmark-cursor.md): compact summary, progressive
  disclosure, one primary launch action and explicit pending state.

## Consequences

More launch state is inspectable without a second resolver in the browser.
Profiles duplicate settings intentionally instead of introducing cascading
live inheritance. Secrets remain possible in the owner-only configuration
editor; the manager is not a credential vault. Diagnostics cannot certify
every vendor event, and mutable native configuration still affects execution.

## Alternatives considered

- Reconstruct commands in React: rejected because execution and preview drift.
- Linked profiles: deferred to avoid hidden changes across existing terminals.
- Promote other CLIs to agents: outside the approved terminal-management scope.
