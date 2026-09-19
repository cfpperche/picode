# 2026-09-19 — feat/connectors-catalog-sync-fix: unblock registry sync so marketplace search works

Shipped: Two live owner reports — marketplace search for "excel" returned
nothing, and a confusing "PiCode" group label. Root cause: the background
registry sync carried a 60s budget against a full registry of 8k+ servers
over 80+ sequential pages — every attempt timed out, the error was
swallowed, production stayed seed-only forever (no
~/.picode/connectors-catalog.json; verified live). A registry probe finds
13 excel-matching servers, most streamable-http (they pass the curation
filter) — reachable once one sync completes. Fixes: fetchTimeout 60s →
10min; failed background refreshes now log to the service log
("mcpcatalog: background refresh failed…") instead of vanishing; the
marketplace drops the "PiCode" group label (it also mislabeled hand-picked
presets like DeepWiki) — only synced registry rows keep one ("Catalog").

Verified: mcpcatalog unit tests green; web tests 64+425 pass 0 fail; npm
build ok; visual check on scratch — label gone, cards flow straight from
the toolbar (an earlier stale-page eval gave a false old-UI reading after
the scratch daemon died mid-check — redone against a live instance);
make ci-scoped PASS (3 paths).

## Next up
- After deploy: first sync takes minutes (81+ pages); confirm
  ~/.picode/connectors-catalog.json appears and
  /api/connectors/gallery?q=excel returns registry rows (owner acceptance).
