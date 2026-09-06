# ADR-0080: Dedicated llama.cpp management

- **Status**: accepted
- **Date**: 2026-09-06

## Context

Providers mixes credentials with local model operations. Connection failures
hide their cause and a long page conceals model management. The owner approved
[a four-delivery roadmap](../plans/llama-manager.md), including later service
management, and authorized starting it.

## Decision

llama.cpp gets a dedicated desktop/mobile route at `#/llama`, with Models and
Server sections initially. Providers links to the manager; the old
`#/providers/llama` link and `/llama` command open it. Existing Pi credential
storage remains authoritative. Configuration is distinct from reachability.
Connection failures have safe typed results; operations report their actual
completion and cancellation outcomes. Desktop and mobile own their UI.

Later deliveries add capability-aware jobs, catalog and agent validation.
Service ownership, installation, update recovery and file removal require a
concrete follow-up ADR before implementation; approval of the roadmap includes
that design work. No runtime ownership or file-deletion change is made here.

## Consequences

Model operations become discoverable and connection failures actionable.
Independent UI implementations need acceptance on both apps. Existing router
compatibility remains required; unsupported servers receive an explicit result.
A dedicated surface costs navigation wiring but preserves Providers' focus.

## Alternatives considered

- A larger Providers footer retains mixed credentials and service concerns.
- A modal-only manager makes long operations and deep links harder to follow.
- Installing or supervising the server in the first increment expands ownership
  before recovery and platform behavior have been designed.
