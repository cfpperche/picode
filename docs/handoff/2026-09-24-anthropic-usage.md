# 2026-09-24 — anthropic-usage

The dashboard/Providers showed "Sign in again" for Anthropic although Claude
Code was signed in. Measured: both Anthropic vault rows answered 401 (the
in-use one since 09-14); Claude Code's file answered 200 (5h 21%, 7d 86%).
Harvest matches by refresh token, so a new sign-in or a rotated token never
reached the row, which still showed "in use" by id.

- `credentials.Mirror` + `mirrorCLILogins` (roster read, and `usage.Client.Sync`
  before each fetch); `usage.Client.HeldBy` → never refresh a CLI-held token.
- ADR-0166 amendment; `docs/architecture/credentials.md`.

## Debts

- pi's own use of an Anthropic vault row still refreshes it; if the same account is also Claude Code's, the two can rotate each other out.
