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

2. **Auth stays in pi.** Settings → Providers lists catalog providers and
   whether `~/.pi/agent/auth.json` has a key for that id (keys only — never
   values). **Sign in** starts the agent interactively and sends
   `/login [<provider>]` through tmux. No API-key fields in PiCode.

3. **MCP is not part of creation.** Settings → MCP is a separate status
   surface. v1 reports whether a community adapter/config is present; it
   does not invent a server manager. Agent creation never asks about MCP.

Starting an agent (interactive or managed) passes stored
`--provider` / `--model` / `--thinking` when set.

## Consequences

- **Easier**: one source of truth (pi) for models and credentials; wizard
  stays three fields; MCP cannot block "add a folder".
- **Harder**: catalog parsing depends on `pi --list-models` table format;
  `/login` requires an interactive session (OAuth still happens in the TUI).
- **If wrong**: a curated provider list would hide working setups (cage);
  a PiCode login form would fork auth from `auth.json`.

## Alternatives considered

- **Curated Anthropic/OpenAI/Google only**: rejected — hides the user's
  existing OpenRouter/Codex/xAI logins.
- **API-key form in Settings**: rejected — duplicates `auth.json`, leaks
  secrets into our UI, fights `/login` OAuth.
- **MCP inside the wizard**: rejected — creation should be name + folder
  + optional model. MCP is infrastructure, not an agent identity.
