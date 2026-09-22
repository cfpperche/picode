# ADR-0014: Local directory backup of the PiCode environment

- **Status**: accepted
- **Date**: 2026-08-26

## Context

PiCode runs on one machine. SQLite + WAL, pin files, and pi session JSONL
can vanish to corruption, a Windows lock leftover, or `rm`. Time Machine
does not take a consistent SQLite snapshot. Cursor checkpoints restore a
turn, not the environment.

V1 is a user-chosen local folder (same disk, other volume, external HD).
Remote buckets are a later transport of the same artifact.

## Decision

PiCode writes **inspectable snapshot directories** under
`<dest>/picode-backup/<stamp>/` with a `manifest.json`.

- The live DB is copied with `VACUUM INTO` (never a raw copy of WAL).
- Unchanged files hardlink to the previous snapshot.
- Interval and retention live in Preferences (defaults 1 hour / 10 days).
  A destination alone does not start the schedule — **Schedule** must be on.
  **Backup now** is a one-shot and does not enable the schedule.
- Sessions and secrets are included by default and can be toggled.
- Project folders, `work/`, GGUF, npm cache, and TLS certs stay out.
- Restore stops agents, refuses a newer schema, and leaves omitted
  files (no-sessions / no-secrets snapshots) untouched.

## Consequences

- **Easier**: disaster recovery without a third-party tool; `ls` the HD.
- **Harder**: dest on the same filesystem does not survive a dead disk
  (UI warns). Hardlink-less volumes copy fully.
- **If wrong**: the snapshot format is versioned (`format: 1`); a later
  restic/S3 transport can ingest the same tree.

## Alternatives considered

- **Copy `picode.db` while open**: rejected — WAL makes a mute backup.
- **Syncthing the live `~/.picode`**: rejected — syncs a mid-write WAL.
- **restic in V1**: rejected — extra binary and UX; right for remote V2.

## Amendment 2026-09-22 — every agent CLI's files (ADR-0179)

A snapshot now carries the other agent CLIs' own files under `clis/<cli>/`,
with the same three switches as Pi: each CLI's machine settings file always,
its declared login file only with secrets (0600), and its file-based session
directory only with sessions (Claude Code, Codex, Grok, Omp, Muse Code).
Paths come from each CLI's declarations (`clisettings.UserFiles`,
`clicreds.CredentialFiles`, `clisession.FileSessionRoots`), never from a
hard-coded list here. Session stores kept in SQLite (OpenCode, Hermes Agent,
Antigravity) are not copied: a file copy of a live database is the mute
backup this ADR rejected for `picode.db`. Restore puts each part back under
`$HOME` with the same swap helpers as Pi's. Cost stated: the first snapshot
with sessions on can be gigabytes (measured on the owner's machine: ~4.6 GB
for Claude Code, Codex and Grok); later ones hard-link unchanged files.
Security stated: with secrets on, a snapshot now holds other vendors' tokens
in the clear (as it already held Pi's `auth.json`); the vault's key still
stays behind.
