# Unified providers surface — one pane for all nine CLIs

Status: **approved by the owner (2026-09-20)** — "migrar pi para unificar o
sistema, levando junto as funcionalidades que temos na tela do Pi para os
demais agent CLIs também quando possível levar".
Supersedes the pane split ADR-0165 drew (pi keeps its editor; the eight guests
get the vault pane). Becomes ADR-0169.

## Why

Two panes over one vault. They already share the store (ADR-0165), the rows
(`internal/credentials`), the activation model (ADR-0166) and the guided
sign-in (ADR-0168) — but the *surface* differs: pi renders a roster table with
Usage and 7d spend, built from `/api/catalog`; the guests render a
provider-grouped list with Import / Sign in / Use, built from
`GET /api/credentials?cli=`. The owner reads that as two systems, and the
guest panes lack the two columns that make a subscription's cost visible.

## Decision

**One component, one roster endpoint, for all nine CLIs.** `CliCredentials.jsx`
serves pi too; `Providers.jsx` (pi's editor) is deleted rather than kept
behind a flag. `GET /api/credentials?cli=<id>` becomes the single roster:

- providers come from the CLI's own source — pi: the catalog (native +
  custom, `internal/catalog`); guests: `clicreds` declarations;
- rows are the same vault rows for everybody;
- every row gains `usage` when a report exists (`internal/usage` adapters), so
  the **Usage** and **7d spend** columns pi had are available wherever the
  vendor answers — the adapters are provider+account based, not pi-specific;
- a new `POST /api/credentials/{provider}/{id}/usage` fetches one report for
  one account, on an explicit click, under ADR-0129's one-call rule. Pi's
  background refresh (`/api/providers/usage`) stays the automatic path.

Actions stay the union, each rendered where it can work: Rename, Verify
(the CLI's own check: `pi auth check` for pi, a listing probe for a guest),
Check usage, Pause/Resume, Sign out, plus the per-CLI doors: **Add provider**
(pi's dialog, OAuth and API keys through pi's own login) and **Edit provider**
(custom endpoints, `#/clis/pi/providers/custom`), **Sign in** (the CLI's own
login, ADR-0168) and the native-import line for guests.

## Contract (frozen — both sides code against this)

```jsonc
GET /api/credentials?cli=<id>
{
  "cli": "pi", "cliName": "Pi",
  "vault":  { "readable": true },
  "signin": { "available": true, "hint": "…" },        // guests: the CLI's own login
  "add":    { "kind": "provider" | "key", "label": "Add provider" },  // what the bar's primary opens
  "custom": { "available": true, "href": "#/clis/pi/providers/custom" },  // pi only: the models.json page
  "providers": [{
    "id": "anthropic", "kinds": ["api_key","oauth"], "env": {"api_key":"ANTHROPIC_API_KEY"},
    "note": "", "custom": false, "singleOAuth": false,
    "verify": "provider" | "row",   // pi: `pi auth check` per provider; guests: the listing probe per row
    "native": { "detected": true, "kind": "oauth", "label": "me@x.com", "imported": false },
    "accounts": [{
      "id": "…", "label": "Default", "type": "oauth", "active": true, "paused": false,
      "origin": "vault", "hint": "sk-…", "email": "me@x.com", "plan": "Max",
      "activatable": true,
      "usage": { /* usage.Entry, the same JSON /api/providers/usage carries */ }
    }]
  }]
}
```

No new route: the per-row **Check** is the existing
`GET /api/providers/{provider}/accounts/{id}/usage` (`usage.FetchAccount`, one
vendor call, cached, writes the identity the vendor volunteers back to the
vault), and the automatic path stays `POST /api/providers/usage`
(`usage.ActiveTargets` + `Refresh`). **7d spend is ours, not the vendor's** —
`/api/sessions/stats?range=7d` + `spendByProvider`, one call for the whole
pane, so the column travels to every CLI whose rows name a tracked provider.
Unchanged: `/api/catalog` (pi's provider + model catalog, the Add-provider
dialog and the model picker read it), the per-row verbs
(`PATCH`/`DELETE`/`use`/`pause`/`verify`), and the usage routes above.

## What carries over, per CLI

| Pi's feature | Guests get it? |
|---|---|
| Usage windows + Check | Yes — same adapters, per account (only where a provider adapter exists) |
| 7d spend | Yes — same summary, same condition |
| Identity (email · plan) | Already there (ADR-0168's display identity) |
| Add provider with OAuth | Becomes **Sign in**: the CLI's own login (ADR-0168) |
| Custom endpoints / models.json | Pi only — no guest CLI reads models.json; the page stays, reachable from pi's pane |
| Verify via `pi auth check` | Guests keep the listing probe (`Verify with the provider (1 request)`) |
| Use / Pause / Sign out / Rename | Both, unchanged |

## Phases

1. **Backend** — `handleCredentials` serves pi from the catalog; `usage` on
   each row from the cached report; the per-account usage route; tests for the
   pi shape and the guest with/without a report.
2. **Frontend** — one `CliCredentials.jsx` for both apps: the pi grid
   (`providers.css` geometry: columns `Provider · Account · Usage · 7d spend ·
   actions`, identity chips inside the Account cell, folding to a card under
   840px) carrying the bare
   `credentials.css` rows it already has; the bar's primary action per CLI
   (`Add provider` / `Add API key`); `CliProviders.jsx` loses the pi branch and
   `Providers.jsx` is deleted.
3. **Docs** — ADR-0169, `docs/architecture/cli-providers.md` (one pane),
   `credentials.md` (the roster is the surface), `docs-site/guide/providers.md`
   (one pane, the same actions for every CLI), changelog fragment.
4. **Verification** — Go tests on both shapes; a scratch run driving pi
   (`Use` on a second account, Check usage, Pause) and a guest
   (`Sign in` → `Check now`, `Use`), screenshots read in a subagent at desktop
   and 414px; `overlayAudit` ok.

## Out of scope

- Removing `/api/providers/usage` (pi's refresh loop feeds the same cache the
  pane reads; it stays until something else needs it gone).
- Guest-side OAuth with pi's client id (refused in ADR-0168).
- A per-CLI "Add provider" wizard for guests — their vendors' CLIs own that
  flow, and the Sign in button is the door.
