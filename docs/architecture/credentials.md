# Credentials (ADR-0165)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

One vault holds every provider account this machine has — for pi and for the
eight guest agent CLIs. It replaces ADR-0013's plaintext `accounts.json`:
`internal/credentials` reads that file once at boot, absorbs its rows
(`origin: migrated`) and never writes it again. The old file stays on disk, so
a snapshot from an older release restores into the same migration instead of
being ignored.

## The store

`<DataDir>/credentials.json` is an AES-256-GCM envelope; the random 32-byte key
lives beside it in `<DataDir>/credentials.key`. Both are 0600, written by
`writeAtomic` (temp file in the same directory, then rename), and the daemon is
the only writer. `credentials.For(dir)` returns the process-wide store for a
directory — one mutex per directory, so two callers cannot interleave a
read-modify-write — and `credentials.Default()` resolves `$PICODE_DATA`, else
`~/.picode` on **every call** (a test that points `HOME` elsewhere must get
that vault; caching the directory once leaked rows between tests and is gone).

Three unreadable states, all reported rather than papered over:

| State | Store returns | What the pane says |
|---|---|---|
| File present, key missing | `ErrLocked` | the rows exist and cannot be read here — the key file is what opens them |
| Ciphertext fails to decrypt, key wrong size | `ErrCorrupt` | restore a backup or the matching key; the store refuses to overwrite what it cannot read |
| No file | empty vault | nothing yet (the legacy file is absorbed first when present) |

A save over an unreadable vault is refused (`Update` loads first), so a
corrupt file keeps its bytes for whoever can still open it.

## Rows

Keyed by provider in pi's vocabulary, each row carries the credential in pi's
shape (`{type:api_key,key}` / `{type:oauth,access,refresh,expires,[accountId]}`)
plus: `label` (the person's alias), `email`/`plan` (learned from the vendor,
never typed), `paused`, `hint` (masked at write time — `sk-ant-…f2a`), `health`
with its age, `origin` (`vault` / `imported:<cli>` / `migrated`), and
timestamps. `Fingerprint` deduplicates rows (`accountId` when the vendor gives
one, else the key itself, else one constant for oauth, whose tokens rotate).

`Slot.Active` is pi's slot and nothing else: `internal/catalog` still owns
`~/.pi/agent/auth.json` and remains its only writer (`ActivateAccount`,
`RemoveAccount`, `PauseAccount` promote into it). The credential API never
activates anything, so adding an Anthropic key in Claude Code's pane cannot
change what pi is using.

## Declarations

`internal/clicreds` states, per CLI and in the vendor's own names: which
providers it can read, the environment variable each kind travels in, and
where the CLI keeps its own login (path, parser format, the config-directory
variable that moves that home, and the entries a per-account directory will
link back in step 2). Step 1 is read-only: `Detect` parses the CLI's file into
a vault-shaped credential for **Import**, and nothing here writes a CLI's file.
A CLI that publishes no credential path (omp keeps its pool in a live SQLite
database; Muse Code and Antigravity document none) declares providers and
env vars only, and the pane shows its `note` where a control would be a lie.

## API

`GET /api/credentials?cli=<cli>` answers the roster: the CLI, its declared
providers with kinds/env/note, the vault's readable state, each provider's
saved accounts (never a secret — `hint` at most), and the CLI's own detected
login when it has one. Writes: `POST /api/credentials` (add an API key, refused
for an unknown provider), `POST /api/credentials/import` (absorb the CLI's own
login), `PATCH …/{provider}/{id}` (rename), `POST …/pause`, `DELETE …/{provider}/{id}`
— all three of the latter delegate to `internal/catalog` so promotion into
pi's slot stays consistent — and `POST …/verify`.

**Verify** spends one listing call against the provider with the stored key
(`probes()` in `internal/server/credentials.go`), on an explicit click whose
button names the cost, and stores the outcome on the row. Classification:
200 → `ok`, 401/403 → `invalid`, 402 → `no_credit`, 429 → `rate_limited`,
anything else → `unknown` with the provider's own words clipped to one line.
A provider with no probe hides the control. No prompt, no completion, no model
traffic (ADR-0003, and ADR-0129's amendment restated).

No change-feed event is published for credential mutations. That is deliberate
parity with the provider surface that already exists (sign-in, pause, custom
providers have never published one): the pane refetches on the response and on
window focus. A single event kind covering both surfaces is a follow-up, not a
half-covered one here.

## Activation (ADR-0166)

**Use** writes the chosen row into the CLI's own credential file — the pi model
generalized. No HOME change, no per-account directory, no config-directory
variable: the CLI is launched exactly as it is, and only the file it reads
differs. `internal/clicreds.RenderLogin` renders one vault credential into that
file's own shape, merging and preserving every key PiCode does not own; a file
that does not parse is never clobbered (the write fails instead).

| Rule | Where it lives |
|---|---|
| Refuse while a terminal of that CLI runs (`409`, the count named) | `handleCredentialActivate` + `liveTerminalsFor` (ADR-0062 presence) |
| Keep the replaced file once, at `<DataDir>/credfiles/<cli>-<unix>.bak` | `keepCredentialBackup` |
| Refuse what cannot be written faithfully (omp, a Grok file with no session, Hermes/Muse API keys, a kind the renderer does not support) | `RenderLogin` returning false → `400` with a reason |
| Write atomically at 0600 | `writeInterceptFile` (temp + rename) |
| The CLI's file is the truth about what is in use | the roster fingerprints what the file holds and marks that row `active` |

`active` therefore means two different things by endpoint, on purpose: in
pi's roster it is pi's `auth.json` slot (ADR-0013), and in a guest CLI's roster
it is "the file this CLI reads holds this account right now". A login made in a
CLI's own TUI shows up as the active row on the next load, and a row PiCode
never activated shows as inactive.

`singleOAuth` on a provider means its login carries no account name (Claude,
Muse, Antigravity), so the vault keeps **one** subscription row per provider
there — ADR-0013's fingerprint rule, said out loud in the pane instead of
letting a second import look like it vanished.

## Backup

A snapshot with secrets carries `picode/credentials.json` and **not**
`credentials.key`: a restore on this machine works because the key never
moved, and a snapshot taken to another machine cannot be decrypted — which is
the point of encrypting at rest. `accounts.json` is still carried when present
(an older release's only credential carrier). Restore writes back whatever the
snapshot has and the migration absorbs the legacy shape.

## Where it lives

| Piece | File |
|---|---|
| Store, envelope, key, atomic write | `internal/credentials/store.go` |
| Rows, operations, fingerprints, hints | `internal/credentials/vault.go` |
| ADR-0013 absorption | `internal/credentials/legacy.go` |
| pi's slot + the public getters usage and the server read | `internal/catalog/accounts.go`, `internal/catalog/cred.go` |
| Per-CLI declarations and native readers | `internal/clicreds/` |
| HTTP surface | `internal/server/credentials.go` |
| Pane (both apps), one surface for all nine CLIs (ADR-0169) | `web/{browser,mobile}/src/components/CliCredentials.jsx` (+ `AddProviderDialog.jsx` for pi's own door), `web/shared/styles/credentials.css` and the roster grid in `providers.css`, `web/shared/domain/credentials.js` |
| Activation renderers + the file path per CLI | `internal/clicreds/render.go`, `internal/clicreds.CredentialPath` |

**Tests must never see the live vault.** Its directory is `PICODE_DATA` first
and `$HOME/.picode` second (`defaultDir`), and the agent runtime exports
`PICODE_DATA` to every PiCode terminal (`internal/rpc/runtime.go`) — so a suite
run from one wrote fixtures into the owner's real vault on 2026-09-21 (30 rows
across 15 providers, reported by the owner as "what are all these accounts?").
Three layers keep it out: `scripts/go-test.sh` runs every package with
`PICODE_DATA` unset, the `internal/server` and `internal/catalog` test mains pin
it to a throwaway directory, and each package has a guardrail test
(`TestSuiteIsVaultIsolated`) that fails if the pin is ever dropped.

Tests: `internal/credentials/credentials_test.go` (encryption round-trip, a
flipped byte, a missing key, migration-once, pause/remove promotion, token
merge, import-vs-activate, hints), `internal/clicreds/parse_test.go` (one case
per vendor shape), `internal/backup/backup_test.go` (the vault travels, the key
does not), plus the server-surface tests in `internal/server/credentials_test.go`.
