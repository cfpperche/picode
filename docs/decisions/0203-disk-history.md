# ADR-0203: Disk history lives on the Windows side as one small JSON-lines file

- **Status**: accepted
- **Date**: 2026-09-23
- **Boundary**: persistence — a new file the desktop tool writes and the Management window reads, outside the PiCode database

## Context

The Management window measures the disk on demand. The owner asked for
history (item 7 of the WSL Management list): "what grew this week", and the
trend of C:'s free space, which on this machine went from 27 GB to 10 GB
while the distro's disk file reached 227 GB. `docs/plans/wsl-control.md`
sketched P4 as "one row per day" stored by the server behind a new route
carrying Windows facts (its own ADR, never written).

The measurement is made by `picode-desktop` on Windows. Routing it through
the daemon would mean a new authenticated route, a table, and a server that
must be up to record a sample — and the samples matter most when the distro
is full or down.

## Decision

`picode-desktop` keeps `%LOCALAPPDATA%\PiCode\disk-history.jsonl`: one JSON
object per line, **one line per calendar day** (a later scan the same day
replaces that day's line), at most 400 lines (oldest dropped). A line holds
the date, the time, C:'s free and total bytes, the disk file's allocated
bytes, the held bytes, the distro's used bytes and each measured cache's
bytes by id (a cache whose stage failed is absent, and growth never counts
an absent baseline as zero). Every full scan that read both halves records
one; the shell additionally runs a scan in the background when the file was
last written more than 20 hours ago, the daemon answers, no flow holds the
distro and no window action runs (it takes the same one-job lock), bounded
at ten minutes and retried at most every six hours. A background scan's
distro stages are skipped while a flow holds the distro. The
window reads it through `picode-desktop history`, which also computes the
per-cache growth over the last seven days. Nothing else writes the file; a
line that does not parse is skipped, never fatal.

## Consequences

- History exists without the daemon and survives a broken distro.
- The file is per Windows user and per machine — it is not synced, not in
  backups of the PiCode data dir, and not visible from a phone. That is
  the trade for independence from the server.
- Size is bounded (~400 × ~1 KB).
- The daily background scan costs one `du` walk of the home directory
  (~15 s here) plus the root system-cache measure (up to 60 s) per day —
  usually right after logon, since the file is older than 20 hours by then,
  at normal priority.
- A future server-side history (P2's route) can import this file; nothing
  here prevents it.

## Alternatives considered

- **Server table behind `POST /api/host/windows`** (the plan's P2/P4): needs
  a new authenticated route, a schema and a running daemon; deferred until a
  surface other than this window needs the data.
- **A line per scan**: grows with every window open and makes "per day"
  a read-side problem; one line per day is the question people ask.
- **Browser storage in the page**: lost with the webview profile, invisible
  to the tool, and only filled while the window is open.
