# 2026-09-19 — the connectors cache race that broke `make ci` (feat/connector-cache-race)

`make ci` failed twice for me on
`TestConnectorsGalleryRegistryAfterRefresh` — "refresh: rename
…connectors-catalog.json.tmp … no such file or directory" — and passed on a
re-run. Not my diff, but a real defect that gates everyone.

## Root cause
`mcpcatalog.writeCache` wrote `cachePath()+".tmp"` and renamed it. Two
concurrent writers (the background refresher and a manual `Refresh`) shared
that one path: the slower rename found the file already moved, and the
error-path `os.Remove(tmp)` could also delete the other writer's file.

## What landed
- `writeMu` serializes the write; `os.CreateTemp` gives each write its own
  tmp file (so a second process on the same data dir cannot steal it either).
- Regression test `TestConcurrentRefreshesWriteTheCacheWithoutRacing`: eight
  concurrent refreshes, then the cache must parse and no `.tmp` may remain.
  **Falsified**: with the old implementation it fails 4/20 runs with the very
  message CI printed; with the fix it passes 5/5 (`-count=5`).

## Debts
- None new.
