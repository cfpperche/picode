# ADR-0169: One providers surface for every agent CLI

- **Status**: accepted
- **Date**: 2026-09-20
- **Boundary**: protocol — the roster contract both apps render, and which
  surface owns a CLI's providers. Amends ADR-0103 and ADR-0165.

## Context

ADR-0165 folded every agent CLI's credentials into one vault and gave the
eight guests a new pane (`CliCredentials.jsx`,
`GET /api/credentials?cli=`), while pi kept the editor it already had
(`Providers.jsx` over `/api/catalog`). Both write the same vault, the same
rows, and the same activation model (ADR-0166/0168) — but the owner reads two
systems: different columns, different vocabulary ("Add provider" vs "Add API
key"), and pi's two useful columns (quota **Usage**, **7d spend**) exist
nowhere else. Asked directly, the owner chose to migrate pi onto the unified
pane and carry those features to the guests "wherever they can travel".

Two facts made that cheap, and both were verified before this decision:

1. **Usage is per provider and per account, not per CLI.** The adapters
   (`internal/usage`) resolve a vault row, call the vendor and cache a report;
   `GET /api/providers/{id}/accounts/{aid}/usage` already serves exactly that,
   for any row in the vault. The roster needs no new route: it embeds the
   cached `usage.Entry`, and the row's Check is that endpoint.
2. **7d spend is PiCode's own number.** `Providers.jsx` computes it from
   `/api/sessions/stats?range=7d` via `spendByProvider` — our session files,
   not a vendor. It applies to any CLI whose rows name a tracked provider.

## Decision

`CliCredentials.jsx` is the providers surface for all nine CLIs, and
`Providers.jsx` (pi's editor) is deleted rather than kept behind a flag.
`GET /api/credentials?cli=<id>` becomes one roster shape:

- **providers** come from the CLI's own source — pi: the catalog
  (`internal/catalog`, native and custom); guests: the `clicreds`
  declarations;
- **rows** are the vault rows, identical for everybody, and each carries
  `usage` when a cached report exists, plus the identity the vendor
  volunteered (ADR-0168's rule: display only, never the key);
- the pane shows the columns pi had — `Provider · Account · Usage · 7d spend ·
  actions`, the identity chips (`Subscription · in use`, `API key · Vault`)
  riding in the Account cell, folding to a card under 840px — for every CLI, and
  renders only the actions that can work there: Verify (pi's own `auth check`,
  a guest's listing probe), Check usage (the route above), Rename,
  Pause/Resume, Sign out, plus the per-CLI doors — **Add provider** (pi's
  dialog, OAuth through pi's own login) and **Edit provider** (custom
  endpoints, `#/clis/pi/providers/custom`), **Sign in** and the native-import
  line (guests, ADR-0168).

Custom endpoints and models.json stay pi-only: no guest CLI reads them.

## Consequences

One components file, one stylesheet, one data shape; a new CLI gets the whole
roster — usage and spend included — by being declared in `clicreds`. The
guest panes gain columns that cost nothing (spend is one session-stats call;
usage is the cache), and a guest's account can now be watched for free where
the vendor publishes a quota.

What becomes harder, and is accepted: pi's pane was the only writer of
`auth.json` through its own editor, so the unified pane must route pi's
**Use** through the same activation path as a guest (it already does —
`clicreds` declares pi's `auth.json` shape); and the custom-provider flow
becomes a sub-page of a surface that is otherwise uniform, which the pane
links to explicitly rather than hiding behind a menu item.

If we are wrong: the failure is cosmetic — a column that cannot be filled says
so (`unknown`, with Check) rather than showing a guessed bar, which is
ADR-0031's rule and the reason `usage.Entry` carries `status: "unknown"`.

## Alternatives considered

- **Keep both panes, copy the columns into the guest one.** Refused by the
  owner's own reading: two surfaces over one vault keep diverging (the same
  account shows different labels and different actions in each), and every
  future feature has to be written twice.
- **Keep pi's pane, port the guest actions into it.** The guest pane carries
  more: the native-import line, the guided sign-in strip (ADR-0168), the
  per-provider Add, and the `singleOAuth` rule. Porting those into a table
  built around `/api/catalog` means rebuilding the new pane anyway.
- **A new third component, retired later.** Nothing to gain: the roster
  contract is what changes, not the renderer.
- **Move usage into the roster as *live* fetches.** Refused: a pane load would
  reach vendors (ADR-0031 forbids exactly that); the cache plus an explicit
  Check is the model pi already used.
