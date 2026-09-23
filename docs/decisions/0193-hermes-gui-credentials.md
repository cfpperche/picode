# ADR-0193: Hermes credentials through PiCode's GUI, by Hermes's own `auth add`

- **Status**: accepted (owner, 2026-09-23: "pode seguir com (a)", a new credential goes last in Hermes's pool)
- **Date**: 2026-09-23
- **Boundary**: process. PiCode runs Hermes's own Python to read its provider registry, and runs `hermes auth add` on the person's behalf. Protocol: `POST`/`GET`/`DELETE /api/hermes/credential`, Hermes's roster gains its catalog and `add.kind: "provider"` with `signin: "device"`.

## Context

Hermes 0.21.4 keeps a credential pool per provider (several credentials, tried in priority order). It adds to that pool with `hermes auth add <provider> --type api-key|oauth`. Measured with fake values in a scratch `HERMES_HOME`:
- A key is read from stdin (so it never has to be in argv) and lands **last** in the pool (priority 1 after 0).
- It is stored in `access_token`, not `key`. PiCode's reader only knew `key`, so it never detected a pooled key.
- Every OAuth sign-in PiCode can offer (Nous, OpenAI Codex, xAI Grok OAuth, MiniMax) is a device code. Once Python is unbuffered, it prints the page and the code and waits. Qwen OAuth needs the Qwen CLI.
- `hermes_cli.auth.PROVIDER_REGISTRY` lists 78 ids, about 45 providers once aliases are merged, with name, auth type and key variables.

## Decision

- **Roster.** Hermes's roster appends its own registry, read through its venv's Python (`PICODE_HERMES_CATALOG` overrides; cached per source), minus declared ids and their Hermes forms, aliases, and the doors PiCode cannot drive (`aws_sdk`, `vertex`, `external_process`, Qwen OAuth).
- **Dialog.** The Add dialog is pi's picker, fed by that roster. Its key door runs `hermes auth add <id> --type api-key` with the key on stdin. Its account door runs `--type oauth --no-browser` and shows the page and code, only where Hermes runs a device code (not Anthropic or Copilot). One sign-in runs at a time.
- **Pool order.** A new credential goes last, Hermes's own default (option (a), the owner's choice).
- **Vault.** What Hermes stored is filed in the vault: the key directly, and a sign-in through the (fixed) pool reader.
- **Ids.** The vault's ids map to Hermes's (`google`→`gemini`, `github-copilot`→`copilot`, `opencode`→`opencode-zen`, `xai` key→`xai`, `xai` account→`xai-oauth`).

## Consequences

About 45 providers sign in from the GUI, and Hermes keeps owning its pool and rotation. Two things are Hermes's internals rather than an API: the registry read and the text `auth add` prints. If either changes, the roster falls back to the declaration and the dialog shows Hermes's last line. Moving a credential to the front of the pool ("use first") is left for later.

## Alternatives considered

- **Write `auth.json` directly.** Rejected: the pool's fields, labels and priorities are Hermes's to manage.
- **Put the key in argv (`--api-key`).** Rejected: the process list would show it.
