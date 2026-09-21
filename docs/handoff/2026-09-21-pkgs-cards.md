# 2026-09-21 — feat/pkgs-cards: the guest plugin lists become two-column cards

Shipped (ADR-0167): both lists (Installed, Marketplace) are Pi's package grid — `1fr 1fr` from 900px, one column below. Each plugin is a card: mono name +
version chip, `→ latest` when the CLI's own catalog says newer, the vendor's description clamped to 3 lines, a meta line (scope chip, the vendor's status word,
the origin as a mono ellipsized path with the full text in its `title`), an error line when an action was refused, and its actions on a footer. The footer is
what aligns (`margin-top: auto`; footers measured 741/741 and 781/781 across a row) — the first cut put `flex: 1` on the description, which stretched the
paragraph and left a hole mid-card; the slack now sits between the description and the meta. No preview frame: Pi's card has one because a package may carry a
capture, a vendor plugin has none, so an empty frame would be decoration with nothing behind it. Skeletons are the same 4-card grid, so the first paint matches.
A plugin the CLI reports as off dims its text but not its actions (Enable is the fix). Refused: virtualising a 309-row catalog — a dependency; the filter is the
tool, and Pi's pane reads its whole gallery too. Files: `web/{browser,mobile}/src/components/GuestPackages.jsx` + `styles/app.css` (which still differ only on
line 2) and `docs/changelog.d/pkgs-cards.md`; commits `efa2f71f`, `db0d5b5c`.
Verified: `make ci-scoped` PASS (4 paths: test-js, build; no Go change; run before the merge of `main`, which is why the stamp reads stale); a scratch at
1500px (two columns, `465px 465px`, footers aligned), 860px and mobile 390px (one column, no horizontal overflow), `window.__picodeOverlayAudit()` `ok: true`
in all three, screenshots read — a rendered pane in a scratch, never on the owner's own display (the method's blind spot).
visual-review: PASS
Not done / debts: none owed by this branch — the catalog stays whole, by the decision above.
Merge: fast-forward ready (3 ahead, 0 behind, clean; `main` ff to `feat/pkgs-cards`).
