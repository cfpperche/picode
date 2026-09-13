# Docs harness: docs-shots mobile-inbox flake

## Debts

- `make docs-shots` can fail at `app-mobile-inbox` under machine load: the list stays on skeleton while badges show data; a retry passes (2026-09-13: 5 consecutive failures during parallel builds, green after). An earlier "deterministic main regression" read was wrong — retried under load, main's tree failed identically. Suspect: reload+WS race in the harness flow (`scripts/docs-shots.mjs`).
- `pi auth check` reports `invalid_state` for a provider whose `models.json` fails schema validation — reads like a credential problem but is a schema one (upstream report candidate).
