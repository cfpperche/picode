# ADR-0165: One credential vault for every agent CLI

- **Status**: accepted (owner approved the plan and its six decisions, 2026-09-20)
- **Date**: 2026-09-20
- **Boundary**: persistence — a new `<DataDir>/credentials.json` beside its key, superseding ADR-0013's `accounts.json`; security model — what encryption at rest does and does not protect, what a backup carries, and what may never reach the browser, the change feed or a log.

## Context

ADR-0013 gave pi a multi-account vault because pi itself stores one credential
per provider: `~/.picode/accounts.json`, plaintext, 0600, with `auth.json` as
the active slot. It served one CLI. The owner's ask is machine-wide: *stop
walking to the provider and to each CLI to log in* — as many accounts per
provider as the person has, key or subscription, under GUI control.

The study (`docs/benchmarks/2026-09-20-agent-cli-credentials.md`) probed all
nine CLIs PiCode launches and found: every one of them keeps a single login, in
its own file, in its own shape; `omp` and `hermes` already keep pools, so
multi-account alone is not the differentiator — a *cross-CLI, GUI-owned* vault
is; refresh tokens differ in kind (Codex's and Hermes' are single-use, Claude
Code rewrites its file hourly, Google's are reusable), so one rotating token
must have exactly one consumer; and WSL2, the machine this product runs on,
has no keyring, which makes every keychain-based design a design that fails
here (measured: no D-Bus session, `secret-service` calls hang or error).

Two steps were agreed. This ADR records step 1 — where credentials live and
what may read them. Injection into launches, per-account directories and the
guided vendor sign-in are step 2 and cross a process boundary; they are not
decided here.

## Decision

PiCode keeps **one** vault at `<DataDir>/credentials.json`, an AES-256-GCM
envelope whose random 32-byte key lives beside it in `credentials.key` (both
0600, atomic replace, the daemon the only writer). ADR-0013's
`accounts.json` is read once at boot and absorbed — rows keep their provider,
kind, label, secret, identity and paused state, stamped `origin: migrated` —
and the old file is left on disk untouched; nothing reads it again.

Rows are keyed by provider (the vocabulary pi already uses), and carry the
credential in pi's shape (`{type:api_key,key}` / `{type:oauth,access,refresh,
expires}`), plus the fields the roster needs: a user label, the vendor's
identity and plan learned from the vendor's own API, `paused`, a masked `hint`
computed at write time, a `health` record with its age, and `origin`
(`vault` / `imported:<cli>` / `migrated`). **Pi's `auth.json` remains the only
active slot pi reads, and the pi provider endpoints remain its only writer** —
this decision does not touch how the agent picks a credential.

A new declaration package (`internal/clicreds`) states, per CLI and in the
vendor's own names, which providers it can read, the environment variable each
kind travels in, and where it keeps its own login (path, format, the
config-directory variable that moves that home). Step 1 uses those
declarations **read-only**: detect, import, and choose what the pane may
offer. PiCode still writes no CLI's own credential file.

The Providers pane serves every launchable CLI: pi keeps its existing editor,
and the other eight get the roster for the providers they can use, with four
verbs — **Import** the CLI's own login, **Add API key**, **Verify**,
**Sign out** — plus rename and pause. The new surface never activates a
credential for pi.

**Verify** is one listing call to the provider with the stored key, on an
explicit click whose button names the cost, and the answer is cached on the
row with its age. No prompt, no completion, no model traffic (ADR-0003,
ADR-0129's amendment restated).

The security model, stated once and enforced everywhere: the vault's key is
not a secret from the person using the machine — it sits beside the data so a
terminal can open without a passphrase — so what the encryption protects is a
**copy**: a backup snapshot uploaded somewhere, a dotfiles repository, a
support bundle, a file another account can read. Secret material never
crosses the API (rows carry a masked hint at most), never enters the change
feed, never lands in a log, a preview or a terminal snapshot. A backup carries
`credentials.json` and **never** `credentials.key`: a restore on this machine
works because the key never moved, and a snapshot taken elsewhere cannot be
decrypted, which is the point. A vault that cannot be read (missing key, wrong
key, tampering) is a state the pane reports in words, and the store never
replaces it — a save over an unreadable vault is refused so the bytes survive
for whoever can still open them.

## Consequences

Easier: one place to see and hold every account on the machine; adding one
account for a provider makes it visible to every CLI that can read that
provider; a wiped CLI profile no longer means visiting the vendor; and the
roster can say which key a row holds (`sk-ant-…f2a`) without ever returning
it. Harder: the vault is now a file PiCode must keep readable across releases
— the format carries a version, an unknown version is refused rather than
rewritten, and a restore from an older release is absorbed by the migration
instead of silently ignored. Uneven by design: support is per CLI and
per kind (Muse Code and Antigravity publish no credential path; omp keeps its
pool in a live SQLite PiCode will not read), and the pane says so in one line
where a control would be a lie.

Who breaks if we are wrong: someone who copies `credentials.key` believing it
is needed for a backup, and then stores that copy next to a snapshot — the
encryption is then only as good as the folder. The mitigation is that the key
file is never copied by the backup engine and the docs say why; a person who
deliberately carries both is making an informed choice. Second: a daemon whose
data directory moved (`PICODE_DATA`) starts with a new vault and reports the
old rows as absent, not as lost — the old file is still on disk.

## Alternatives considered

- **Plaintext 0600, as ADR-0013 shipped** — refused: it held a handful of pi
  credentials; this holds every key and every subscription token for nine
  CLIs, and the snapshot feature copies it by design.
- **A passphrase-locked vault** — deferred, not refused: it would break
  unattended launches and can lock a person out of their own accounts. The
  store's envelope leaves room for wrapping the same key with a passphrase
  later, which is a decision of its own.
- **The OS keychain** (macOS Keychain, Windows Credential Manager, Secret
  Service) — refused for the primary path: it does not exist on WSL2 or on a
  headless server, which is where this product runs, and a vault that fails
  there fails the owner.
- **A local proxy or router holding the pool** (better-ccflare, claude-code-router)
  — refused: ADR-0003 keeps the agent talking to the vendor directly, and a
  proxy is the only way to rotate mid-request, which this product does not do.
- **Rows in SQLite** — refused: ADR-0005 keeps the database for orchestration
  data, and every credential on this machine already lives in a file its owner
  can read, move or delete.
- **A second store per CLI, leaving pi's alone** — refused: the same account
  would be added twice, and "which account is this" would have two answers.
