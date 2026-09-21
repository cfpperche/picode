# 2026-09-21 — feat/keyboard-envelope: one door for every CLI's key map (P1)

Shipped: `internal/clikeys.Registry` — nine rows (Pi, Claude Code, Codex, Grok,
Hermes, OpenCode, Muse, Antigravity, Omp) declaring each CLI's key-map shape, when
the CLI picks an edit up, its contexts and how far PiCode's support has got, every
row citing where those vendor facts were read. `GET/PUT /api/cli-keys` answers the
pane for any of them: pi's report is *wrapped* into the envelope (never restated),
a CLI whose editor has not shipped answers with `state` and `writable: false` and
no file and no actions, and a write to it is refused by name ("cannot write X's key
map yet" vs "X does not allow its keys to be remapped"). `/api/pi-keys` is
untouched. `web/shared/domain/cliKeys.js` owns the copy (the pickup sentence per
state; one line + one action for a pane with no editor) and
`TestJSListMatchesTheKeyboardRegistry` holds its ids, states, pickups and order
equal to the Go registry's. The pane now talks only to the envelope — the same
component renders a guest the day its adapter lands.
Verified: `make ci-scoped` PASS; `qa-cli-settings.mjs` — all six keyboard rows pass
on both apps, writing through `/api/cli-keys` and reading through `/api/pi-keys`
(cross-door, so neither side can drift alone). `make keys-drift` (new, P1's probe)
re-reads the installed pi 0.87.0 against `pikeys.Catalog`: 90 actions, 10 unbound,
no drift — and it exits 1 on a deliberately broken catalog, checked.
visual-review: PASS (one copy change on an already-verified surface: the pickup
sentence comes from the table now, so `/reload` lost its code styling. The first
read caught a real blemish — the period after the file path sat one padding-width
away, and orphaned to the next line on the phone — fixed by rendering it inside
the code box; both frames re-read and confirmed).
Not done: P2 (the per-CLI adapters, Omp first, plus the `guest-keymaps` ADR) is the
next session; `qa-cli-settings.mjs` still fails on the two rows that fail on `main`
without this branch (`docs/handoff/open/agent-clis-native.md`). No changelog
fragment: nothing user-visible changed.
Merge: fast-forward ready.

## Next up

- P2 of `docs/plans/keyboard-pane.md`: `make adr NAME=guest-keymaps`, then `internal/clikeys` adapters with Omp first (its map is `keybindings.yml|.yaml|.json` in the agent dir, 116 ids, same id namespace as pi) and Hermes' three scalar keys.
