# ADR-0183: Omp's roster reads its installed /login catalog

- **Status**: accepted (owner approved 2026-09-22, in session)
- **Date**: 2026-09-22
- **Boundary**: protocol — the roster `GET /api/credentials?cli=omp` serves is no longer only the `clicreds` declaration, and each row gains an optional `name`. Amends ADR-0169's "guests: the `clicreds` declarations" for omp only.

## Context

The owner compared Omp's own `/login` (about 90 providers) with PiCode's Omp pane, which offered only 11. Those 11 were a hand-written declaration: the providers PiCode shares with pi, each with the variable a launch injects (ADR-0178, since Omp's own logins sit in a SQLite database PiCode does not read). Measured on Omp 18.2.9: `omp models --json` lists only providers that already have a credential, which does not help with adding one. The installed package ships the whole roster, compiled, in `@oh-my-pi/pi-catalog/src/compat/rules.json`. `auth.providers[]` gives the id, the name and the login kind in `/login` order, and `providers{}` gives the `envVars` for each provider.

## Decision

`clicreds.For("omp")` and `Declarations()` append the providers in that file that the declaration lacks. The file is found next to the `omp` on PATH, and `PICODE_OMP_RULES` overrides that. Each appended row carries:

- Omp's name;
- `api_key` with the first key variable, when there is one;
- `oauth` plus a note, when Omp has a sign-in flow. That sign-in happens in Omp's own `/login`.

Omp's aliases of declared ids are skipped, and so is a provider with neither a variable nor a sign-in. The rest of PiCode follows from the declaration unchanged: vault acceptance, launch injection and the pane. The Add dialog lists providers alphabetically. It offers no key field for a provider that takes no key.

## Consequences

The Omp pane offers what Omp's `/login` offers, and a key saved for one of those providers reaches Omp at launch. The cost is that the file belongs to Omp's internals, and Omp releases almost daily. The owner accepted that: when the file moves, the reader finds nothing, the roster falls back to the 11 declared providers, and we fix the reader then. Vault ids are shared across CLIs, so a key saved under an Omp id that pi's catalog also knows (for example `cerebras`) also shows in pi's pane. That is the same vendor, so it is intended. No other CLI's roster changes.

## Alternatives considered

- **Keep extending the declaration by hand.** Rejected: ~80 rows that go stale with every Omp release, and an owner who can already see the real list.
- **`omp models --json`.** Rejected: it lists only providers that already have a credential.
- **Run `omp` to ask for the list.** Rejected: no subcommand prints the `/login` roster, and spawning a Node CLI on each pane load costs more than reading one file.
