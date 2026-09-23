# ADR-0189: Claude Code on a cloud platform, set up from PiCode's GUI

- **Status**: accepted (owner approved 2026-09-22, in session; slice 2 of ADR-0187)
- **Date**: 2026-09-22
- **Boundary**: persistence. PiCode writes the `env` block of Claude Code's user settings (`~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR`'s), and that block can hold a platform secret. Protocol: `PUT`/`DELETE /api/claude-code/platform`, and the roster's `platform` field.

## Context

Claude Code's `/login` has a third option, "3rd-party platform: Amazon Bedrock, Microsoft Foundry, or Vertex AI". Read from 2.1.280, its wizard:
- lists the same sign-in methods offered here (Bedrock: AWS profile, Bedrock API key, access key + secret, or the environment; Vertex: gcloud ADC, a service account key file, or the environment);
- saves the choice as an `env` block in the user settings: the platform's switch and values are set, and every other platform's keys and the model pins are removed.

Foundry has no wizard in that version, only its variables (`CLAUDE_CODE_USE_FOUNDRY`, `ANTHROPIC_FOUNDRY_RESOURCE`, `ANTHROPIC_FOUNDRY_API_KEY`).

Measured with fake values: a platform outranks a Console key ("dispatching to bedrock" with `ANTHROPIC_API_KEY` set), and a settings-only `env` block is honoured.

## Decision

PiCode writes the block the wizard writes. That makes the choice Claude Code's own: it holds in any terminal, and the file is the truth, as `.credentials.json` is for the subscription (ADR-0166). Every other settings field, and every env key a platform does not own, is kept. A settings file that does not decode is refused, not overwritten.

One login is in use:

| Choice | Effect |
|---|---|
| a platform | its block is written; any other platform's keys go; the chosen Console key is cleared |
| the subscription or a Console key (Use) | the platform block is removed |

The roster reports the platform's kind, method, region, project, resource or key-file path, never a secret. The pane shows "Claude Code uses …" with Edit and Stop using. Edit never shows the saved secret; it has to be entered again. Changes apply to new terminals.

## Consequences

A terminal-averse person can move Claude Code onto their cloud account without learning its variables. The secret for a key-based method lives where Claude Code's own wizard puts it: plaintext in the user settings, not in PiCode's encrypted vault. That matches what Claude Code does on its own; the vault holding a second copy would only drift. Removing the platform removes its key, and the confirmation says so.

If a future Claude Code changes the names, the wizard's own block and PiCode's diverge. The test pins the names measured on 2.1.280.

## Alternatives considered

- **Inject the platform at launch from the vault** (the ADR-0187 key channel). Rejected: it would hold only inside PiCode terminals, and it would duplicate what Claude Code already persists.
- **Open the wizard in a terminal.** Kept only as the "Sign in from a terminal" fallback: the owner's direction is GUI first.
