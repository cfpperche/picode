# Study: credentials for every agent CLI — one vault, many accounts

- **Date:** 2026-09-20
- **Sources:** web sweeps on 2026-09-20 (vendor docs, open repos, issue
  threads — links inline and in **Sources** at the end); **primary evidence
  read on this machine**:
  `~/.pi/agent/auth.json` (keys only, never values — 12 providers, shapes
  below), `~/.omp/agent/` (no `auth.json`; credentials live in
  `agent.db`, schema read from `sqlite_master`/`pragma_table_info`, row
  counts only), and the omp docs bundle (`omp://providers.md`,
  `omp://models.md`, `omp://auth-broker-gateway.md`, `omp://secrets.md`,
  `omp://environment-variables.md`). Repo receipts: ADR-0013, 0031, 0058,
  0069, 0103, 0129, 0163 and the 2026-09-03 providers study.
- **Scope:** where each of PiCode's nine agent CLIs keeps credentials, how
  the field does multi-account, what is safe to store at rest on WSL2, and
  the BYOK/roster patterns a GUI vault must match. Not the model catalog,
  not quota adapters, not the Pi-only RPC surface.

## The problem, in PiCode terms

The owner's words: *"several times I have to go to the LLM provider, log in
to the CLIs…"*. Every login today happens inside a CLI's own TUI, one
account at a time, in a file PiCode does not know about:

| Today | Consequence |
|---|---|
| PiCode's vault (ADR-0013) exists for **pi only** — `~/.picode/accounts.json`, `auth.json` as the active slot | A second account exists for `pi`, but Claude Code, Codex, OpenCode, Grok, Hermes, Muse and Antigravity each keep their own single login |
| CLI launches inherit the daemon environment wholesale (`cliEnvironment`, `internal/server/cli_launch.go:390`) | The only cross-CLI credential channel today is an env var typed into `picode install --env`, which is one global slot for every CLI and every terminal |
| `#/clis/<cli>/providers` is a real pane for **pi** and a "coming soon" placeholder for the other eight (`web/shared/domain/cliProviders.js:2`) | The GUI that was built for exactly this question answers it for one of nine CLIs |
| Each CLI's login state is a file or a DB in `$HOME` | Re-login after a wiped profile, a second machine, or a vendor logout means visiting the vendor again |

PiCode is in an unusually good position: it **starts the processes**
(ADR-0069), it already owns the launch environment (`clilaunch.Config.Env`,
`tmux -e`, ADR-0070's plan/inspection), it already writes vendor config files
with merge-by-key discipline (ADR-0150 connectors, ADR-0163 settings), and it
already keeps a multi-account credential store with a roster, identity,
pause, sign-out blast radius and verify (ADR-0013/0058). What is missing is
scope (nine CLIs, not one), multi-account on the CLIs' side, and an injection
layer.

## What PiCode already has (receipts)

| Piece | Where | What it gives us |
|---|---|---|
| Provider vault, multi-account, active slot | `internal/catalog/accounts.go` (`~/.picode/accounts.json`, 0600; `ActivateAccount` writes pi's `auth.json`) | The account shape, the roster semantics, `OAuthCred`/API-key kinds, `syncFromAuth` import by fingerprint |
| Provider roster: identity, plan, quota, source badge, pause, verify, blast radius | ADR-0058, `web/browser/src/components/Providers.jsx`, `internal/usage/` | The row spec is settled; `usage` already holds 26 vendor endpoints and reads the **active slot per provider**, not per CLI |
| Guest-CLI credential reads | `internal/usage/grok_cli.go` (`GROK_HOME` → `~/.grok/auth.json`, else `GROK_COOKIE`) | Precedent: PiCode already reads another CLI's credential file to answer a quota question |
| Per-CLI declaration + one generic driver | `internal/clisettings/specs.go` (`spec{id, layers, fields}`, `Field.Secret`), `internal/climemory/specs.go` | The house pattern for "declare per CLI, drive generically"; `Field.Secret` already redacts a credential-shaped config value |
| Launch env assembly and Tmux delivery | `internal/server/cli_launch.go:390` `cliEnvironment`, `:404` `dropEnv`, `:1003` `prepareCLITerminal`, `internal/tmux/tmux.go:359` (`-e` per var) | The exact seam an injection layer writes into; `dropEnv` already solves "an inherited value must lose to ours" |
| Per-terminal env injection that is not user-editable config | `pi` spawn env merge and `launchIdentityEnv` (`internal/server/cli_launch.go:964`) | PiCode already injects secrets-adjacent env (`PICODE_*`, agent grants) at launch without persisting values |
| Launcher-owned env vars | `clilaunch.Validate` (`internal/clilaunch/config.go:160`) refuses user-set `HOME`, `SHELL`, `GROK_HOME`, `HERMES_HOME`, `OPENCODE_CONFIG`, `PI_CODING_AGENT_DIR`, `PICODE_*`, `PATH` | The namespaces a credential injector needs are already reserved for the launcher — the user cannot fight it |
| A private durable per-agent dir outside the vendor home | `ompAgentSessionDir` → `<DataDir>/omp-sessions/<agent-id>` (`internal/server/cli_launch.go:984`) | The precedent for "PiCode-owned directory injected into a CLI via a flag/env, vendor home untouched" |
| Redaction rules that must hold | `clilaunch.Snapshot` masks secret-looking argv and reports `EnvKeys` without values (`config.go:186-216`); `climemory/mask.go`; `clisettings` `Secret` | Contract: no credential value in previews, terminal views, change-feed events or logs |
| Backup treats secrets explicitly | `internal/backup/snapshot.go:65` copies `accounts.json` only when `secrets` is on | A vault file has a defined place in backup/restore, and a defined opt-in |
| Legacy import of another tool's store | `internal/catalog/accounts.go:285` `syncFromAuth` | Precedent for "import what the CLI already logged in" |

**Not in the code today:** any write to another CLI's credential file, any
per-account directory for a CLI, any credential binding on a terminal, and
any crypto dependency (`go.mod`: stdlib only — no age, nacl, keyring; `x/oauth2`
is indirect).

## What the field ships

### 1. Where each of the nine CLIs keeps credentials

| CLI | Store | Mode | Auth env vars | Config-dir isolation | Native multi-account |
|---|---|---|---|---|---|
| **pi** | `~/.pi/agent/auth.json` — `Record<provider, {type:api_key,key} \| {type:oauth,access,refresh,expires(,accountId)}>` (12 providers on this machine) | 0600 file, `proper-lockfile` cross-process lock | `--api-key` → auth.json → ~40 provider env vars | `PI_CODING_AGENT_DIR` | **none** — one credential per provider; a login overwrites |
| **omp** | `~/.omp/agent/agent.db` SQLite: `auth_credentials(id, provider, credential_type, data, disabled_cause, identity_key, created_at, updated_at)` **plus** `auth_credential_blocks(credential_id, provider_key, block_scope, blocked_until_ms)` and `auth_credential_refresh_leases`; 3 credential rows on this machine; no `auth.json` exists | DB file in 0700 dir; broker mode keeps tokens off the machine, client cache AES-256-GCM | `--api-key` → models.yml key → OAuth → login key → env (incl. `.env` chain) | `PI_CODING_AGENT_DIR`, `PI_CONFIG_DIR` | **full** — "multiple accounts are ranked and rotated automatically"; per-credential blocks; optional auth broker |
| **claude-code** | `~/.claude/.credentials.json` (Linux/Windows); macOS Keychain ("Claude Code-credentials") | file elsewhere, keychain on macOS | `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) | `CLAUDE_CONFIG_DIR` — moves all state incl. credentials | none — one OAuth session |
| **codex** | `$CODEX_HOME/auth.json` (default `~/.codex/auth.json`): `OPENAI_API_KEY` and/or ChatGPT tokens (`id/access/refresh`, `account_id`) | 0600 file (keychain only for MCP OAuth) | `OPENAI_API_KEY`, `CODEX_API_KEY` | `CODEX_HOME` — full isolation incl. auth | none — `[profiles.*]` change model/sandbox, not auth |
| **opencode** | `~/.local/share/opencode/auth.json` (per-provider `{type:api\|oauth,…}`); newer builds moving toward SQLite (`opencode.db` present here) | file | `{PROVIDER}_API_KEY` (e.g. `ANTHROPIC_API_KEY`, `OPENROUTER_API_KEY`), `opencode auth login` | `OPENCODE_CONFIG` (file), `OPENCODE_CONFIG_DIR`, `XDG_DATA_HOME` moves the data dir (and sessions DB) | none |
| **grok** | `~/.grok/auth.json` — map keyed `"<oidc_issuer>::<oidc_client_id>"`, per entry `key` (access JWT), `auth_mode`, `refresh_token`, `expires_at`, `user_id`/`email`/`team_id`; `auth.json.lock` beside it. Read today by `internal/usage/grok_cli.go` | 0600 file, lock sidecar | `XAI_API_KEY` (**fallback only** — a session token wins); per-model `env_key` in `config.toml` outranks both | `GROK_HOME` | none documented (the key structure allows several issuer entries) — **hot-reloads external edits**, so a written file is picked up on the next call |
| **hermes** | `~/.hermes/auth.json` — `providers` **and** a native `credential_pool[<provider>]` (entries with `id`, `label`, `auth_type`, `priority`, `access_token`, `refresh_token`, `base_url`, `request_count`, `secret_fingerprint`); plus `.env` (provider keys) and a separate `.anthropic_oauth.json` | file + lock | provider keys via env or `.env` (`OPENAI_API_KEY`, `GLM_API_KEY`, …); `HERMES_HOME` | `HERMES_HOME` (profiles under `<home>/profiles/<name>/`) | **native multi-credential pool with priority** — the second of nine that ships one |
| **muse** | `~/.config/muse/auth.json` — `{schema_version, providers:{"meta":{api_key}}}`; `settings.json` pins the provider | file + `.auth.json.lock` | none found | none found | none |
| **agy** (Antigravity) | `~/.gemini/antigravity-cli/antigravity-oauth-token` — `{token:{access_token,refresh_token,expiry}, auth_method:"consumer", id_token}`; older `~/.gemini/oauth_creds.json` and `google_accounts.json` also present. Quota lives at `cloudcode-pa.googleapis.com` (2026-09-03 study, debt 2) | file (vendor docs mention a keyring on other platforms) | none confirmed | none confirmed | none |

Grok, Hermes, Muse and Antigravity rows were probed on this machine
(paths, modes, field names — never values); Hermes' pool and both Google
shapes are also in the vendors' own sources/docs. Muse has no public
documentation at all: its row is *observation only* until the vendor
publishes something, and the pane must say so rather than guess.

### 1b. Refresh semantics decide what may be copied

The refresh token, not the access token, is what a vault is really storing —
and vendors differ on whether it survives being copied:

| Store | Access token | Refresh token | A vault copy is… |
|---|---|---|---|
| Claude Code `.credentials.json` | `claudeAiOauth.expiresAt` (ms), rewritten in place on every refresh | `refreshTokenExpiresAt` observed ~21 days; rotation under concurrent consumers unverified | stale within hours unless harvested |
| Codex `auth.json` | expiry inside the JWTs, `last_refresh` on top | **single-use (rotating)** — proven on this machine through Hermes' pool: a second consumer gets `refresh_token_reused`/`relogin_required` | **dead on first live refresh**; copies must not be shared |
| Hermes `auth.json` | per-provider `last_refresh` | single-use for `openai-codex`/`nous`; API-key pool entries never expire | same hazard as Codex for those providers |
| Grok `auth.json` | `expires_at` (ISO) | rotating, but the store **hot-reloads external edits** | copy-friendly, and injectable by writing the file |
| Antigravity / Gemini OAuth | `expiry`/`expiry_date` | Google refresh tokens are **reusable, not rotated** | copy-friendly and long-lived |
| API keys (all) | — | — | static until revoked |

Design consequence, stated as a rule rather than a hope: **one rotating
refresh token, one consumer.** Either the CLI's own home owns an account
(and the vault only *references* it) or a PiCode account dir owns it (and the
vendor's own copy must not be used again) — never both. Env-token kinds
(`CLAUDE_CODE_OAUTH_TOKEN`) sidestep the problem entirely.

**Login shapes:** loopback browser OAuth (Claude Code, Codex, OpenCode),
device code (Copilot; pi/omp implement it for several providers), API-key
paste (OpenCode, Codex `--api-key`, Claude Code env). Refresh tokens are
long-lived and each CLI refreshes in-process, rewriting its own file — which
is why every file-swap tool works, and why a *copy* in a vault goes stale
until it is harvested back.

### 2. pi and omp, side by side (the two named inspirations)

| | pi | omp |
|---|---|---|
| Store | one JSON file, 0600, locked (`proper-lockfile`, 10 retries) | SQLite `agent.db` in a 0700 dir, tables above |
| Accounts per provider | **one** (login overwrites) | **many**, ranked and rotated automatically; per-org/workspace counts as its own account |
| Selection | implicit: the single credential | automatic ranking (`#rankOAuthSelections`, recency/provider priority); no documented manual per-request picker |
| Out-of-play state | delete only | `disabled_cause` per credential + `auth_credential_blocks` windows + refresh leases |
| Reference tricks | `key` may be `!command` (shell), `$ENV`, or a per-credential `env` map | precedence chain incl. `.env` files; `!command` in models.yml |
| Centralization | none | optional **auth broker**: tokens live on a host, clients get redacted snapshots (`REMOTE_REFRESH_SENTINEL`), cache encrypted AES-256-GCM keyed on the broker token |
| GUI | `/login`, `/logout`, `pi auth check --json` | `/login`, `/logout`, `omp auth-broker {serve,token,login,list,import,migrate,status}` |

Takeaway that corrects the brief: **omp already has multi-account and
rotation**; what neither tool has is a *GUI-owned, cross-CLI* vault — one
place that holds accounts for pi **and** the eight guest CLIs and injects the
right one into the right launch. That is PiCode's differentiator, not
multi-account alone.

### 3. How multi-account is done in the wild

| Mechanism | Tools | What it does | What it cannot do |
|---|---|---|---|
| **Credential-file swap** | cc-switch, claude-swap, Codex account switchers | copies one account's file over the live one | single active account; unsafe while the agent runs; breaks on keychain-backed stores (Claude on macOS) |
| **Config-dir / profile swap** | AI Switcher (`aisw`), per-account `CODEX_HOME`/`CLAUDE_CONFIG_DIR` recipes | points the CLI at a whole private config dir | process-global unless the *launcher* sets the env per process; duplicates settings/sessions unless seeded |
| **Env-var auth** | `CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `{PROVIDER}_API_KEY` | per-process identity with no file mutation | only where the vendor ships an env path; not available for ChatGPT OAuth in Codex |
| **Local proxy pool** | better-ccflare/ccflare, opencode-go-multi-auth, claude-code-router (base-URL rewrite) | true concurrent multi-account, rotation mid-request | puts a third party in the token path; ADR-0003 already refuses PiCode-as-proxy |
| **Vault + launch injection (proposed)** | none shipped for these CLIs | account pinned per terminal, no file swap, no proxy | needs a per-CLI declaration and a per-account directory or env var |

### 4. Storage at rest: keychain or file (and what WSL2 actually allows)

- **Linux desktops**: Secret Service over D-Bus (gnome-keyring/KDE Wallet).
  `zalando/go-keyring` hard-depends on it; `99designs/keyring` (what
  aws-vault uses) exposes additional backends — **kernel keyctl** and an
  **encrypted file** backend — which is why it survives headless Linux.
- **WSL2 / headless servers**: no D-Bus session, no keyring daemon by
  default; Secret Service calls fail (`org.freedesktop.secrets was not
  provided`) or hang for minutes (jaraco/keyring#531); VS Code/Electron on
  Ubuntu report *"the OS keyring is not available for encryption"*. The
  documented WSL escape hatches are (a) delegate to the Windows side (Git
  Credential Manager stores in Windows Credential Manager via interop), or
  (b) a file the user's uid can read.
- **The de-facto fallback** across CLIs and tools is a `0600` file:
  `~/.aws/credentials`, `~/.netrc`, `gh`'s `hosts.yml`, pi's `auth.json`,
  `~/.claude/.credentials.json`, `~/.codex/auth.json`, git's
  `credential-store`. aws-vault, pass, sops and 1Password's `op run` exist
  for the stricter end of the market; `op run`'s pattern is *inject at
  launch, never at rest on disk*.
- **App benchmarks at rest**: VS Code `SecretStorage` is OS-crypto via
  Electron `safeStorage` (and degrades with a logged warning where no
  keyring exists); LibreChat encrypts per-user BYOK keys in Mongo with a
  master `CREDS_KEY`/`CREDS_IV` from env — with a documented rotation gap
  (changing the key does not re-encrypt existing rows); Open WebUI derives
  from `WEBUI_SECRET_KEY`; AnythingLLM and Cursor are reported
  instance-global/app-local (unverified).
- **What encryption-at-rest without a passphrase can and cannot do**: it
  protects *copies* — a backup snapshot on cloud storage, a dotfiles repo, a
  support bundle, a file someone else's uid can read. It does **not** stop
  an attacker running as the same user: the key sits beside the data.

### 5. Multi-account UX: what the good ones do

| Source | Pattern worth copying |
|---|---|
| `kubectl config get-contexts` | a `CURRENT` column — "which account am I using" answered by a command, not a habit |
| `gh auth` (multi-account since 2.40) | both accounts retained, `switch` flips the active one, `GH_TOKEN` overrides — and the docs' own complaint is that switching is machine-wide |
| `gcloud config configurations` | named full profiles, `--configuration` per command, `CLOUDSDK_CORE_CONFIG_FILE` override |
| AWS profiles / `AWS_PROFILE` | the profile name *is* the label; per-invocation override; `aws sso login` per profile |
| direnv / mise | per-directory defaults, hook-visible active env |
| 2026-09-03 study (cc-switch, claude-swap, OpenRouter BYOK, Cloudflare alias) | quota inline, three honest states, credential origin displayable, disable ≠ sign out, blast radius named, key-format detection on paste |

### 6. Verification and health

- Vendor status commands where they exist: `pi auth check --json --no-refresh`
  (already wired), `codex login status` (exits 0 when authed),
  `hermes auth status [provider]`, `opencode auth list`. Claude Code has no
  standalone one (its `/status` is in-REPL), and Grok, Muse and Antigravity
  publish none — those get the probe below.
- Minimal probes: OpenAI/Anthropic `GET /v1/models` (401 = invalid),
  OpenRouter `GET /api/v1/key` (validity **plus** credits), Google
  `GET /v1beta/models?key=`. PiCode already ships this class of call in
  `internal/modellist.Probe` (ADR-0129 amendment: one minimal request, on an
  explicit user action, answer body discarded).
- Tools render four honest outcomes — ok / invalid (401) / no credit / rate
  limited — and label a cached result as stale rather than re-checking on a
  blocking path.

## Patterns the field agrees on

1. **One account per CLI is the status quo; a pool is the differentiator.**
   Only omp and Hermes Agent ship multiple accounts natively (omp with
   ranking and blocks, Hermes with a priority pool); every other CLI needs a
   launcher or a swapper to get there.
2. **Config-dir isolation is the safest swap primitive.** It beats file
   mutation (no race with a running agent) and beats a proxy (no third party
   in the token path) — but only when the *process launcher* sets it, which
   is exactly PiCode's position.
3. **Env-var auth wins where the vendor ships it**, because it needs no
   directory and no seeding (`ANTHROPIC_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`,
   `{PROVIDER}_API_KEY`).
4. **Plain `0600` files are what actually runs on WSL2/headless.** Keyrings
   are a desktop luxury; a vault that requires one is a vault that fails on
   the owner's machine.
5. **The credential's origin is displayable** (Zed's "environment variable
   ANTHROPIC_API_KEY" vs "system keychain"; Kilo's source badges) — vault /
   native CLI login / environment are three different facts.
6. **Disable ≠ sign out**, and both are different from "session expired"
   (omp `disabled_cause` + blocks; ADR-0058 Pause; Anthropic's reversible
   Disable).
7. **Identity comes from the vendor, the label is the user's** (email/plan
   read back and cached; omp's `identity_key` is the same idea as a key).
8. **Never invent a number or a state**: quota shows live / stale / a word
   saying which nothing it is (ADR-0058, restated by every honest tool).
9. **Validation happens at entry and on demand, never on a blocking launch
   path** — and the cheapest probe that answers the question is the right one.
10. **Rotation is a policy, not a storage feature** — and swapping under a
    running agent is unsafe in every tool that warns about it.

## What PiCode adapts (the proposal)

**One vault, one roster, one injector — for all nine CLIs.**

- **Store**: `<DataDir>/credentials.json` (0600, atomic replace, the daemon
  is the only writer), holding accounts keyed by **provider** with
  `kind: api_key | oauth`,
  `label`, cached vendor `identity`, `paused`, `health{state,at,message}`,
  `createdAt`, `lastUsedAt` — the shapes omp proved out (`identity_key`,
  `disabled_cause`) expressed as fields PiCode can render. `~/.picode/accounts.json`
  (ADR-0013) migrates in once and is superseded.
- **Encryption at rest**: an AES-256-GCM envelope (stdlib) with a random
  32-byte key in `<DataDir>/credentials.key` (0600), both riding the
  existing backup `secrets` flag together. Honest scope: protects copies,
  not a same-uid attacker. No keyring dependency, no passphrase lockout, no
  new dependency.
- **Declaration per CLI** (the `clisettings` pattern): which providers it
  reads, and for each — the env var(s), the credential file(s), the
  config-dir env var, how to log in, how to verify.
- **Binding**: a `credential` (account id) on `clilaunch.Config` and
  `clilaunch.Overrides`, so CLI defaults and per-terminal overrides resolve
  through machinery that already exists (`Resolve`, preview origins, feed
  events carrying ids only).
- **Injection at launch** (`prepareCLITerminal`): env first, account
  directory second; `dropEnv` semantics so the injected value wins;
  `ManagedEnv` names the key, never the value; a terminal is pinned to one
  account until it restarts (every switcher in the field warns about the
  alternative).
- **Native logins are first-class rows**: detected, labelled `native`, and
  importable into the vault — never silently rewritten.
- **Verify on demand**: the vendor's own status command where one exists,
  otherwise one minimal probe; result cached with its age.

## What PiCode refuses

| Temptation | Why not |
|---|---|
| A proxy or router in the token path (better-ccflare, claude-code-router) | ADR-0003: the agent talks to the vendor directly; PiCode is not in the data path, so it can never rotate mid-request — and must not claim to |
| Swapping a credential file under a running agent | Every tool in the field warns; a running agent and a vault writing the same file is the corruption case |
| Writing into the user's real `~/.claude`, `~/.codex`, `~/.grok` | Those belong to the CLI and to launches made outside PiCode; PiCode writes only its own account dirs, plus pi's `auth.json` where ADR-0013 already does |
| Requiring an OS keyring | It does not exist on WSL2/headless; a vault that fails there fails the owner |
| A passphrase-locked vault by default | Unattended injection at launch would break; it is an opt-in hardening, not the default |
| Auto-switch between accounts without a user click | ADR-0058 left this to the owner explicitly; a visible default is not the same as a silent rotation |
| Scraping browser cookies or vendor dashboards for a second identity | Already refused for Grok (ADR-0031) and re-refused here |
| Putting credentials in SQLite or in the change feed | ADR-0005 is orchestration data only; ADR-0048 events carry ids. The vault is a file, like its predecessor |

## Open questions

1. **Encryption**: key-file envelope (recommended — no dependency, no
   lockout), plaintext `0600` (parity with ADR-0013 and with every CLI),
   passphrase-wrapped, or Windows DPAPI delegation for the WSL case?
2. **Concurrency**: may two terminals of the same CLI run two different
   accounts at once (needs account dirs + seeding), or is one active account
   per CLI enough for v1 (simpler, `gh`-shaped)?
3. **Seeding**: what does an account directory symlink back into the real
   config dir (settings, sessions, memory, MCP) so the isolated profile does
   not lose history — and what happens when the CLI rewrites a symlinked
   file?
4. **Harvest**: after a CLI refreshes tokens inside an account dir, does the
   vault read the file back on terminal stop (keeping the stored copy valid)
   or keep the CLI's file authoritative for that account only?
5. **Pi itself**: does the Pi vault fold into the new store (one vault, one
   migration) and does per-agent pinning (the 2026-09-03 study's open
   question) come with it, or stay separate?

## Sources

- **pi**: [providers.md](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md),
  [auth/types.ts](https://github.com/earendil-works/pi/blob/main/packages/ai/src/auth/types.ts),
  [auth-storage.ts](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/src/core/auth-storage.ts);
  local `~/.pi/agent/auth.json` read for shapes only.
- **omp**: [can1357/oh-my-pi](https://github.com/can1357/oh-my-pi) and the local
  docs bundle (`omp://providers.md`, `omp://models.md`,
  `omp://auth-broker-gateway.md`, `omp://secrets.md`,
  `omp://environment-variables.md`); local `~/.omp/agent/agent.db` schema read
  for column names and row counts only.
- **CLI stores**: Claude Code [IAM docs](https://code.claude.com/docs/en/iam)
  (login methods, credential precedence, `apiKeyHelper`) and the settings
  reference for `CLAUDE_CONFIG_DIR`; Codex `$CODEX_HOME/auth.json` per the
  vendor docs plus a third-party sandbox analysis and issue thread;
  [OpenCode providers](https://opencode.ai/docs/providers) and
  [CLI reference](https://opencode.ai/docs/cli); Hermes' own source
  (`auth_commands.py`, `get_hermes_home()`);
  [Antigravity CLI reference](https://antigravity.google) for the keyring
  wording. Muse Code has no public documentation.
- **Keyring vs file on WSL2**: [99designs/keyring](https://github.com/99designs/keyring)
  (keyctl + file backends), [zalando/go-keyring](https://github.com/zalando/go-keyring),
  [jaraco/keyring#531](https://github.com/jaraco/keyring/issues/531) (minutes-long
  hang), [aws-vault#513](https://github.com/99designs/aws-vault/issues/513),
  [GCM on WSL](https://github.com/git-ecosystem/git-credential-manager/blob/main/docs/wsl.md)
  and [Microsoft's WSL git tutorial](https://learn.microsoft.com/en-us/windows/wsl/tutorials/wsl-git),
  [Electron safeStorage](https://www.electronjs.org/docs/latest/api/safe-storage),
  [VS Code SecretStorage](https://code.visualstudio.com/api/references/vscode-api#SecretStorage),
  [“why most CLIs don't use keyring”](https://news.ycombinator.com/item?id=34653342).
- **Vault patterns**: [age](https://github.com/FiloSottile/age),
  [sops](https://getsops.io), [ansible-vault](https://docs.ansible.com/ansible/latest/vault_guide/index.html),
  [pass](https://www.passwordstore.org),
  [1Password `op run`/`op inject`](https://developer.1password.com/docs/cli).
- **BYOK at rest**: [LibreChat env vars](https://www.librechat.ai/docs/configuration/env_vars)
  (`CREDS_KEY`/`CREDS_IV`), [Open WebUI hardening](https://docs.openwebui.com/getting-started/advanced-topics/hardening),
  [Continue](https://docs.continue.dev).
- **Multi-account UX**: [gh auth switch](https://cli.github.com/manual/gh_auth_switch),
  [GitHub's multiple-accounts guide](https://docs.github.com/en/github-cli/github-cli/using-multiple-accounts),
  [gcloud configurations](https://cloud.google.com/sdk/docs/configurations),
  [kubeconfig contexts](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/),
  [aws-vault](https://github.com/99designs/aws-vault).
- **Switchers**: [claude-code-router](https://github.com/musistudio/claude-code-router),
  [better-ccflare](https://github.com/tombii/better-ccflare),
  [ccflare](https://github.com/snipeship/ccflare),
  [cc-switch](https://github.com/farion1231/cc-switch),
  [claude-swap](https://github.com/realiti4/claude-swap).
- **Health**: [OpenRouter key endpoint](https://openrouter.ai/docs/api-reference/limits),
  [Anthropic API overview](https://platform.claude.com/docs/en/api/overview).
