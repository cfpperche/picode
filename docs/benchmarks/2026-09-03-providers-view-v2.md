# Study: the Providers view, v2 — accounts, quota and the model catalog

- **Date:** 2026-09-03
- **Sources:** three live web sweeps (2026-09-03), receipts inline. Open
  repos read at HEAD: [cline/cline](https://github.com/cline/cline),
  [RooCodeInc/Roo-Code](https://github.com/RooCodeInc/Roo-Code),
  [Kilo-Org/kilocode](https://github.com/Kilo-Org/kilocode),
  [sst/opencode](https://github.com/sst/opencode),
  [zed-industries/zed](https://github.com/zed-industries/zed),
  [charmbracelet/crush](https://github.com/charmbracelet/crush),
  [farion1231/cc-switch](https://github.com/farion1231/cc-switch),
  [realiti4/claude-swap](https://github.com/realiti4/claude-swap),
  [ryoppippi/ccusage](https://github.com/ryoppippi/ccusage),
  [Maciek-roboblog/Claude-Code-Usage-Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor),
  [steipete/CodexBar](https://github.com/steipete/CodexBar),
  [can1357/oh-my-pi](https://github.com/can1357/oh-my-pi), and pi v0.84.4
  read on this machine (`~/.pi/agent/`, the installed bundle).
  Public docs for Cursor, Claude Code, Codex, Gemini CLI, Goose, Aider,
  Warp, Windsurf, Continue, Zed, OpenRouter, Vercel AI Gateway, LiteLLM,
  Portkey, Cloudflare AI Gateway, Anthropic Console, OpenAI Platform,
  Stripe, GitHub, Tailscale, Doppler, Zapier, Raycast, Notion, Auth0.
  Hosted products have no receipts; their internals are marked *inference*.
- **Scope:** `#/providers` only — the roster, the vault, the Add wizard and
  the Usage dialog. Not the composer chips, not the dashboard, not billing.

## The problem, in PiCode terms

v1 answers one question: **which logins does this machine have.** It was
the right v1 and it shipped in eight increments (`cd3856bd` → `b5be6900`).
The owner's screenshot shows the ceiling: eight providers, eight identical
`Default` rows, `account · active` on each, and a `Usage` button that only
tells the truth after you click it. Nothing on that page answers the
questions a person actually has in front of it:

- Which of these can I use right now, and which is out of quota?
- Which account is *this* one — my work Claude or my personal one?
- What is each one costing me?
- Which models do I get, at what price and what context?
- Which agents break if I sign this out?

Every one of those is answerable from data PiCode already has on disk or
one documented endpoint away. That is what v2 is.

## What v1 is, exactly (receipts)

| Piece | Where | Behaviour |
|---|---|---|
| Roster | `web/src/components/Providers.jsx:260-292` | only `signedIn` providers; per provider a face + id + **Add account**; per account a renamable label, `account`/`api key` + `· active`, Usage (conditional), Use (when inactive), Sign out |
| Vault | ADR-0013, `internal/catalog/accounts.go` | `~/.picode/accounts.json`; `auth.json` holds the **active** slot; `syncFromAuth` imports a TUI `/login` by credential fingerprint |
| Catalog | `internal/catalog/catalog.go:47-92` | `pi --list-models --offline` → provider + models (context, maxOut, thinking, images, thinkingLevels); `LoginMethods` (41 ids) adds providers that have no models |
| Add wizard | `Providers.jsx:324-397` | cmdk search → method (`api_key`/`oauth`/`both`) → key field or OAuth; loopback for Claude/Codex, device code for Copilot/Kimi/xAI |
| Usage | ADR-0031 + 2 amendments, `internal/usage/`, `UsageDialog.jsx` | per-account fetch **on open**; `windows[]` bars with `usedPercent`/`remaining` + `resetsAt`; `resets[]` banked credits with confirm-to-redeem; 26 vendor endpoints in `defaultEndpoints()` |
| Recently used | `lib/providerRecents.js`, `Providers.jsx:294-312` | localStorage list of signed-out providers, Sign in / Remove |
| llama.cpp | `LlamaPanel.jsx` | URL + key, load/unload, HF download |
| Mobile | `mobile/screens/More.jsx:16,75` | same component in a sheet, subtitle "Accounts, keys, usage" |
| Elsewhere | `lib/providers.js`, `ConfigFields.jsx` | agent/automation forms list signed-in providers as three bare `<select>`s |
| Spend | ADR-0041, `internal/session/stats.go:104` | `byProvider[]` cost/messages already computed — **on the dashboard, not here** |

## What the field ships

### Family 1 — agent IDEs and CLIs (the roster)

| Tool | Row shows | Distinctive |
|---|---|---|
| **Kilo Code** `webview-ui/src/components/settings/ProvidersTab.tsx` | one config per provider in **four sections**: gateway (pinned), Connected, Popular, collapsible **Disabled**;each row a **source badge** — `environment` / `API key` / `config` / `custom` / `ChatGPT` / `local` | env-sourced rows are **read-only**, labelled "Set by environment variable" |
| **Zed** `crates/language_model/src/api_key.rs` | five access paths per provider (Zed-hosted, API key, **existing subscription**, gateway, local); keys in the OS keychain, never `settings.json` | `ApiKeySource`'s `Display` renders **"environment variable ANTHROPIC_API_KEY"** or **"system keychain"** — the credential's *origin* is a first-class displayable value |
| **Roo Code** `src/core/config/ProviderSettingsManager.ts` | named **API Configuration Profiles** (provider + key + model + temperature + thinking budget + `rateLimitSeconds` + `consecutiveMistakeLimit`) in VS Code Secret Storage | `chat/ApiConfigSelector.tsx`: composer-level profile dropdown, pinned/unpinned sections, search above 6 entries, and a **lock icon** (`lockApiConfigAcrossModes`) making one profile global or per-mode |
| **Cline** `apps/vscode/src/shared/api.ts` | 47-provider union, no profiles — every field doubled `planMode…`/`actMode…` | provider list is a **live catalog** (`useProviderListings()` + Fuse.js), not a static array |
| **Cursor** `cursor.com/docs/settings/api-keys` | per-provider key + enable toggle, per-model checkboxes | the only tool in the study with an explicit **Verify** button firing a real test request on save |
| **Raycast** `manual.raycast.com/ai/bring-your-own-keys` | provider dropdown, key, **Verify**, **"Manage in [Provider] Console"**, enable/disable toggle | a **key icon appears next to that provider's model names in the model picker** — credential state legible at the point of use |
| **Claude Code** `code.claude.com/docs/en/iam` | `/status` rows: **Login method** (subscription / Console / cloud), Profile, API key — and within **3 days of expiry**, `Login: Expired — log in again` with the saved org/email | a documented **7-step credential precedence**, and `apiKeyHelper`: a script re-run every 5 min, warning at >10s, failing after 3 attempts |
| **Goose** `goose-docs.ai/docs/getting-started/providers` | provider grid, keychain or `secrets.yaml` | Quick Setup **auto-detects the provider from the key's format** |
| **Aider** `aider.chat/docs/llms/warnings.html` | no UI at all | failure text that names the fix: "Missing these environment variables: `AZURE_API_BASE`, `AZURE_API_VERSION`, `AZURE_API_KEY`" |
| **pi** v0.84.4, `packages/coding-agent/src/core/auth-storage.ts` | `auth.json` is `Record<string, Credential>` — **one credential per provider, no arrays, labels or account ids**; native multi-account has **not** landed | `pi auth check --provider anthropic --json` → `{"status":"ready","provider":"anthropic","authType":"oauth"}`, with `--no-refresh`; resolution order `--api-key` → `auth.json` → env → `models.json` |

### Family 2 — multi-account switchers and quota monitors

| Tool | What it does | Distinctive |
|---|---|---|
| **cc-switch** v3.13 | 8 host tools, tray menu switches instantly | **quota rendered inline on each provider card** — percent + reset countdown, green <70 / orange 70–89 / red ≥90; automatic for Claude/Codex/Gemini/Copilot, template or custom JS for the rest; refresh 0–1440 min and **only the enabled provider refreshes in background** |
| **claude-swap** | slots with short aliases, creds in Keychain | **proactive** auto-switch: at 90 % of the active window it moves to the account with the most quota left, 5-minute cooldown, hysteresis, strategies `best` vs `consume-first`; `cswap run` **scopes an account to one terminal** so two accounts run in parallel |
| **codex-multi-auth** | accounts keyed **by email**, hotkeys 1–9 | `best --live`, `report --json`, and a bounded outbound request budget so the pool cannot spin when everything is limited |
| **opencode-go-multi-auth** | pooling proxy with priority failover | a **control room**: KPI row (enabled / active / cooldown), an "errors told us" panel surfacing exhausted keys **with the upstream's own reset time**, per-key circuit state, live test buttons |
| **ccusage** `ccusage.com/guide/blocks-reports` | local JSONL only, 19+ agent CLIs | 5-hour blocks with **burn rate in tokens/min and "projected final cost if this rate holds"**; `--token-limit max` auto-detected from your own history; bar at <70 / 70–90 / ≥90 |
| **Claude-Code-Usage-Monitor** v4 | TUI, burn rate, "tokens run out at…" | an **official-limit trust layer** — the vendor's `rate_limits` win when fresh, local estimates are fallback **and are labelled as estimates**; plus exit codes for automation (0/10/11/20/30) |
| **CodexBar** | menu bar, 69+ providers | icon is a two-bar meter — top = 5 h (or credits once weekly is exhausted), bottom = weekly; per provider windows, credit balance, monthly spend, **incident/status badges**, threshold and weekly-reset notifications |
| **oh-my-pi** | 60+ providers | **multiple credentials per provider with round-robin rotation, session affinity and per-credential backoff** — the thing pi itself refuses |
| **pi-statusline** / **pi-codex-status** / **pi-zai-usage** | pi extensions | `pi-statusline` is **off by default and makes zero network calls until enabled**, and **omits the segment entirely for xAI / OpenCode Go rather than guessing**; `pi-codex-status` refreshes **opportunistically from `x-codex-*` response headers**, spending no extra call |

### Family 3 — credential dashboards (the row spec)

| Source | Row / behaviour | Distinctive |
|---|---|---|
| **OpenRouter** keys + BYOK | key object carries `limit`, `limit_remaining`, `limit_reset`, `usage_daily`/`weekly`/`monthly`, `byok_usage`, `expires_at` — but **no `last_used`** | BYOK holds **multiple named keys per provider with drag-and-drop priority**, split into **Prioritized** and **Fallback** sections |
| **Cloudflare AI Gateway** | multiple provider keys, aliased | a request picks one with `cf-aig-byok-alias`; **`default` is used implicitly** — an active slot *plus* named overrides |
| **Vercel AI Gateway** | one list of all providers, each with **Add** | **Test Key** runs a real cheap query and the result is a **clickable badge opening the code and the raw JSON response**; budgets show `$1.04 / $10 spent` or "Unlimited budget", alert at 50/75/100 %, and the docs say plainly it is **a soft cap, not a hard limit** |
| **Anthropic Console** | secret shown once, redacted hint after; **Disable is reversible**, Delete is permanent | **graduated expiry mail** — 7 days ahead for keys living ≥14 days, 1 day ahead for ≥7 days, silence below (GitHub's warn-immediately behaviour is the complaint this avoids) |
| **Stripe** | ⋯ per key: Edit (name + free-text **note on where you saved it**), request logs, Expire, Restore, **Roll** | rolling keeps both keys valid up to 7 days and **prints the remaining time under the key name**; 180 days unused → limited access |
| **Zapier connections** | columns Name, App, **Zap workflows (a count of dependents)**, Last modified, People | status is Active / **Expired with a Reconnect button inline on the row**, and the delete dialog **names the blast radius** |
| **gh CLI** | `gh auth login` adds, `status` marks the active one, `switch` changes it | the documented failure mode: switching is **machine-wide**, and the docs' own mitigation is "get into the habit of running `gh auth status`" — a habit is not a UI |

## Patterns the field agrees on

1. **Quota belongs on the roster row, not behind a button.** cc-switch v3.13,
   claude-swap's TUI, the Antigravity monitors and CodexBar all render every
   account's window inline. Behind-a-button quota is the pre-2026 design, and
   it is what v1 ships.
2. **Percent + reset countdown is the atomic unit**, and the traffic light is
   70/90 — cc-switch and ccusage landed on the same thresholds independently.
   PiCode's `barTone()` already uses 70/90 (`lib/providerUsage.js:16-21`).
3. **Never invent a number**, with three independent receipts: `pi-statusline`
   omits the segment for providers it cannot measure; CodexBar refuses a
   synthetic 0 %; Claude-Code-Usage-Monitor labels estimates as estimates and
   prefers the vendor's own `rate_limits`. Claude Code shows
   **"Showing last-known usage"** with a retry key when the usage endpoint is
   itself rate-limited. Three states, never two: **live / stale / unavailable**.
4. **The credential's origin is displayable.** Zed prints "environment variable
   X" vs "system keychain"; Kilo badges each row and makes env rows read-only.
   Env beats stored credential everywhere, and everywhere it is documented.
5. **Subscription and API key are different first-class row types**, labelled —
   pi's own TUI prints `subscription configured` vs `API key configured`.
6. **Validate at entry with a real call.** Cursor Verify, Vercel Test Key,
   Raycast Verify, Zapier Test Connection, Auth0 Try Connection. The agent
   IDEs that skip it (Cline, Roo, Kilo, Zed) validate implicitly on the first
   request — which means the user finds out mid-task.
7. **Disable and Sign out are different verbs.** Anthropic (reversible
   Disable), OpenRouter (`disabled`), LiteLLM (`blocked`), Raycast and Vercel
   (Enabled toggle). v1 only has Sign out.
8. **Identity comes from the API, the label is a user alias.** codex-multi-auth
   keys accounts by email; plan tier is read from the vendor (`copilotPlan`),
   never typed.
9. **Destructive confirmations name the blast radius**, and the row carries a
   dependent count (Zapier's "Zap workflows").
10. **Model metadata is externalized and always the same field set**:
    context, max output, four prices, image/tool/reasoning/cache booleans —
    models.dev (OpenCode, pi), catwalk (Crush), litellm (Aider).
11. **Two-tier model roles are universal**: Crush `large`/`small`, OpenCode
    `model`/`small_model`, Goose lead/worker, Aider main/editor/weak, Claude
    Code `opusplan`, Continue's seven roles. PiCode already made this call in
    ADR-0028 (`packages/pi-roles/`) — the v2 page is where those roles become
    visible next to the models they name.
12. **Rotation, where it exists, is provider-scoped and cooldown-driven**, and
    swapping a credential file mid-session is unsafe — the careful tools block
    it while the agent runs.

## What PiCode adapts (the v2 proposal)

The page becomes three tabs on one route. Everything below is either data
already on this machine or one documented endpoint away.

### Tab 1 — Accounts (the roster, rebuilt)

| Pattern | From | PiCode version |
|---|---|---|
| **Quota inline on the row** | cc-switch v3.13, claude-swap, CodexBar | each vault row gets a compact 5h/7d strip (percent + reset) beside the label, reusing `usage.Report.windows[]`. The Usage dialog survives for per-model windows, banked resets and the redeem confirm |
| **Three honest states** | pi-statusline, CCUM v4, Claude Code | `live` (fetched now) · `stale · 4m` (cached, shown dimmed) · `—` with a one-word reason (`no plan windows`, `sign in again`). No bar is ever rendered from a guess. This is ADR-0031's "never invent a percentage" extended from the dialog to the roster |
| **Refresh policy** | cc-switch (only the enabled provider), pi-codex-status (`x-codex-*` headers) | background refresh for the **active slot only**, interval in settings with 0 = off; inactive rows show their last cached value with its age and refresh on demand. Fetch-on-open stays for the dialog |
| **Identity from the API** | codex-multi-auth (email), Copilot `copilotPlan` | second line on the row: `you@work.com · Max 20x`, read from `api.anthropic.com/api/oauth/profile` and the plan fields the usage adapters already parse. The editable label stays the user's alias |
| **Credential source badge** | Zed `ApiKeySource`, Kilo `ProvidersTab.tsx` | `vault` / `auth.json` / `environment` — pi resolves `--api-key` → `auth.json` → env → `models.json`, so a provider answered by `ANTHROPIC_API_KEY` must say so and not offer Sign out |
| **Verify** | Cursor, Raycast, Vercel, Auth0 | a Verify action per row calling **`pi auth check --provider <id> --json --no-refresh`** — pi already ships the exact primitive. Vercel's touch: the result badge opens the raw response, because the audience is developers |
| **Dependent count + blast radius** | Zapier | "3 agents · 1 automation" on the row, from `agents.provider` and `automations.provider`; Sign out's confirm names them |
| **Pause, not just Sign out** | Anthropic, Raycast, OpenRouter | a vault row can be disabled without deleting the credential — it stops being an auto-switch candidate and disappears from the agent forms |
| **Expiry warning** | Claude Code (3 days), Anthropic (graduated mail) | OAuth rows carry `expires`; a row whose refresh has been failing says `sign in again` **before** the next agent start does |
| **7-day spend on the row** | OpenRouter `usage_weekly`, Vercel `$1.04 / $10` | `stats.byProvider[]` is already computed for the dashboard (ADR-0041). The number moves to where the account lives; the dashboard keeps the fleet view |
| **One searchable catalog** | Vercel BYOK, Notion connections | replace the signed-in list + "Recently used" split with **Signed in / Available / Disabled** (Kilo's shape), search at the top, "Recently used" surviving only as ordering inside Available |
| **Console link** | Raycast "Manage in [Provider] Console" | one link per provider in the Add dialog and on the row. A link, not prose — the copy rule stands |
| **Key-format detection** | Goose Quick Setup | paste `sk-ant-…` and the wizard preselects anthropic. Collapses the common Add path from two steps to one |
| **Failure text that names the fix** | Aider | "llama.cpp: no router at `http://127.0.0.1:8080`" beats "couldn't sign in" |

### Tab 2 — Models

pi's `--list-models` has **no price column**, but `~/.pi/agent/models-store.json`
— which `internal/catalog` already reads for `thinkingLevelMap` — carries
`cost.{input,output,cacheRead,cacheWrite}`, tiered pricing and
`compat.allowedFallbackModels`. Roo's `ModelInfoView.tsx` and Cursor's model
table are the display bar: context, max output, four prices, capability
checkmarks. Two things this unlocks that no other PiCode surface offers:

- **A real model picker.** `enabledModels` already exists in pi settings
  (`internal/pisettings/pisettings.go:24`, edited today as raw glob patterns in
  `PiSettings.jsx:169`). Checkboxes over a priced catalog write those patterns.
- **Roles at the point of use.** ADR-0028's `packages/pi-roles/` binds a role to
  a model; the models tab is where a role badge belongs, the way Continue puts
  a role dropdown next to each model.

### Tab 3 — Activity

Claude Code's `/usage` breaks cost down **by skill, subagent and MCP server**
and prints a cache-health line. PiCode computes the same class of numbers from
its own session JSONL (`internal/session/stats.go`: `byModel`, `tokens`,
`series`) — per provider, filtered to this account, this is the retrospective
half the vendor endpoints cannot give. ccusage's burn rate and "at this pace,
exhausted at HH:MM" is the one derived number worth adding, computed from our
own events and labelled as a projection, never as a vendor figure.

### The one decision that needs the owner

ADR-0013 says `auth.json` holds **the** active slot, because pi reads exactly
one credential per provider (confirmed at v0.84.4: `Record<string, Credential>`,
no arrays). Every serious multi-account tool in family 2 has moved past a single
global slot — OpenRouter's ordered Prioritized/Fallback lists, Cloudflare's
alias-with-a-`default`, oh-my-pi's round-robin with session affinity and
per-credential backoff, claude-swap's `cswap run` scoping an account to one
terminal. `gh auth switch` is the counter-example the field cites: machine-wide
switching whose documented mitigation is a habit.

PiCode is closer to solving this than any of them, because it **starts** the
processes: ADR-0009 already passes `--provider/--model/--thinking` per agent,
and ADR-0039/0040 already isolate a session. Two candidate mechanisms, both
unproven and both needing measurement before an ADR:

1. **Per-agent credential pinning** — spawn pi with a pinned account
   (`--api-key` for key rows; an isolated pi home for OAuth rows), so two agents
   use two accounts concurrently and the global slot stops being contended.
2. **Proactive auto-switch on the slot we have** — claude-swap's rule (at 90 %
   of the active window, move to the account with the most quota left, cooldown
   + hysteresis), applied only *between* agent starts, never mid-session,
   because every switcher in the study warns that swapping a credential file
   under a running agent corrupts it.

Neither ships without the owner's call: (1) enlarges ADR-0009's spawn contract,
(2) makes PiCode take an action the user did not click. Non-negotiable #6 —
decisions are provisional, and this one is a candidate for re-measuring.

## What PiCode refuses

| Temptation | Why not |
|---|---|
| **Proxying model traffic** (LiteLLM, 9router, opencode-go-multi-auth) | It is the only way to rotate accounts mid-request, and it puts PiCode in the data path of every token. ADR-0003: pi talks to vendors directly. A gateway is a different product |
| **Scraping browser cookie jars** for Groq, Mistral, Cursor, the DeepSeek dashboard | Already refused for Grok in ADR-0031. Those endpoints are session-scoped; the honest row is `unavailable`, with the vendor's console one link away |
| **Budgets and spend caps enforced by PiCode** | We are not in the request path, so any cap is advisory. Vercel is honest that even a gateway's budget is "a soft cap, not a hard limit". We can **alert** at a threshold (push already exists, `internal/push`); we must not claim to cap |
| **A percentage for providers without an endpoint** | ADR-0031's rule, restated for the roster. `pi-statusline` omitting xAI is the standard to match |
| **Key rotation with a grace period** (Stripe, LiteLLM) | We do not mint keys. The equivalent is the Raycast console link |
| **Named profiles bundling provider + model + settings** (Roo) | ADR-0028 already made this call: roles live in `packages/pi-roles/`, per-agent config lives on the agent (ADR-0009). A third profile store on this page would be a fourth place to look |
| **Setup prose in chrome** | Owner directive 2026-08-23. Every explanation stays in `www/guide/providers.md`; the page carries state and actions |

## Debts this study found

1. **GitHub Copilot's premium requests were retired on 2026-06-01**, replaced by
   token-metered *AI Credits* with budgets at enterprise / cost-center / user
   level. `internal/usage` reads `copilot_internal/user` for
   `quotaSnapshots.premiumInteractions` — that adapter needs re-measuring
   against the new billing model, and it never had a reset date to begin with
   (the field does not exist; the reset is the 1st at 00:00 UTC and must be
   computed).
2. **Gemini CLI stopped serving Google AI Pro/Ultra individuals on 2026-06-18.**
   The consumer quota surface moved to Antigravity's
   `cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary`
   (`buckets[].remainingFraction`, `buckets[].resetTime`, split into "Gemini
   Models" and "Claude and GPT models"). PiCode reads no Google quota today.
3. **Endpoints PiCode does not yet read**, all API-key or OAuth scoped, all
   fitting `usage.Report` unchanged: `api.anthropic.com/api/oauth/profile`
   (email + plan for the identity line), Anthropic `overage_spend_limit` and
   `prepaid/credits`, `openrouter.ai/api/v1/credits` (account credits, distinct
   from the per-key `/api/v1/key` we already call),
   `api.deepseek.com/user/balance` (`granted` vs `topped_up` separately),
   `api.moonshot.ai/v1/users/me/balance`, Fireworks `billing/summary`, the Qwen
   coding-plan endpoint (`per5Hour*`, `perWeek*`, `perBillMonth*`) — which
   ADR-0031 refused in V3 for lack of an API-key path and which now has one
   worth re-checking.
4. **z.ai banked resets** remain blocked on an unknown endpoint
   (`docs/handoff.md`), unchanged by this study.

## Open questions for the ADR

1. **Does the roster strip cost too much?** Eight accounts × two windows on
   every page load is eight undocumented vendor calls. cc-switch's answer —
   background-refresh the active slot only, everything else cached with a
   visible age — is the proposal, but the cache TTL and the "what does a cold
   page show" behaviour need to be decided before UI work, not after.
2. **Per-agent credential pinning: is there a mechanism at all?** `--api-key`
   covers key rows. For OAuth rows pi reads `auth.json` from its home; whether
   an isolated home per agent is acceptable (and what it costs in `models-store`
   duplication and refresh races) is a spike, not a design.
3. **Auto-switch: opt-in, and at what threshold?** claude-swap defaults to 90 %
   with a 5-minute cooldown. PiCode taking that action silently would be the
   first time the product changes a credential the user did not click. Proposal:
   off by default, a visible "switched to *personal* at 91 %" line in the feed.
4. **Does the identity line leak?** `oauth/profile` returns an email address
   that would then render on a page a paired phone can open (ADR-0049). Masked
   by default (`c…e@gmail.com`) or shown in full is the owner's call.
5. **Three tabs or one page?** Kilo's four-section single page and Roo's tabbed
   settings both work at this size. Tabs assume the models catalog earns its
   own surface, which depends on question 2 of the models tab: does anyone edit
   `enabledModels` often enough to deserve a picker?
