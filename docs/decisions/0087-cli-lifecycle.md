# ADR-0087: CLI lifecycle — update check, update, reinstall, uninstall

- **Status**: accepted (owner approved the phased plan, 2026-09-06)
- **Date**: 2026-09-06

## Context

The Agent CLIs surface (ADR-0069) shows installed CLIs and a setup check, but
the user cannot see whether a newer CLI release exists, nor update, reinstall
or uninstall a CLI without a terminal. ADR-0069 drew the boundary "PiCode does
not implement other CLIs' package managers". Investigation (2026-09-06, local
verification plus vendor docs) shows every catalogued CLI ships a native
lifecycle mechanism, so PiCode can orchestrate without becoming a package
manager:

| CLI | Install method (detected) | Check | Update | Reinstall | Uninstall |
|---|---|---|---|---|---|
| pi | npm global | npm registry vs `--version` | `pi update` | `pi update --force` | `npm rm -g` (npm-managed only) |
| claude-code | native (versions dir) or npm | npm registry (`@anthropic-ai/claude-code` tracks native) | `claude update` | `claude install` | vendor-documented per method; guided |
| codex | npm global | npm registry | `codex update` | npm `install -g pkg@latest` | `npm rm -g` (npm-managed only) |
| grok | vendor installer (`~/.grok`) | `grok update --check --json` | `grok update` | `grok update --force-reinstall` | none exists; guided |
| hermes | git + venv (`~/.hermes`) | `hermes update --check` | `hermes update --yes` | `hermes update --force --yes` | `hermes uninstall --yes` |

Options considered: (a) keep the boundary — users stay with manual terminal
work; (b) shell out to npm for everything — breaks for native/grok/hermes and
inherits npm's environment assumptions; (c) orchestrate vendor-native
lifecycle commands behind PiCode jobs.

## Decision

Extend ADR-0069 with a lifecycle layer in `internal/clilifecycle`. PiCode
orchestrates the vendors' own commands; it still does not implement their
package managers. Detection classifies the resolved executable's realpath:
`npm`, `native`, `vendor`, `git`, `unknown`. A check compares the installed
version against a latest-version source (npm registry reusing
`pipkg.npmLatest`/`Newer`, or vendor `--check` output) and stores
`Latest`, `UpdateAvailable`, `UpdateCheckedAt`, `InstallMethod` in the
existing `cli_checks` JSON diagnostic. Update and reinstall run the vendor
command for the detected method; uninstall runs a vendor/npm command when one
exists and otherwise shows a guided uninstall (detected method, exact
commands, docs link) — grok and claude-code native land there.

Mutating lifecycle operations are durable jobs modeled on ADR-0083:
`cli_jobs` table, HTTP 202, request-key idempotency, store writes and
`cli.job` feed events in one transaction, one lifecycle job at a time,
bounded combined-output tail, restart recovery marks jobs `interrupted` and
never replays an install. A lifecycle job refuses while live terminals of
that CLI run, unless the caller confirms with the terminal count. Uninstall
additionally requires typing the CLI name (same pattern as terminal remove).
`unknown` install method offers no mutation buttons — only the docs link.

The UI subscribes to the feed: an update badge on list rows, a version line
with latest/checked-at in the detail header, inline job progress with the
last output lines, and one-line blocked states (managed externally,
interrupted, check failed). Check refreshes on demand and opportunistically
server-side when the stored check is older than six hours and the surface is
opened; there is no browser polling timer.

This adapts t3code's explicit waiting states and reload-safe job routes from
the benchmark study (`docs/benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md`),
the same adaptation ADR-0083 made for llama operations.

### Decision table

| Conditions | Action / observable result |
|---|---|
| Installed version < registry/vendor latest | Badge and detail line show update available with the version |
| Versions equal, or either side unparsable | No badge; check still records latest/checked-at |
| Registry or vendor check unreachable | Stores checked-at with error; UI shows one-line "couldn't check" — never invents state |
| Install method unknown | Read-only view + docs link; no lifecycle buttons |
| Live terminals of the CLI + update/uninstall without confirmation | 409 naming the count; nothing runs |
| Live terminals + confirmed update | Job runs; terminals keep the old binary until restarted (UI says so) |
| Daemon restarts mid-job | Job becomes `interrupted`; no replay; UI offers Check result |
| Vendor command fails | Job `failed` with bounded output tail; install state unchanged |
| Uninstall without typed confirmation | Refused |
| Vendor uninstall exists (hermes; npm for pi/codex) | Runs vendor/npm command; config/auth data untouched unless the vendor command itself removes it |
| No uninstall command (grok, claude native) | Guided uninstall only |
| Job requested while another lifecycle job runs | 409; existing job keeps running |
| Repeated request with same request key | Same job returned (idempotent) |

## Consequences

Easier: users manage CLI versions without a terminal; the surface stays
honest because every state is observed, never inferred. Harder: PiCode now
depends on vendor command surfaces (`pi update`, `grok update --json`,
`hermes update --check`) that can change between releases — adapters pin argv
and tests assert against recorded `--help` output, with a dated note. npm
registry data can lag native Claude releases by hours; the UI labels the
source. If wrong, the failure mode is a failed job with the vendor's own
error output — installs are never half-applied by PiCode because the vendor
command owns the filesystem. Windows-native hosts (outside WSL) are out of
scope for this ADR; adapters are command-based so the debt is path handling.

## Alternatives considered

- **Keep ADR-0069's boundary unchanged**: users keep doing terminal work the
  surface exists to remove; refused by the owner.
- **Drive everything through npm**: wrong for native claude, grok, hermes;
  also assumes npm is on PATH with a writable prefix. Lost.
- **Parse vendor update APIs directly (e.g. grok's release feed)**: duplicates
  vendor logic PiCode would have to track; the `--check` commands already
  encode channel/installer specifics. Lost.
