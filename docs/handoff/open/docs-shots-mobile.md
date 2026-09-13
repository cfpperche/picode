# Docs harness: docs-shots mobile-inbox flake

## Debts

- `make docs-shots` can fail at `app-mobile-inbox` under machine load (skeleton with data in store; retry passes). Suspect: reload+WS race in the harness flow (`scripts/docs-shots.mjs`).
- `pi auth check` reports `invalid_state` for a provider whose `models.json` fails schema validation — reads like a credential problem but is a schema one (upstream report candidate).
