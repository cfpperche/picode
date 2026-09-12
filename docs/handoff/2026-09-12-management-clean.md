# 2026-09-12 — feat/management-clean: Management window + clean + .wslconfig (Phase 2 session C)

Shipped (fast-forward ready, installed live): the tray item is **Management**
and the window has three tabs. Disk = the session-A view + live Give back
(session B). **Clean** = `picode clean` (new in-distro subcommand; the prune
table IS the hostfs consumer table — one source of truth; KindData refused
even by exact name; `rm -rf` rows pinned to the consumer's own paths) wrapped
by `picode-desktop clean` (stdout pass-through; same JSON-lines contract).
**Config** = `.wslconfig` edit in place: four owned keys in [wsl2], unknown
lines untouched, .wslconfig.bak before every save, never restarts WSL (that
is Give back's job, where the cost is stated). Rust INI logic lives in
src/lib.rs (no tauri) so `cargo test --lib` runs on the build host — 6 tests.
Go: clean refusal matrix + happy path + rm-guard tests.

Verified live: picode-desktop clean --list over the real distro (49 GB
prunable, 38 GB go-build safe); exe installed to %LOCALAPPDATA% (tray via
desktop-restart, in-distro picode swapped rm+cp over the running daemon).
The window itself is owner-visual (tabs, checkboxes, form).

Debts: clean measures with two full home sweeps (before/after, ~20 s); the
toast Start Menu shortcut still pending; owner visual check of the three
tabs pending.

Merge: fast-forward ready.
