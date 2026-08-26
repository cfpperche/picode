# Backup V1 — local snapshots

ADR: [0014](../decisions/0014-local-backup.md). Preferences owns the UI.

## What is in a snapshot

| Path | When |
|---|---|
| `picode/picode.db` | always (`VACUUM INTO`) |
| `picode/pins/**` | always |
| `picode/accounts.json` | secrets on |
| `pi/settings.json`, `pi/trust.json` | always if present |
| `pi/auth.json` | secrets on |
| `pi/sessions/**` | sessions on |

Out: project folders, `~/.picode/work`, llama GGUF, npm cache, certs, lock.

## Decision table

| # | dest | scheduled | due | dest ok | dest in live tree | sessions | secrets | schema vs app | action |
|---|---|---|---|---|---|---|---|---|---|
| 1 | empty | * | * | * | * | * | * | * | skip |
| 1b | set | off | * | * | * | * | * | * | skip (configure only; Backup now still works) |
| 2 | set | on | no | * | * | * | * | * | skip |
| 3 | set | on | yes | missing | no | * | * | * | fail clean, `last_error` |
| 4 | set | * | * | * | yes | * | * | * | refuse dest |
| 5 | set | on | yes | ok | no | on | on | * | snapshot + hardlink |
| 6 | set | on | yes | ok | no | off | on | * | omit `pi/sessions` |
| 7 | set | on | yes | ok | no | on | off | * | omit `auth.json` + `accounts.json` |
| 8 | * | * | restore | * | * | * | * | newer | refuse |
| 9 | * | * | restore | * | * | omitted | omitted | older/same | restore present files only |
| 10 | * | * | prune | * | * | * | * | * | drop older than keep-days; keep newest |

Same-disk dest is allowed with a warning. It is not a refuse.

## Layout

```
<dest>/picode-backup/<YYYY-MM-DDThhmmssZ>/
  manifest.json
  picode/...
  pi/...
```
