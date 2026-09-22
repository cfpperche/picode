# ADR-0178: Guided vendor OAuth from the Providers pane (supersedes ADR-0168's refusal clause)

- **Status**: accepted (owner approved 2026-09-22, in session)
- **Date**: 2026-09-22
- **Boundary**: security model — PiCode drives a vendor's browser OAuth with the agent CLI's own public client id and receives the tokens; persistence — guest credentials land in the vault and travel to the CLI by env.

## Context

The owner asked when omp gets pi's login UX: the provider's authorize page
opens, the user approves, the redirect returns, and the subscription is
stored and used. ADR-0168 answered "PiCode performs no vendor OAuth and
presents no other product's client id" — but the codebase outgrew that
clause before this ADR existed: `internal/oauth` (the Add provider dialog's
engine) already runs PKCE + loopback callbacks and device flows for
anthropic, openai-codex, github-copilot, kimi-coding and xai **with those
products' own public client ids**. The refusal clause described an
architecture the house had already shipped past; re-measuring it (AGENTS.md
§6) showed the load-bearing concerns are two, and both are costs to record,
not impossibilities: (a) the token-exchange handshakes are undocumented and
versioned — PiCode becomes a shadow client that drifts when vendors change
their build-gated headers; (b) a subscription token minted for one product
and spent from another is ToS-gray — the flag risk lands on the user's
account, not on PiCode.

For omp the feature was also blocked on storage: its credentials live in an
SQLite database PiCode deliberately does not read (ADR-0165), so even a
completed login had nowhere to land that the roster could show.

## Decision

The Add provider dialog's Guided sign-in, for a guest provider the OAuth
engine supports (`oauth.Supports`: anthropic, openai-codex, github-copilot,
kimi-coding, xai), starts the browser flow from the pane: PiCode opens the
authorize page in a tab, runs the loopback/device callback, and the minted
credential lands in the **vault** as the provider's row (kind oauth) —
`oauth.StartSink` with a vault sink replaces the auth.json write for that
path. The credential reaches omp through the env names its own declaration
declares (`ANTHROPIC_OAUTH_TOKEN`, …), exported when PiCode spawns the CLI's
terminal — env is the lowest channel in omp's own resolution chain, so a
native `/login` outranks an injected token and a user-configured env entry
outranks both. Providers the engine does not support keep the terminal strip
(ADR-0168's flow, unchanged). PiCode performs no refresh: when a token
expires the user re-runs Guided sign-in (the vault's fingerprint for oauth
rows is per provider, so the re-login replaces the account instead of
stacking).

## Consequences

One click signs omp into a subscription, from the GUI, with nothing
upstream — and the vault becomes the single place the credential lives.
The costs: PiCode now owns shadow-client drift for five vendors (a vendor
that changes its exchange breaks the pane until PiCode follows — visible
immediately as a failed Verify, not silently); the token spends inside the
spawned terminal's whole process tree (same exposure as the user exporting
it themselves); and a native `/login` plus an injected env token can
coexist, with omp preferring its own — the pane cannot tell which one won,
so the account row says "vault" and the CLI's behavior is the truth.
Scope: omp only. Other guests keep the ADR-0168 flow; extending the bridge
to them is a later, separate call.

## Alternatives considered

- **Keep the terminal strip for omp** (ADR-0168 as written): the flow works
  but lives in a TUI the pane can only hint at, and the credential stays
  invisible. Refused by the owner.
- **Read `agent.db` to import native logins**: another tool's credential
  store, encrypted and versioned — an ADR of its own, and still does not
  drive the browser flow. Recorded in `docs/handoff/open/
  omp-oauth-parity.md`; not chosen here.
- **PiCode requests its own registered OAuth clients per vendor**: the
  honest long-term shape, but it needs vendor registrations PiCode does not
  have today. The client-id reuse is the pragmatic step this ADR takes,
  with the drift cost said out loud.
