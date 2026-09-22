# 2026-09-21 — feat/codex-fallback-rows: the three rows the live check found

Follow-up to the Codex keymap: the catalog now takes the **generated schema's**
per-context key list as its source — 149 keys, not the runtime inventory's 146 —
because the live check found the inventory omits `global.submit`,
`global.queue` and `global.toggle_shortcuts`, three keys the struct accepts and
the resolver reads as the composer's **global fallbacks** (`built_in_defaults()`
zeroes them whenever the composer's own key is set, and
`configured_binding_for_action` reads `keymap.global.submit` when the composer's
slot is unset). They ship unset — which is what the file says — and the vendor's
own descriptions label them ("Submit the current composer draft.", "Queue the
current composer draft while a…", "Toggle the composer shortcut overlay.").
A pinned test holds the `global` context to the twelve names **codex itself
printed** when refusing an unknown action, so a vendor release that adds or
renames one fails the suite instead of the user.

Also in the live checks: **Omp applies a map PiCode wrote** (`app.model.select`
as `ctrl+n` through the real engine, a fresh session opening the model picker on
it, the file deleted afterwards — the config had none) and **Codex reads and
validates the table** (an unknown action in `[tui.keymap.global]` makes it refuse
the file and print the context's twelve accepted names). Recorded in
`docs/handoff/open/agent-clis-native.md`. Not yet observed: pressing a rebound
key inside a live codex TUI — its startup in an isolated HOME exits; that half
needs a run against the real HOME and is the owner's call.
Verified: `make ci-scoped` PASS; the catalog test pins the count (149), the
contexts' order (the runtime inventory's, which `/keymap` exposes), the global
twelve, and the fallback slots shipping unset; `qa-cli-settings.mjs`'s codex
guest asserts 149 rows end to end. visual-review: the pane's layout is the one
reviewed in the previous slice; only three rows of the same shape were added.
Merge: fast-forward ready.

## Next up

- Antigravity (the rest of P3): measure its pickup, take the catalog from the two vendor doc pages that match the installed build, declare it as a flat map. Then Claude Code and OpenCode (P4).
