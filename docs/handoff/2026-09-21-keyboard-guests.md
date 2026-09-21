# 2026-09-21 — feat/keyboard-guests: the Keyboard pane answers for every CLI (P2a)

Shipped: the Keyboard tab of **every** agent CLI now answers for that CLI. The
pane takes its CLI from the route and the registry decides: a map PiCode can edit
falls through to its own editor (Pi), and every other row draws one state line
plus one action from `blockNote()` — the vendor's key list for Grok and Muse
(which refuse remapping), the documentation for Codex, Claude Code, OpenCode and
Antigravity, and **Hermes' own Settings tab**, where its three rebindable keys are
now declared rows (group **Keyboard**: `voice.record_key` `ctrl+b`, `copy_shortcut`
`auto|ctrl_c|ctrl_shift_c|disabled`, `display.busy_input_mode`
`interrupt|queue|steer` — defaults and ranges read from the CLI's own
`hermes_cli/config_defaults.py`). `noteIsExternal()` picks how an action opens (a
vendor page in a new tab, an in-app route in the tab) and the harness asserts
both, because a note whose action does nothing is not an action. The placeholder
`"…are in development — coming soon"` is gone from this pane — it promised a
feature instead of describing the CLI. New `CliKeyboard.jsx` (one per app);
`clikeys` keeps one state vocabulary and `Keymap: Partial` tells Hermes' case
apart, including in the refusal ("keeps no key map file; the keys it does allow
are in Settings"). This is P5 of the plan shipped early plus the half of P2 that
needed no writer.
Verified: `make ci-scoped` PASS; `qa-cli-settings.mjs` — 30 rows green up to the
one row recorded red on main (below), including the two new guest rows on both
apps and the mobile trust-matrix rows that used to die on a covered element.
visual-review: PASS (7 captures read in a subagent: three guest panes × two apps
plus Pi's own map as the regression reference; card 5/5 both surfaces; the only
clipped text anywhere is the mobile tab strip's inactive tab, a shell artifact —
`overlayAudit` ok on both apps).
Debts: the blocked-project-layer `ready()` row in `qa-cli-settings.mjs` is still
red and is not this pane's (`docs/handoff/open/agent-clis-native.md`, updated
with today's measurement: three stale assertions, not two, and the toast stack
that covered a captured pane). `qa-cli-settings-recovery.mjs` was not re-measured.
Merge: fast-forward ready.

## Next up

- P2b: the flat key-map adapter + the Omp catalog (70 ids, labels from the CLI's own descriptions) + the `omp` registry row from `planned` to `shipped` + `PiKeys` taking its CLI from the route. Start point: `docs/plans/keyboard-pane.md` "P2's inputs"; decision: ADR-0174.
