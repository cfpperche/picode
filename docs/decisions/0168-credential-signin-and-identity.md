# ADR-0168: The guided sign-in runs the CLI's own login, and a vault row is named by the store or by the person

- **Status**: accepted
- **Date**: 2026-09-20
- **Boundary**: process (who performs a vendor's OAuth) and persistence (what names
  a vault row) — amends ADR-0165 and ADR-0166

## Context

ADR-0165 made the vault hold many accounts per provider; ADR-0166 made "Use"
mean writing one CLI's own credential file. Both left two questions open, and
the owner hit each of them in the same afternoon:

1. **There was no sign-in in PiCode.** A person who was not already logged in
   had to leave the app, run `claude`, `grok login` or `codex login` in a
   terminal, come back, and click "Import into the vault". The pi pane the
   owner uses daily starts that login from the GUI; ours could only read what
   already existed.
2. **A second subscription of a nameless store replaced the first.** The pi
   model gives one row per provider when the credential's file carries nothing
   that names the account: Claude Code's `.credentials.json` has no email and
   no account id, so two different Claude subscriptions collapsed into one row
   that was overwritten on each import (ADR-0166's Consequences recorded this
   deliberately). Muse's file is the same shape — and worse: our declaration
   read only `providers.meta.api_key`, so an account login
   (`providers.meta.mechanism == "oauth"` with an `access_token`, which Muse's
   own launcher writes and requires) was invisible to the pane entirely. The
   owner's correction: "you can't infer a CLI's login from the file that
   happens to be on my machine."

The vendor's own identity endpoints were the obvious fix for (2), and they are
a trap. `https://api.anthropic.com/api/oauth/profile` answers the account email
for an OAuth token — but using it as the row key means the roster must make the
same call to match a live file to a row, on every pane load, for every account,
online. It would also make a row's identity depend on a network answer and a
token that rotates.

## Decision

**A guided sign-in opens a terminal and runs the CLI's own login.** PiCode never
performs a vendor's OAuth and never presents another product's client id: the
person runs the vendor's binary, in the vendor's flow (a browser approval, a
device code, a `/login` inside a TUI), and PiCode files the result with the
import it already had. `POST /api/credentials/signin` creates that terminal
with the CLI's own login argv (`codex login`, `grok login`, `muse login`,
`opencode auth login`, `hermes auth add`; the CLIs whose login lives in their
TUI get no arguments and a hint naming the command inside it), and the roster
carries `signin: {available, hint}` so the button is honest before it is
clicked. A CLI whose vendor publishes no sign-in PiCode can start says so and
points at **Import**.

**A vault row's identity comes from the store or from the person, never from the
network.** In order: the name the person gave (`as`, asked only when the
otherwise-unnamed slot would be replaced), the account the store itself names
(Grok's `principal_id`, Codex's `tokens.account_id`, Hermes' `account_id`,
Antigravity's `id_token` subject), else the provider's own profile answer *for
display only* — filling the row's email line, exactly as `internal/usage`
already does for pi's roster, never the key. A key nothing can name stays one
row per provider, and the pane says so on screen.

**Naming a login re-keys the row that holds it.** `Store.Adopt` moves the
unnamed row's key and id to the named one instead of adding a second row with
the same credential; after that, the next sign-in to the unnamed slot is a row
of its own. The roster matches a live file to a row by key, and — for rows
whose key no offline reader can recompute — by the token the row was saved
with. It never guesses: a store that says nothing matches nothing.

**The fingerprint of a credential that carries no identity of its own is a
constant.** An `api_key` entry with no `key` field (llama.cpp's env-shaped
credential) joined the oauth constant, because hashing its bytes minted a new
row every time the endpoint changed — the pi roster showed two "Account 2"
rows, both active, on 2026-09-20. Rows saved before this change keep their ids:
`Key` uses an identity only where the bytes cannot tell two accounts apart.

## Consequences

Two subscriptions of one provider are two rows for every CLI in the pane:
automatically where the store names the account (Codex, Grok, Hermes,
Antigravity), and with one short prompt where it does not (Claude Code, Muse,
and pi itself, whose file names nothing). "Use" then alternates between them
under ADR-0166's rules, unchanged.

Harder: the roster's match is exact rather than clever. A row keyed by a name
the offline reader cannot compute (Claude Code, after a name) shows as in use
only while the saved token still matches the live file; once Claude rotates its
token the pane falls back to its "signed in here · Import into the vault" line,
and the next import restores the match. We accept that over a network call in a
pane load.

Cost if we are wrong: the identity order is the whole mechanism. If a store
names an account with something unstable (a token fragment, a session id), rows
multiply — which is why every parser fills `Identity` with an id or an email and
nothing else, and why `IdentityBearing` is checked against the parsers by test.
If a person names a login after the vendor's profile already named it, the
profile's name is not overwritten by theirs: the person's name is the key, and
their own name for it is what the row shows.

## Alternatives considered

- **PiCode performs the vendor's OAuth with its own client id.** The pi model.
  Refused: on a guest CLI it means presenting ourselves as another product's
  client, which the vendor's own token endpoint may reject and which we would
  be doing with someone else's registration. The owner's ask — "the GUI does
  the login" — is satisfied by running the vendor's binary in a PiCode
  terminal, which is the same number of clicks and no impersonation.
- **Key rows by the vendor's profile email.** Refused above: it moves the
  roster's match onto the network, and it re-keys rows on every token rotation.
  The profile answer is used for display, where a wrong answer costs a wrong
  second line rather than a wrong row.
- **One row per provider for nameless stores, as ADR-0166 recorded.** Refused
  by the owner's own use: two Claude subscriptions are the case that started
  this. The named-login prompt is the smallest thing that keeps both.
- **A PiCode-owned per-account config directory per CLI (several live slots at
  once).** Still refused for the same reason ADR-0166 refused it: it is not the
  CLI's own model, and it breaks on the CLIs that do not publish such an
  override.
- **Muse keeps its API-key-only declaration.** Refused: Muse's own launcher,
  running on this machine, reads `providers.meta.mechanism == "oauth"` and
  requires `access_token`. A declaration that cannot see a subscription is
  wrong, not conservative.
