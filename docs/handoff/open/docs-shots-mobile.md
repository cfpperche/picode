# Docs harness: docs-shots mobile-inbox flake

## Debts

- `make docs-shots` can fail at `app-mobile-inbox` under machine load (skeleton with data in store; retry passes) — reload+WS race in the harness flow. 2026-09-21: failed 2/2 runs (the deploy's recapture, then a retry) with 5–6 `MARKER_NO` rounds each, while the surface itself was healthy — the same fixture answered the marker in ~1–2 s, and a manual replay of the harness's own round (390×844, `about:blank`, `?_r=` url, dark `location.reload()`, 4 s settle) passed 3/3, with and without the query nonce. So a retry does not clear it on a loaded machine; the long single-session run (five desktop surfaces first) is the remaining suspect, not the surface.
- `pi auth check` reports `invalid_state` for a provider whose `models.json` fails schema validation — reads like a credential problem but is a schema one (upstream report candidate).
