# 2026-09-22 — feat/provider-picker-search: Add provider's search picks what you typed

Shipped: `AddProviderDialog.jsx` (browser and mobile) filters and orders its own rows with cmdk's `shouldFilter={false}` and a controlled search and selection. The untouched list is alphabetical, and a search ranks rows by cmdk's `defaultFilter` score. The selection moves in the same update as the search, and the list scrolls back to the top on the next frame. The effect is keyed by row ids, so arrow keys and hover still move the selection. Custom provider is now a footer button under the list, always visible. `providerIcon.js` maps about 30 of Omp's catalog ids onto marks from `@lobehub/icons-static-svg` (each name checked against the 1.73.0 file list). It also drops `opencode`, which has no mark in the set. `ProviderFace` takes `name`, so the letter plate reads the display name.

Verified: `make ci-scoped` PASS. Six scratch QA rounds; the first five were FAIL:
1. Custom pinned first stole Enter.
2. cmdk re-sorted by score.
3. The list stayed scrolled after "o…" and after clearing.
4. cmdk scrolled to the old selection after my reset.
5. A re-created array snapped the selection back to row 0.
The sixth round PASSED on Omp desktop, Pi desktop and Omp mobile: typing, clearing, arrows+Enter, hover, and no-match with the footer. Overlay audit ok, 0 broken icons.
visual-review: PASS (pks6-omp-cleared-after-open.png, pks6-omp-arrowed.png, pks6-mobile-cleared.png, pks6-mobile-arrowed.png; card 5/5)

Not done: after Back from a provider step, focus does not return to the search field.

Merge: fast-forward ready.
