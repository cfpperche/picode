# 2026-09-20 — pi-resolver: the layer resolver follows the table instead of repeating it

Why this branch exists: Pi's rows became a table in the previous branch, but `web/shared/domain/resolveLayer.js` — the function that decides which value each layer shows — stayed a hand-written list of keys beside it. A setting took two edits, and a key added to only one of them rendered an empty control beside a row claiming "Set here". That is not hypothetical: it is how the Theme row shipped earlier the same day.
What landed: every field in `piRows.js` declares its type and its `unset` value — what pi itself does when no layer sets the key — and the resolver walks the table instead of restating it. `catalogBase` derives from the same place. A test asserts the resolver's key set *is* the table's, so the two cannot drift.
The subtlety the derivation had to keep, which the hand-written version encoded in five lines of `||`: a layer that **sets** a key keeps its value even when that value is the type's zero — an explicitly empty tool list means no tools, not the built-in set, and an explicit `false` is false rather than pi's default `true` — while a layer that sets nothing inherits its parent, and the field's `unset` runs only where the parent has nothing. That is a test now.
Verified live against a copy of the owner's real `~/.pi/agent/settings.json`: all eleven rows resolve to what pi would do — Auto-compact inherited on, Steering and Follow-up `one-at-a-time`, Tools the full built-in set, New folders `ask`, Theme `dark` marked Set here, Hide thinking off and marked Set here because the file says `false`. Overlay audit clean.
One thing worth writing down rather than filing as a bug: handing a key back removes this layer's override, it does not restore a value the layer held before. Setting Theme to `light` and then clicking Use inherited leaves no theme, not `dark`. That is what reset means everywhere in this pane, and in VS Code's, but it surprised the round-trip check that found it.
State: `make close` green, `main` can fast-forward. The debt this closes is removed from `docs/handoff/open/agent-clis-native.md`.

## Next up

- Nothing on this thread. Pi's pane and the eight guests' now share a shape: a table declares the rows, the resolver follows it, and adding a setting is one line plus its field in `internal/pisettings`.
