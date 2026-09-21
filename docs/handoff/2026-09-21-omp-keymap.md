# 2026-09-21 — feat/omp-keymap: Omp's key map is editable (P2b)

Shipped: **the flat key-map engine and its first declaration**. Omp's Keyboard
tab (Agent CLIs → Omp → Keyboard) now renders its 70 actions — 32 `tui.*` + 38
`app.*`, every label the CLI's own description — and lets a row be added,
replaced, handed back, or cleared wholesale. PiCode writes Omp's own file: the
`keybindings.yml` it is using, else `.yaml`, else the legacy `.json` (never a
second file beside them), in the agent dir the CLI itself resolves
(`$PI_CONFIG_DIR` or `~/.omp`, then `profiles/$OMP_PROFILE` or `$PI_PROFILE`,
then `agent`). `[]` unbinds; `Reset all` removes only rows the catalog knows.
The engine reuses the settings package's guarantees instead of rewriting them:
four primitives exported from `clisettings` (`internal/clisettings/keymap.go` —
`OpenDoc`, `Strings`, `SetStrings`/`RemoveKey`, `Save`), the same parser,
byte-span splice, atomic rename and revision check ADR-0163 established, plus the
one piece of syntax they never needed — a list literal — and its refusals: a row
held as a table is `ErrShape` (reported as `unreadable`, dropped from the pane,
refused on write), TOML has no literal, and a write sends the revision the pane
read, so a file that moved answers 409 instead of being overwritten. The
envelope grew `revision`, `unreadable` and the CLI's own `platform`, and its
`actions` are now `clikeys.Action` — pi's catalog is converted in one place
rather than joining the two packages. `clisettings` itself is unchanged in
behaviour.
Verified: `make ci-scoped` PASS; `internal/clisettings` and `internal/clikeys`
tests, including the new engine suite (create-the-file, the three-name
precedence, a legacy JSON map edited in place, an unknown action refused by name,
a stale revision refused, reset leaving unknown keys and comments alone, an
unreadable row reported rather than swallowed) and the endpoint test (GET 70 rows
+ file + platform, PUT writes Omp's syntax, read-back, 409 on a stale revision,
reset). `qa-cli-settings.mjs`: 32 rows green on both apps, including a new pair
that captures `Ctrl+Alt+O` into Omp's first row, reads it out of Omp's own file
and proves `Reset all` hands it back — the run stops at the one row recorded red
on main (`docs/handoff/open/agent-clis-native.md`).
visual-review: PASS (see the card below).
Debts: none new in this pane. The pickup question P1 left as `unknown` is
answered by the measurement that shipped it: `restart`.
Merge: fast-forward ready.

## Next up

- P3 of `docs/plans/keyboard-pane.md`: Codex's nested TOML (`[tui.keymap.<context>.<action>]`) and Antigravity's flat JSON, both as declarations into the engine P2b left behind; then P4 (Claude Code's inverted contexts, OpenCode's `tui.json`), whose readers answer through the same envelope.
