# ADR-0013: Multiple accounts per provider (PiCode vault)

- **Status**: accepted
- **Date**: 2026-08-25

## Context

pi's `auth.json` is one credential per provider id (`CredentialStore`:
"one credential per provider"). `/login` overwrites. Concurrent agents
cannot use two Claudes at once.

Users still need a work account and a personal account. Waiting on pi
for native multi-account.

## Decision

PiCode keeps extra logins in `~/.picode/accounts.json` (0600).
`auth.json` always holds the **active** slot — what pi reads.

- **Add account** appends to the vault and makes it active (writes `auth.json`).
- **Use** copies that cred into `auth.json`. pi reloads on next request
  (file revision).
- **Sign out** removes one vault row; if it was active, another is promoted
  or the `auth.json` key is deleted.

GET APIs never return secret material.

## Consequences

- Two running agents still share whatever is in `auth.json` right now.
- TUI `/login` is imported on the next catalog load (fingerprint match or new row).
- If pi later grows multi-account, this vault can fold into it.
