# 2026-09-14 — feat/load-models

The custom endpoint dialog loads the model list instead of making people copy
it (open topic P1, now paid). `POST /api/providers/custom/models` →
`internal/modellist` asks the configured host what it serves: OpenAI,
Anthropic (`x-api-key`) and Google (`x-goog-api-key`, `models/` prefix
stripped) shapes; `/models` first and `/v1/models` only when that 404s;
bounded at 12s / 2 MiB. The key travels only to the host the user configured
for that provider, and a provider echo of it is redacted before the message
leaves the server — both sides have tests.

Form behaviour: ids merge into what was typed (nothing reordered or dropped);
context window and max output fill **only** when every listed model reports
the same number, and a list whose models differ says so instead of leaving a
blank field unexplained. An endpoint listing nothing is an answer, not an
error. The copy lives in Go, so both apps say the same words;
`web/shared/client/modelLoad.js` is framework-free (shared holds no React) and
holds the merge/line logic the dialogs share.

Verified on a scratch instance with a fake gateway serving each state
(`var/qa/fake-gateway.py`): idle, success+filled, blocked 401, error 404 and
empty, desktop at 1280 and mobile at 480 and 390, light and dark — screenshots
read, `__picodeOverlayAudit()` ok each time, no clip, no horizontal overflow.

## Next up

(shipped: `customApiHint` P2 — bullet pruned 2026-09-15 to hold the board budget)

## Debts

- The listing is not used to *validate* a key (P4, owner call): a good key on
  an endpoint whose listing is disabled still reads as "no model list".
- `internal/modellist` has no per-endpoint timeouts beyond the 12s total; a
  slow gateway holds a request that long.
