# ADR-0009: Lifecycle surfaces — catalog, auth, MCP

- **Status**: accepted
- **Date**: 2026-08-24

## Context

M3 needs three product choices: which providers the wizard shows, where
credentials are entered, and whether MCP belongs in agent creation.

Constraints already in force: ADR-0003 (user-installed pi owns auth and
models), philosophy "door, not cage", UI chrome carries state not docs.

## Decision

1. **Catalog is whatever pi reports.** `GET /api/catalog` runs
   `pi --list-models --offline` and groups by provider. The wizard always
   offers **Inherit** (empty `provider`/`model`/`thinking` columns = pi
   defaults). PiCode never ships a curated subset.

2. **Auth stays in pi's `auth.json`.** `#/providers` lists catalog providers
   and whether that file has an entry (keys only — never values).
   **API keys** are written by the GUI (`{ "type": "api_key", "key" }`).
   **OAuth / subscriptions** (Codex, Claude Pro, xAI sub, OpenRouter PKCE)
   stay TUI `/login` until pi exposes RPC login. Composer `/login` opens
   `#/providers`; it does not type into tmux.

3. **MCP is not part of creation.** Settings → MCP is a separate status
   surface. v1 reports whether a community adapter/config is present; it
   does not invent a server manager. Agent creation never asks about MCP.

Starting an agent (interactive or managed) passes stored
`--provider` / `--model` / `--thinking` when set.

## Consequences

- **Easier**: one source of truth (pi) for models and credentials; wizard
  stays three fields; MCP cannot block "add a folder".
- **Harder**: catalog parsing depends on `pi --list-models` table format;
  OAuth still happens in the TUI (no RPC `login`). If that door hurts,
  a PiCode credential store is a future ADR — not this one.
- **If wrong**: a curated provider list would hide working setups (cage);
  a PiCode login form would fork auth from `auth.json`.

## Alternatives considered

- **Curated Anthropic/OpenAI/Google only**: rejected — hides the user's
  existing OpenRouter/Codex/xAI logins.
- **API-key form in Settings**: rejected — duplicates `auth.json`, leaks
  secrets into our UI, fights `/login` OAuth.
- **MCP inside the wizard**: rejected — creation should be name + folder
  + optional model. MCP is infrastructure, not an agent identity.
