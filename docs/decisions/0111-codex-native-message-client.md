# ADR-0111: Codex native message client

- **Status**: accepted
- **Date**: 2026-09-10
- **Boundary**: protocol and security model — Codex resolves its mailbox capability per native shell invocation instead of inheriting an MCP bearer at process startup.

## Context

The owner approved a common native messaging mechanism in ADR-0107 and automatic
workspace enrollment in ADR-0110. Codex 0.154.0 emits the observed SessionStart
hook at its first user turn, including after resume. Restarting it to attach an
MCP endpoint therefore discards the current runtime observation and leaves
onboarding waiting for another prompt. Injecting a setup prompt would violate
the approved enrollment contract.

## Decision

Extend the existing `picode messages` client used by Grok/Hermes to Codex. Its
native shell invocation supplies `CODEX_THREAD_ID`; a present `CODEX_SESSION_ID`
must agree. Discovery resolves one private connection with the same CLI, native
session and terminal owner. Fresh launchers export a discovery directory and
client executable, never a conversation bearer. Existing PiCode terminal tools
can discover the directory under `PICODE_DATA` or `HOME/.picode`. Enrollment
prepares the connection without restarting Codex. The native SessionStart hook
adds command discovery context through its supported hook output, without an
extra model turn or edits to user instructions.

Modern hooks remain authoritative for identity and activity. Once one is observed
in a wrapper incarnation, legacy notify events cannot overwrite it, including
from older wrappers which injected both. The legacy-only branch remains available
and does not inherit the modern-hook flag. This implements the already approved
common-client direction; it does not add an orchestration engine or mailbox.

## Consequences

Enabling an already identified Codex conversation needs no process restart.
Switching conversation, ambiguous identity or a mismatched owner fails before
network access; server authorization still validates consent and revocation.
Local capability authorization is unchanged and is not process attestation.
A brand-new Codex conversation still needs its first native lifecycle event.
Native shell permissions and the installed executable remain prerequisites.
If Codex changes its per-tool identity variables or hook output, discovery fails
closed and the adapter requires revalidation. Native input guards still protect
drafts and permission prompts.

## Alternatives considered

- Keep a startup MCP bearer and restart on enable: reproduced loss of immediate
  readiness on resume, and risks inheriting credentials across conversation changes.
- Send an invisible setup prompt: consumes a model turn and violates ADR-0110.
- Infer identity from the newest session file: ambiguous across concurrent sessions.
- Introduce a Codex app-server or terminal log parser: another runtime boundary
  without a requirement that the existing native client cannot satisfy.
