# 2026-09-23 — feat/wsl-mgmt-history: disk history, one line per day, and the growth since a named day

Item 7 of the owner's 8-item WSL Management list; last of four branches, so all 8 items exist (1, 2, 6 restart-memory; 3, 8 root; 4, 5 move; 7 history).

Shipped (bed416175, ADR-0203):
- Disk history as JSONL on the Windows side (one line per day, same-day scan replaces it, 400 lines max); `picode-desktop history`.
- The shell's daily background scan: takes the one-job lock, 10 min bound, 6 h retry; stages skip under a distro hold. disk-compact now holds the distro.
- System tab History card: chart plus growth "since <day>", naming the baseline day.

Verified live on the owner's machine: two real scans today wrote one line (same-day replace), no temp file left; C: 10.4 GB free, disk file 227 GB, 119 GB used inside the distro. Blind spot: the shell's daily background scan never ran live (needs `make desktop-restart`).
Adversarial review, fixed: background scan racing compact/move/backup/update (job lock); growth inventing a whole cache size when the baseline lacked it; a "7 days" heading after gaps; unbounded, hourly-repeating scan; shared temp file and Windows rename failures; chart drawn on a hidden tab; unescaped day text; docs claiming `disk` changes nothing.
visual-review: PASS (hist/, 6 states, synthetic 21-day data; chart per the dataviz skill, palette validated on both surfaces). The later tooltip-side, date-format and heading fixes were verified by geometry only; 07-hover-right-half.png was not reviewed by a subagent.
Merge: `make close` had not run when this note was written.

## Next up

- Owner runs `make deploy` + `make desktop-restart` to get all four WSL Management branches' shell and page changes live.
