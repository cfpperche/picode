# 2026-09-21 — feat/guest-keymaps: ADR-0174 accepted, and P2's inputs measured

Shipped: **ADR-0174 (guest key maps) accepted** — the per-CLI declaration and
driver behind `/api/cli-keys`, over the format layer of `internal/clisettings`
(exported unchanged: parse, span, splice, atomic, revision/409, the same
refusals), unknown actions refused by name, rows the file carries and the catalog
does not shown read-only, per-CLI chord vocabularies never translated, and
`planned` (adapter not written) told apart from `refused` (the vendor forbids
remapping). Plus the correction it forced: the omp registry row said **116 action
ids**; the bundle has **70** (32 `tui.*` + 38 `app.*`, every one with a label),
and 116 was a grep artifact of mine — dotted identifiers in a minified bundle
also match settings keys. The row now cites the bundle version it was read from
and the three-file precedence (`keybindings.yml` → `.yaml` → legacy `.json`, a
JSON file being migrated to YAML by the CLI), and `docs/plans/keyboard-pane.md`
gained a "P2's inputs" section with everything the first adapter needs.
Verified: `make ci-scoped` PASS; `internal/clikeys` tests pass with the corrected
row (the seam test holds ids, states, pickups and order against the UI table).
visual-review: n/a — no surface changed. The registry's `Source` text is not on
the wire; the ADR is a document.
Not done, on purpose: **P2's implementation did not start.** The engine files I
began (a chord grammar, the settings-layer export) were deleted rather than
committed: nothing uses them until the flat adapter and the Omp catalog exist
together, and dead code is a smell this repo rejects. The plan's "P2's inputs"
section is the start point: file precedence, the 70-id catalog and where its
tables live (the pinned `@oh-my-pi/pi-tui@18.2.8` sources and the bundle region),
the platform default for `app.clipboard.pasteImage`, the unbind verb, the
no-context fact, and the still-unknown pickup.
Merge: fast-forward ready.

## Next up

- P2: the flat key-map adapter + the Omp catalog (70 rows, labels from the CLI's own descriptions) + the `omp` row from `planned` to `shipped` + the pane taking its CLI from the route; then Hermes' three scalar keys. Start point: `docs/plans/keyboard-pane.md` "P2's inputs"; the decision is ADR-0174.
