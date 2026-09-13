# 2026-09-13 — feat/browser-cdp: the work browser's CDP bridge and its tier gate

Shipped: Phase 3 slice 2, increments 1–2 (`desktop-shell/`). `btab_cdp_call`
runs one CDP method on a tab's controller through `CallDevToolsProtocolMethod`
— no debug port anywhere in the path; `btab_cdp_events` records a per-tab
read-tier event ring (Page/Runtime/Network/Log, the domains enabled on the
first poll) with sequence numbers, so a poller can tell a quiet page from an
overflowed ring. The gate is `src/cdppolicy.rs`: a named method catalog per
tier, deny by default at every tier. The loopback debug port is now the
explicit `PICODE_CDP_PORT` opt-in (ADR-0128 items 1–2) instead of every
launch's default. Also fixed: `make desktop-shell` was shadowed by its own
directory (not in `.PHONY`) and had to be built by hand.

Verified: `cargo xwin check --target x86_64-pc-windows-msvc --all-targets`
clean; `make desktop-shell` release build (2m 04s, `picode-shell.exe`); the
catalog's 9 tests green (tier matrix, deny-by-default, refusal copy — run from
a dependency-free copy, because this crate's tests do not build on Linux);
`make close` → ci-scoped PASS (full matrix, after merging main).
visual-review: n/a (no UI surface changed; the shipped window is untouched).
Not done / debts: increment 3 (the daemon endpoint and the `browser` Pi tool),
the navigation gate, the per-agent policy UI and slice 3's surface are in
`docs/handoff/open/work-browser-tabs.md`. The board sat at its 12 KB cap, so
three paid bullets went with this branch (the favicon 404, the activation
fast-forward, the mobile Git switcher); each deletion is the record.
Merge: fast-forward ready (main merged in, `make close` green).
