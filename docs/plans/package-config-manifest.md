# Package configuration for every extension (config descriptors)

> **Status: plan — direction requested by the owner (2026-09-11), pending ADR.**
> Extends [ADR-0099](../decisions/0099-package-configuration-gui.md) and pulls
> forward the end state it deferred: a **declarative config descriptor** per
> package, so any extension with a config file gets a Configure affordance in
> the Packages view without a bespoke editor.
> **Trigger:** the Packages grid shows Configure on `pi-roles` alone
> (screenshot 2026-09-11 15:05); `pi-web-search`'s `web-search.json` was never
> configured because nothing in the UI said it existed — the tool only names
> the file inside its own error message.
> **Study:** ADR-0099's sources (VS Code settings scopes, Raycast manifest
> preferences, Obsidian plugin tabs, Home Assistant reconfigure/options) — the
> adaptation is Raycast's: configuration is a property of the package,
> surfaced where the package is managed.

---

## Context

ADR-0099 built the roles editor and set the contract this plan must keep:
files stay the only source of truth; unknown keys survive every write; saves
are atomic (tmp + rename); a file the parser rejects is never silently
overwritten (409 + explicit replace); reset is scoped; the page says when a
change applies; saves publish a `packages.config` feed event. It also fixed
the shape of the gap: adapters are **code** (`pipkg.ConfigKindOf` matches
`pi-roles` by substring; `internal/server/packages_config.go` and
`web/desktop/src/components/PackagesConfig.jsx` are roles-specific), and the
declarative manifest was deferred *"after pi-compact proves the second
case"*.

Three things changed that make extracting it now cheaper than a second
hand-written adapter:

1. **The second case already exists, and it is not pi-compact-shaped.**
   `pi-web-search` reads `~/.pi/agent/web-search.json` (`{provider, model}`,
   provider must back native search). A roles-style adapter (two layers,
   inheritance, scoped resets) would be mostly dead code for it.
2. **The failure mode is discoverability, not editing.** The file was never
   created because no surface says it exists. A Configure button that only
   appears for adapters guarantees the next package with a config file fails
   the same way.
3. **The roles editor is the outlier, not the pattern.** Two-layer overlay
   inheritance is one package's semantics. The common case — one JSON file,
   a handful of typed fields, one scope — is exactly what a descriptor
   renders generically.

## Decision being requested (ADR to seed at C1: `make adr NAME=package-config-descriptors`)

A package is configurable in the GUI when a **descriptor** resolves for it.
Resolution order:

1. **Package manifest** — `picode.config` in the extension's `package.json`
   (Raycast `preferences` shape, adapted). The upstream-friendly path: a
   package carries its own description.
2. **PiCode catalog** — `internal/pipkg/configcatalog.go` maps known
   installed packages to descriptors. Entries: `pi-web-search` →
   `web-search.json` (agent scope) and `pi-compact` → `.pi/compact.json`
   (workspace scope). Sweep result (2026-09-11): pi-inbox/pi-checklist's
   `server.json` is connection identity PiCode provisions itself — not
   exposed; `pi-mcp-adapter`, `pi-diff`, `pi-browser-capture`,
   `pi-byteplus-modelark`, `pi-agent-browser-native` have no user config
   file — they show no Configure button, honestly.
3. **Nothing** — no descriptor, no Configure button. ADR-0099 §5's honest
   absence is kept verbatim: never a generic JSON editor, never a claim that
   a package "has no settings".

Descriptor v1 (declarative data, not code):

| Field | Meaning |
|---|---|
| `id`, `title` | Stable config id (route + API key) and display name |
| `match` | Which installed package this describes (name / path pattern) |
| `files` | One or more `{scope, path, format}`; v1 scopes: `agent` (machine-global, e.g. `~/.pi/agent/web-search.json`) and `workspace` (e.g. `.pi/roles.json`); per-agent overlays stay roles-bespoke |
| `fields` | Typed list: `string`, `enum`, `boolean`, `number`, `secret` (masked), `model` (provider+model pair with a capability filter — "backs native search" reuses the provider-kind check the tool itself applies) |
| `application` | The honest-apply sentence the page renders (ADR-0099 §6) |

Everything ADR-0099 guarantees is inherited by the generic engine: server
reads/writes the same files the extension does; validation before write;
merge onto the raw document so unknown keys survive; atomic rename; 409 on
an unparseable file with explicit replace; scoped reset; `packages.config`
feed event. pi-roles keeps its bespoke editor — the descriptor system is
additive, and the roles adapter remains the proof that rich semantics stay
possible in code.

## Phases

| Phase | Deliverable | Acceptance |
|---|---|---|
| C0 — spike | Hand-written web-search descriptor + generalized GET/PUT round-trip + minimal form behind the existing route | `web-search.json` created and edited from the Packages card with zero bespoke client code; tool picks the model up on next search |
| C1 — descriptor engine | `internal/pipkg` catalog + manifest reader; `packages_config.go` driven by descriptors; `ConfigKindOf` → descriptor resolution; ADR seeded and accepted | Decision-table tests below green; roles path byte-identical in behavior |
| C2 — generic form | `PackageConfigForm.jsx` driven by the descriptor (Zod contracts in `web/shared/contracts/schemas.js`, `noValidate`, one control height); Configure button renders whenever a descriptor resolves; per-field errors, empty/blocked states one-line-plus-action | Desktop only (mobile keeps the ADR-0102 "config links offer the desktop layout" rule); visual-review PASS with empty, error and saved states |
| C3 — catalog coverage | Descriptors for every installed package we know has a config (discovery step: sweep `~/.pi/agent/npm/node_modules` + `packages/` for config-file reads); upstream `picode.config` proposal drafted for pi-web-search | Every installed package with a config file shows Configure or is listed as "no descriptor yet" in the plan's debt |
| C4 — docs | ADR accepted; `docs/architecture/cli-packages.md` + `model-roles.md` updated; changelog fragment; docs-site entry for the Packages view | `make close` green; docs say which packages are configurable and where files live |

Estimated 2–3 sessions (C1 ≈ 1, C2 ≈ 1, C3+C4 ≈ ½–1).

## Decision table (C1/C2 tests must cover every row)

| Conditions | Observable result |
|---|---|
| Installed package, no descriptor | No Configure button; nothing claims "no settings" |
| Descriptor resolves, config file absent | Form opens with defaults; PUT creates the file atomically |
| File exists, unparseable | GET reports the parse error; PUT answers 409; UI requires explicit "Replace file…" |
| Field fails descriptor validation | PUT 400 with per-field errors; UI shows inline errors, draft retained |
| Unknown keys in the file | Preserved verbatim after PUT (merge onto raw doc) |
| `model` field names a provider without native search | PUT 400 listing supported provider kinds; UI offers the supported set in the picker |
| Scope agent vs workspace | v1 renders exactly the scopes the descriptor declares; no invented scope |
| Save | Feed event `packages.config` with package + scope; page shows the descriptor's `application` sentence |

## Open questions for the owner

1. **Default search model for C0** — `web-search.json` needs `{provider,
   model}`. Suggestion: `google-generative-ai` + a Gemini flash (native
   search, cheap). The C0 spike makes this editable, so the initial value is
   not a commitment.
2. **Catalog vs upstream-first** — recommend catalog-first (works today for
   packages we do not publish), with the `picode.config` manifest offered
   upstream as packages adopt it.
3. **Secrets** — v1 supports a masked `secret` field type but no credential
   vault integration; configs with real keys (e.g. brave-search) stay
   hand-edited until that is wanted.

## Validation through the GUI (owner directive, 2026-09-11)

Config files are **never created by hand** — creating and editing them
through the Packages GUI *is* the acceptance test for this work, for
web-search and for every descriptor added after it. Validated on a scratch
instance: web-search (create → edit → clear → broken-file replace) and
compact (workspace file pre-populated, tri-state enabled, min/max refusal,
save preserving `fallback`/`atTokens`).
