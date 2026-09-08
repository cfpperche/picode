# 2026-09-07 — feat/term-find-modes: case and regex in the terminal find

Shipped: two toggles in the find field — **Match case** and **regular
expression** (`aria-pressed`, lucide `CaseSensitive` / `Regex`) — carried into
every search through `ISearchOptions`. The mode is remembered for the life of
the page and deliberately not persisted (`findMode`/`setFindMode` in
`lib/termFind.js`): a mode that outlives a reload turns the next plain search
into a mystery. A pattern that does not compile is named by `findProblem` —
the counter reads **Invalid pattern**, prev/next go dim and the addon is never
called, so a half-typed regex cannot reach its failure state.

One upstream quirk, worth knowing: `@xterm/addon-search@0.16.0` caches its
match list per term and does **not** re-scan when only the options change —
toggling case on `alpha` kept the old count of 6. `clearDecorations()` drops
that cache with the highlights, so the bar clears before any run where the
mode changed (`ranWith` ref in `TermFindBar.jsx`).

Verified: `make ci-scoped` green (6 tests in `termFind.test.js`, 340 JS
total). Live on scratch `modes` (:8471) over `Alpha/alpha/ALPHA` and
`beta 42/beta 7`: case off `1/6` → on `1/2` → off again `2/6`; `beta \d+`
plain "No results" → regex on `1/4` landing on `beta 42`; `beta [` →
"Invalid pattern", prev/next disabled, decorations cleared.

visual-review: PASS (find-modes.png; overlayAudit ok, `[data-align-row]`
36×7 with both toggles pressed; card 5/5).

Debts: no whole-word toggle (the addon supports it, one more button); the
toggles are desktop only, like Find itself.

Merge: fast-forward ready.
