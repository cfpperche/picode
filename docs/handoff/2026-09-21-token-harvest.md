# 2026-09-21 — token-harvest: vendor renewals flow back into the vault

Shipped: `Store.Harvest(provider, live)` — on a roster read, a guest CLI's
own OAuth file that holds a renewed token pair (same refresh token, newer
access/expiry) is merged into the vault row it belongs to, in place, before
the rows are read. Matching is by the stable refresh token and never
guesses: access-only logins (Muse) and strangers are left alone, and the
CLI's own file is only ever read (writing there is Use, ADR-0166). Covers
every CLI whose login carries a refresh token; Muse stays on the re-import
path.

Verified: `TestHarvestRenewsTheMatchedRow` (vault: renewal merged, stranger
and access-only refused, no stray writes) and
`TestCredentialRosterHarvestsRenewal` (server: roster read harvests, the
saved row holds the renewed access, the row stays matched). Full affected
suite green. Blind spot: real vendor rotation was simulated in fixtures, not
observed against live Anthropic/xAI renewals.

visual-review: n/a (no UI change; the pane's in-use and hint rows now simply
stop going stale)
Not done: Muse (no refresh token in its file — nothing stable to match);
logged as accepted in the token-harvest handoff.

## Next up

- none in this slice
