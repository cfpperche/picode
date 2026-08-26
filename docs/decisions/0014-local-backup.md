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
- Interval and retention live in Preferences (defaults 1 hour / 10 days)
  and stay **off until a destination is set**.
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
