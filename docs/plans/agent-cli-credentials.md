# Agent CLI credentials — one vault, then guided use

Status: **proposed** — the six owner questions at the end gate step 1.
Study with receipts: [../benchmarks/2026-09-20-agent-cli-credentials.md](../benchmarks/2026-09-20-agent-cli-credentials.md).
Related: ADR-0013 (the Pi-only vault this generalizes), ADR-0058 (roster
semantics), ADR-0069/0070 (launch, inspection), ADR-0103 (providers pane per
CLI), ADR-0129 (custom providers; the one authorized probe), ADR-0163 (per-CLI
declaration + file-driver pattern).

The owner's brief, restated: stop walking to each vendor and each CLI to log
in. Keep BYOK under GUI control, with **as many accounts per provider as the
person has** — key or subscription. Step 1 is the storage problem; step 2 is
guided use inside the agent CLIs.

## Shape

```
        ┌──────────────────────── one vault ────────────────────────┐
        │ <DataDir>/credentials.json   (AES-256-GCM envelope, 0600) │
        │ <DataDir>/credentials.key    (random 32B, 0600)           │
        │ accounts: provider · kind(api_key|oauth) · label ·        │
        │           identity · paused · health · origin             │
        └───────────┬───────────────────────────────┬──────────────┘
                    │ providers pane (all 9 CLIs)   │ launch injection
   #/clis/<cli>/providers  ── roster, add, import,  ── env var  (dropEnv wins)
                              verify, pause, sign out    or per-account dir
                                                         (config-dir env var)
```

Three facts make this cheap in PiCode: the vault machinery exists for pi
(ADR-0013), the per-CLI declaration pattern exists (`internal/clisettings`),
and the launcher already owns the environment (`cliEnvironment`,
`prepareCLITerminal`) and already reserves the exact env names an injector
needs (`HOME`, `GROK_HOME`, `HERMES_HOME`, `OPENCODE_CONFIG`,
`PI_CODING_AGENT_DIR`, `PICODE_*` are refused to the user in
`clilaunch.Validate`).

## Step 1 — the vault (storage)

### 1.1 Store

`<DataDir>/credentials.json` encrypted with AES-256-GCM (stdlib) under a
random 32-byte key in `<DataDir>/credentials.key`; both 0600, atomic replace
(tmp + rename), one in-process writer (the daemon). No new dependency, no
keyring, no passphrase to lose. Backup/restore treats the pair exactly as
`accounts.json` is treated today: copied only when the snapshot's `secrets`
option is on, and restoring an **old** snapshot (which carries
`accounts.json`) is absorbed by the same migration that runs at boot —
that is the compatibility path, not a second reader.

Threat model, stated where the user will read it: the envelope protects
*copies* — a snapshot on cloud storage, a dotfiles repo, a support bundle, a
file another uid can read. It does not stop an attacker already running as
this user, because the key is beside the data. Honest, and the same posture
as ADR-0007 ("PiCode executes with the user's permissions, like Pi itself").

### 1.2 Account model

```json
{
  "id": "acc_…", "provider": "anthropic", "kind": "oauth",
  "label": "Work", "hint": "sk-ant-…3f2a",
  "identity": { "email": "…", "plan": "Max 5x" },
  "secret": { "access": "…", "refresh": "…", "expires": 0 },
  "paused": false,
  "health": { "state": "ok|invalid|expired|no_credit|unknown", "at": "…", "message": "…" },
  "origin": "vault|imported:claude-code|migrated",
  "createdAt": "…", "lastUsedAt": "…"
}
```

Vocabulary is the roster's: `provider` is a pi provider id (the existing
`catalog.LoginMethods` set), so `internal/usage` adapters, plan windows and
identity keep working unchanged. `identity_key`/`disabled` from omp's schema
become `identity` and `paused` — fields PiCode can render, without omp's
rotation policy (ADR-0058 left auto-switch to the owner; nothing here
changes that).

### 1.3 Per-CLI declaration

New package (`internal/clicreds`), one declaration per CLI in the
`clisettings`/`climemory` house pattern — the vendor's own names, a golden
test, and nothing invented:

```go
{ id: "claude-code",
  providers: []entry{
    { provider: "anthropic",
      env:   {apiKey: "ANTHROPIC_API_KEY", oauth: "CLAUDE_CODE_OAUTH_TOKEN"},
      dir:   &dirSpec{envVar: "CLAUDE_CONFIG_DIR", vendor: ".claude",
                      credential: ".credentials.json",
                      seed: []string{".claude.json", "settings.json", "mcp.json",
                                     "plugins", "skills", "projects", "history.jsonl"}},
      verify: probeSpec{provider: "anthropic"} } } }   // no `claude auth status` exists
```

A CLI that does publish a status command declares it instead — Codex
(`codex login status`), Hermes (`hermes auth status`), OpenCode
(`opencode auth list`) — and pi keeps `pi auth check`.

`dir` is optional per CLI and kind — Codex needs `CODEX_HOME` for its
ChatGPT login, Claude Code has an env token and may not need a dir at all.
A CLI with no declaration renders a blocked pane, not an empty one.

### 1.4 Native logins are rows, not shadows

Detection (read-only) per declaration: does `~/.claude/.credentials.json`
exist, `$CODEX_HOME/auth.json`, `~/.grok/auth.json`, … A detected login
becomes a roster row with `origin: imported:<cli>` and exactly one action —
**Import into the vault** — which reads it once, stores the copy, and never
writes back to the vendor's file. PiCode remains unable to break a login made
outside it (ADR-0013's `syncFromAuth` import already works this way).

### 1.5 Migration

First boot after the change reads `~/.picode/accounts.json` (if present),
maps each row into the vault (`provider`, `kind`, `label`, `secret`,
`paused`, `origin: migrated`), writes the vault, and leaves the old file
untouched on disk, unused. Pi's active slot is untouched: `auth.json` stays
the file pi reads, and "Use" on a pi row keeps writing it (ADR-0013's
semantics, now fed by the shared vault). No ADR-0013 behavior changes for pi
except where the extra accounts live.

### 1.6 The pane

`#/clis/<cli>/providers` becomes real for all nine (the capability list
`web/shared/domain/cliProviders.js` grows from `pi` to every launchable CLI;
ADR-0103's Pi-only scope is amended, not superseded).

Roster per provider the CLI can read, one row per account:
`label · identity · kind chip · source chip · health · usage strip · actions`.

| Element | Spec (adapted from ADR-0058's settled row) |
|---|---|
| Source chip | `vault` / `native` / `environment` — the credential's origin is displayable |
| Kind chip | `Subscription` / `API key` — different first-class row types, labelled |
| Usage | only where `internal/usage` has an adapter for that provider; otherwise nothing, never a synthetic zero |
| Actions | **Sign in** (guided, step 2), **Add API key** (step 1), **Import from \<CLI\>** (when native), **Use for new terminals** (default binding), **Pause**, **Sign out** (blast radius: how many terminals are bound), **Verify** |
| Empty | "No accounts for this CLI yet." + Add API key / Import |
| Blocked | CLI declares no credential mechanism: one line + the CLI's own docs link |
| Error | failed catalog read: one line + Retry, never a false empty |

Both apps in the same branch (ADR-0072); shared geometry in
`web/shared/styles/providers.css`, which already exists.

### 1.7 API and events

`GET /api/credentials?cli=` (no secret material, ever), `POST
/api/credentials` (api key), `PATCH` (label), `POST …/{id}/pause|resume`,
`POST …/{id}/verify`, `DELETE`, `POST …/import` (native). Every mutation
appends its ADR-0048 event in the same transaction — payload carries
`{id, provider, cli, action}` and never a secret, an email or a token. Feed
subscribers refresh the roster; no polling timer.

## Step 2 — guided use (injection)

### 2.1 Binding lives where launch settings already live

`clilaunch.Config` and `Overrides` gain `Credential string` / `*string`:
CLI defaults and per-terminal overrides resolve through `Resolve`, appear in
the launch preview with their origin (`CLI defaults` / `Terminal override`),
persist in the existing `cli_configs`/`terminal_launches` rows, and ride the
existing `cli.updated`/`terminal.launch` events as an id. Empty means
**native**: the CLI's own login, untouched — so nothing changes for anyone
who never opens the pane.

### 2.2 Resolution at launch

1. terminal override → 2. CLI default → 3. empty (native). An id that is
missing, paused or (on this machine) undecryptable is **a refused launch with
a named problem** in the preview — the same shape as a missing executable.
Silently launching with the wrong account is the one outcome that is worse
than not launching.

### 2.3 What each CLI can accept (the honest matrix)

Per (CLI, kind), derived from the study's probed stores and declared per CLI;
the pane renders each row as *injectable*, *import-only* or *blocked* rather
than pretending every CLI takes every credential:

| CLI | API key | Subscription / OAuth | Mechanism |
|---|---|---|---|
| pi | env or the `auth.json` slot (ADR-0013, unchanged) | `auth.json` slot (unchanged) | native |
| claude-code | `ANTHROPIC_API_KEY` | `CLAUDE_CODE_OAUTH_TOKEN` (minted by `claude setup-token`) | **env** — zero file state, no keychain dependency, no refresh race |
| codex | `OPENAI_API_KEY` | **`CODEX_HOME` account dir only** — a ChatGPT login cannot be expressed by an env var | dir (seed `config.toml`, `skills/`, `memories/`; sessions and logs regenerate) |
| opencode | `{PROVIDER}_API_KEY` | its own store → import-only | env |
| omp | provider env vars (models.yml outranks env for some ids) | its own pool → import-only | env |
| grok | `XAI_API_KEY` (fallback — a session token wins) | `GROK_HOME` account dir; the store hot-reloads a written file | env / dir |
| hermes | provider env vars and `.env` | its own pool (`hermes auth add`) → import-only | env |
| muse | nothing found beyond its own file → import-only | import-only | — |
| agy | nothing confirmed → import-only | import-only | — |

Mechanics, once a credential is injectable:

- **Env first** where the vendor ships one for that kind: the launcher
  appends it after `dropEnv` so an inherited value loses, `ManagedEnv` names
  the key, and the value never enters `Snapshot`, the preview or an event.
- **Per-account directory** where the vendor only reads a file:
  `<DataDir>/cli-accounts/<cli>/<accountID>/` (0700) holds the credential
  file (0600, written from the vault at launch) plus symlinks for the
  declared `seed` entries — so settings, sessions, memory and MCP survive
  while the identity changes. Seeding is declared per CLI; an entry the CLI
  rewrites by atomic replace is copied, not symlinked.
- The vendor's real home is never written: a launch made outside PiCode keeps
  the login it always had.
- Maintenance and lifecycle runs (`clijob`, update checks) are not credential
  clients; injection stays scoped to terminal launches.

### 2.4 One rotating refresh token, one consumer

The study's §1b rule, because it decides the model rather than the code:
Codex and Hermes rotate refresh tokens **single-use** (a second consumer gets
`relogin_required`), Claude Code rewrites its store hourly, Grok hot-reloads a
written file, and Google's refresh tokens are reusable. So an account has
exactly one owner:

- **native** — the CLI's own store owns it. PiCode reads it for the roster,
  may import a *snapshot*, and never injects it. This is the launcher's
  default (no binding).
- **vault** — a PiCode account dir owns it. The vault's copy is the seed, and
  the daemon harvests the file back when the terminal stops (and at boot for
  an account dir whose file is newer), silently: a token refresh is not a
  user action and does not belong on the change feed. A conflict keeps the
  copy with the later expiry and says so once on the terminal's row.

Importing a rotating-refresh login is therefore an **adoption** decision, not
a copy: the pane offers it, names the consequence ("your own Codex will be
signed out the first time PiCode refreshes this login"), and a person who
wants both keeps them by logging in twice. That consequence is the vendor's
token semantics, not a PiCode limitation.

### 2.5 Guided sign-in

**Sign in** on a subscription provider opens a normal PiCode terminal
(ADR-0069) for that CLI with the account dir injected and the vendor's own
login command, so the person logs in where the vendor expects — no PiCode
re-implementation of a vendor's OAuth. The pane shows *waiting for the
login…* with **Check now**; when the credential file appears it is imported,
verified and the row is renamed from the vendor identity. Nothing is
auto-clicked, and the flow works over the phone (ADR-0044) because it is an
ordinary terminal.

### 2.6 Terminal identity

The terminal row and the launch snapshot carry the **label**, never the email
and never a secret: `claude-code · Work`. The roster owns identity; a paired
phone opening a terminal list learns "Work", not an address.

### 2.7 Verify

Cheapest answer first, declared per CLI: the vendor's own status command where
one exists — `pi auth check --json --no-refresh` (shipped),
`codex login status`, `hermes auth status [provider]`,
`opencode auth list` — and where none does (Claude Code's `/status` is
in-REPL only; Grok, Muse and Antigravity publish nothing), one minimal probe
on an explicit click, on ADR-0129's amendment terms: one request, the cost
named on the button, the answer body discarded. Results cache with their age
and a stale result is labelled stale; a probe never sits on a launch path.

## Phases

| Phase | Deliverable | Files |
|---|---|---|
| **P1 — vault** | envelope crypto + key file + atomic writes + migration from `accounts.json` + store API with tests | `internal/credentials/` (new), `internal/backup/` |
| **P2 — declarations** | per-CLI credential declarations, native detection, verify, API + events | `internal/clicreds/`, `internal/server/` |
| **P3 — pane** | roster for all nine CLIs, add key, import, pause, sign out, verify, empty/blocked/error states | both apps' providers components, `web/shared/domain/cliProviders.js`, `providers.css` |
| **P4 — step 1 close** | architecture file, docs-site guide, changelog fragment, `make close` | `docs/` |
| **P5 — binding** | `Credential` on launch Config/Overrides, resolution, preview origin, refusal states | `internal/clilaunch/`, `internal/server/cli_plan.go`, launch editors |
| **P6 — injection** | env layer + account dirs with seeding; snapshot/managed-env rules | `internal/server/cli_launch.go`, `internal/clicreds/` |
| **P7 — guided login + harvest** | login terminal, Check now, harvest on stop | `internal/server/`, pane |
| **P8 — identity and docs** | terminal row label, architecture + docs-site, close | `docs/`, both apps |

P1–P3 are useful on their own (one place to see and hold every credential,
and the roster answers "which account is this" machine-wide); P5–P7 are the
part the owner called step 2. Inside P6, API keys and env-kind accounts land
first — they carry no ownership question — and directory-owned subscription
accounts (Codex, Grok) follow only after §2.4's rule is agreed in writing.

## Decision table

| Conditions | Action | Verified by |
|---|---|---|
| CLI has no credential declaration | Pane blocked: one line + docs link; no add controls | browser: blocked |
| Provider not readable by that CLI | Row absent (not an error) | Go: declaration projection |
| No accounts, no native login | Empty state + the two actions | browser: empty |
| Native login detected | `native` row, actions Import only; the vendor's file is never written | Go + browser |
| API key added | Row appears, hint masked, Verify offered, event carries id only | Go + browser |
| Subscription account, no login yet | Sign in opens the login terminal; row stays "waiting" until the file appears | browser with fixture CLI |
| Verify 401 / expired / rate limited | Health word + vendor message in one line + the fix | Go (intercepted) + browser |
| Account paused, bound to a launch | Launch refuses with a named problem; pane offers another | Go: prepare refuses |
| Account deleted while a terminal runs | Running terminal keeps working; next launch refuses | Go |
| Bound account's secret unreadable (key file lost) | Row reads "secrets unavailable on this machine"; launches refuse; re-add or restore the key file | Go |
| Daemon env sets `ANTHROPIC_API_KEY`, a vault account is bound | Injected value wins (`dropEnv`); Snapshot shows the key name, never the value | Go: env assertion |
| Nothing bound | Native/ambient behavior — byte-for-byte today's launch | Go: existing tests stay green |
| Two terminals of one CLI, two accounts | Allowed where the declaration has env or a dir; refused with the reason otherwise | Go |
| Importing a rotating-refresh login (Codex, Hermes) | Import is a snapshot; the first launch that *uses* it takes ownership and the pane names the consequence before that click | Go + browser |
| CLI refreshes tokens in the account dir | Harvest on stop updates the vault silently; expiry on the row follows | Go: fixture CLI rewrites its file |
| CLI's own login refreshes a token the vault also holds | The vault copy is marked stale, not silently retried; the row offers Sign in again | Go |
| Binding changed while a terminal runs | Applies to the next launch; the pane says so | browser |
| Old snapshot restored (has `accounts.json`) | Migration absorbs it on boot; no second reader | Go: restore fixture |
| `credentials.key` missing, vault present | Secrets unavailable, accounts listed, nothing invented | Go |

## Verification

1. Go: envelope round-trip, tamper → refuse to decrypt, atomic replace, 0600,
   migration from a real `accounts.json` fixture, declaration projection,
   `dropEnv` precedence, refusal rows, harvest round-trip with a fixture CLI
   that rewrites its credential file, and a HOME-swapped integration launch
   asserting the injected env **and** that no secret value reaches
   `Snapshot`, the preview, the attempt record or the feed.
2. Browser fixture `scripts/qa-cli-credentials.mjs` on a `qa-scratch.sh`
   instance (fixture CLIs, fixture HOME): empty, blocked, populated, paused,
   deleted-while-running, verify states, binding control, narrow + mobile 390.
3. Screenshots read with the `read` tool (`var/screenshots/`), overlay audit
   `ok`; the five-question visual card answered in the closing reply.
4. Real vendor logins stay out of automated tests: one manual end-to-end
   (one API key + one subscription account) on a scratch instance, reported
   in the handoff note.

## Out of scope

- Proxying or routing model traffic, and any rotation the user did not click
  (ADR-0003, ADR-0058).
- A managed/non-terminal runtime for guest CLIs (ADR-0091 stands).
- Per-agent credential pinning for pi beyond the terminal binding.
- Importing credentials from a different machine, or syncing them anywhere
  off-box.
- A passphrase-locked vault, keyring backends and key rotation (named as
  follow-ups; the open questions decide whether any of them belongs).

## ADRs to write

1. **`credentials-vault`** — Boundary: *persistence* — a new
   `<DataDir>/credentials.json` + `credentials.key`, superseding ADR-0013's
   `accounts.json`; *security model* — encryption-at-rest envelope and its
   honest scope, redaction, backup interaction.
2. **`credential-injection`** — Boundary: *process* — the launcher sets
   vendor auth env vars and per-account config dirs for CLI launches;
   *security model* — an account is pinned at launch, no mid-session swap,
   harvest rules, what is never written.

ADR-0103's Pi-only capability is amended by the implementation (no new ADR);
ADR-0058's "what this does not do" is respected — nothing switches by itself.

## What shipped (step 1, 2026-09-20)

The owner approved all six recommendations (encrypted vault with a key file
beside it; concurrency staged by kind; pi folded into the same vault; harvest
— step 2; import adoption for rotating logins; one account per provider across
CLIs). Step 1 landed as ADR-0165 with:

| Piece | Where |
|---|---|
| Encrypted store, key file, atomic writes, `ErrLocked`/`ErrCorrupt`, ADR-0013 absorption | `internal/credentials/` |
| pi's `auth.json` slot and the public getters, now delegating to the vault | `internal/catalog/accounts.go`, `internal/catalog/cred.go` |
| Per-CLI declarations (nine CLIs) and the readers for each vendor's own login | `internal/clicreds/` |
| Roster + add/import/rename/pause/delete/verify | `internal/server/credentials.go` |
| Pane for the eight guest CLIs, both apps | `web/{browser,mobile}/src/components/CliCredentials.jsx`, `web/shared/styles/credentials.css`, `web/shared/domain/credentials.js` |
| Vault travels in a secrets snapshot; the key never does | `internal/backup/snapshot.go`, `internal/backup/restore.go` |
| Architecture + guide + ADR | `docs/architecture/credentials.md`, `docs-site/guide/providers.md`, `docs/decisions/0165-credentials-vault.md` |

Deliberate deviations from this plan, each with its reason:

1. **No change-feed event for credential mutations.** The provider surface
   that already existed (sign-in, pause, custom providers) has never published
   one, and the pane refetches on the response and on window focus. One event
   kind covering both surfaces is a follow-up; half-covering one of them would
   be worse than neither.
2. **Verify covers API-key rows only.** It spends one listing call to the
   provider (anthropic, openai, openrouter, xai, google, zai, deepseek,
   mistral, groq, cerebras, fireworks, together, nvidia, huggingface,
   kimi-coding, moonshot, minimax, github-copilot) and hides itself where
   there is no honest call. A subscription row's answer comes from its Usage
   window today and from the step-2 terminal sign-in later.
3. **Import covers pi, Claude Code, Codex, Grok, Hermes, OpenCode, Muse and
   Antigravity.** Omp keeps its credentials in a live SQLite database PiCode
   will not read, so its pane manages API keys and says so; so do the
   subscription rows of Hermes, OpenCode, Muse and Antigravity, whose logins
   live only in their own stores.
4. **The vault's key never travels in a backup.** A snapshot carries
   `credentials.json` and not `credentials.key`: a restore on this machine
   works because the key never moved, and a snapshot taken elsewhere cannot be
   decrypted. The pane names the missing file and the action when it sees that
   state.

Step 2 was revised by the owner (2026-09-20) and shipped as **ADR-0166**:
activation is the pi model generalized — **Use** writes the chosen account into
the CLI's own credential file. The plan's per-account directories
(`CODEX_HOME`, `GROK_HOME`, `CLAUDE_CONFIG_DIR`) are **refused**: that variable
moves the CLI's whole home, so settings, sessions and memory would stop being
where the tool expects them. Consequences to keep in mind: one live account per
CLI at a time (switching is machine-wide, like `gh auth switch`), a write is
refused while a terminal of that CLI runs, and the replaced file is kept once
at `<DataDir>/credfiles/<cli>-<unix>.bak`.

Still open after ADR-0166:

- **Harvest** (read a refreshed token back into the vault) — not started; the
  vault copy goes stale after the CLI renews it, and re-importing is the manual
  workaround.
- **Env-var injection at launch** (`ANTHROPIC_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`,
  `{PROVIDER}_API_KEY`) — the declarations carry the names; not wired to the
  launcher yet, and it does not touch HOME.
- **Automatic identity for Claude Code** — the file names no account; the vault
  asks the person to name a second login, and the vendor's profile endpoint
  only fills the row's second line (ADR-0168 keeps the network out of the key).

## What shipped (guided sign-in + identity, 2026-09-20)

Landed as **ADR-0168**, amending 0165/0166 — the two open items above that
gated the owner's "as many accounts per provider as the person has":

- **Sign in** — `POST /api/credentials/signin` opens a terminal running the
  CLI's own login (`clicreds.Spec.Login`: `codex login`, `grok login`,
  `muse login`, `opencode auth login`, `hermes auth add`; TUI logins get a hint
  naming `/login`); the roster carries `signin: {available, hint}`, and
  **Check now** files the result. PiCode performs no vendor OAuth and presents
  no other product's client id.
- **Identity** — a row is named by the store (Codex's `account_id`, Grok's
  `principal_id`, Hermes' `account_id`, Antigravity's `id_token` subject) or by
  the person (`as`, offered when an unnamed row would be replaced; `Store.Adopt`
  re-keys the live row instead of copying it). The vendor's profile endpoint
  (`usage.Identity`) fills the row's second line only — never the key, so the
  roster never calls out to match a file to a row; a named row is matched by
  the token it was saved with.
- **Muse reads and writes both shapes** — the API key and the account login
  (`mechanism: "oauth"`, `access_token`, `expires_at` as *unix seconds* — the
  launcher inside the vendor's binary demands a number — with a carried
  `refresh_token` kept), and a key present beside a login still wins, as the
  CLI itself decides it. An access-token-only login is a whole credential here,
  because the vendor's reader has no refresh token in it.
- **The env-shaped api_key fingerprint is a constant** — llama.cpp's entries
  stopped minting a row per endpoint change (the pi roster showed two active
  "Account 2" rows).

Where the gap remains: a Claude/Muse/pi subscription still keeps one row unless
named; naming is one prompt and the pane says so on the row (`singleOAuth`).

## Open questions (owner) — answered 2026-09-20

Every one of them was decided the recommended way, and the six answers are the
spec step 1 shipped against:

1. **Encryption** — the key-file envelope: `credentials.json` plus
   `credentials.key` beside it, both 0600. A passphrase and DPAPI delegation
   stay named as follow-ups: the envelope leaves room for wrapping the same
   key later.
2. **Concurrency** — yes, staged by kind: API keys are per-launch through an
   environment variable from the start; subscription logins follow where the
   vendor ships a variable (Claude Code's `CLAUDE_CODE_OAUTH_TOKEN`) and
   where only a directory works (Codex) in step 2.
3. **Pi folds in** — one vault for pi and the guests; `accounts.json` was
   absorbed at first start and `auth.json` remains pi's own slot.
4. **Harvest** — read the CLI's refreshed credential back when a terminal
   stops (step 2; the store already carries the tokens to write back).
5. **Import of a rotating-refresh login** — adoption, with the consequence
   named before the click. Codex and Hermes rotate single-use tokens; the
   pane says so on the row rather than pretending a copy survives.
6. **One account per provider** — an account added in one pane is offered in
   every pane whose CLI can read that provider; the roster is one list
   filtered per CLI, not one list per CLI.
