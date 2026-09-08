# 2026-09-07 — feat/term-find-word: whole word in the terminal find

Shipped: the third search toggle, **Match whole word**, completing the set VS
Code shows (`Aa` / `ab` / `.*`). The mode object in `lib/termFind.js` is now
driven by `FIND_FLAGS`, so a flag is one entry in a list rather than three
edits, and `modeChanged(a, b)` names the comparison the field uses to force
the addon to re-scan (it caches per term and ignores an options change on its
own — `clearDecorations()` drops that cache).

Verified: `make ci-scoped` green (7 tests in `termFind.test.js`). Live on
scratch `word` (:8471) over `beta here / betamax there / BETA caps /
sub-beta dash`: plain `beta` 12 matches → whole word 5 → plus match case 4.
The semantics, proved without arithmetic: with whole word on, `betamax`
matches (`1/1`, it is a word) while `etamax` says "No results". The
screenshot shows the same thing in one frame — `beta`, `BETA` and `sub-beta`
highlighted, `betamax there` untouched.

visual-review: PASS (find-wholeword.png; overlayAudit ok, `[data-align-row]`
36×8 with eight controls; card 5/5).

Debts: none new. Find stays desktop only — the mobile terminal has no menu
to open it.

Merge: fast-forward ready.
