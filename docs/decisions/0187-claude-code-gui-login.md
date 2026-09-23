# ADR-0187: Claude Code signs in through PiCode's GUI

- **Status**: accepted (owner approved 2026-09-22, in session)
- **Date**: 2026-09-22
- **Boundary**: process: a Claude Code launch gets `ANTHROPIC_API_KEY` from the vault. Persistence: PiCode records which key Claude Code uses (setting `credentials.claude-code.key`) and writes the key's approval into Claude Code's own `~/.claude.json`. Protocol: the roster's `add.kind: "claude-code"`, and `signin: "browser"` for Claude Code.

## Context

The owner's direction: every CLI's logins and credentials happen in PiCode's GUI. Each CLI gets a dialog shaped like pi's but adapted to its runtime, and the CLI's terminal `/login` stays as an optional fallback. Claude Code's `/login` offers three things: a Claude subscription, an Anthropic Console account (API billing), and a third-party platform (Bedrock, Foundry, Vertex).

Before this ADR, PiCode offered the terminal `/login` plus Import. A Console key saved in the vault **never reached Claude Code**: Use writes only subscriptions into `~/.claude/.credentials.json`, and nothing set the key's variable.

Measured on Claude Code 2.1.280, with fake keys in an isolated HOME:
- `ANTHROPIC_API_KEY` outranks both the subscription file and `CLAUDE_CODE_OAUTH_TOKEN` ("API-key auth precedence active").
- A new key is approved by storing its last 20 characters in `customApiKeyResponses.approved` in `~/.claude.json` (`trim().slice(-20)` in its own code).
- PiCode's Anthropic OAuth uses the same client id as Claude Code.

## Decision

Exactly one login is in use for Claude Code, and Use chooses it.

| Condition | Action |
|---|---|
| Use on a subscription row, Claude Code terminals running | refused (ADR-0166's rule) |
| Use on a subscription row, none running | written into `.credentials.json` (scopes filled on a fresh file); the key setting is cleared |
| Use on a key row | the setting records it; its tail is approved in `~/.claude.json`; applies to new terminals |
| Launch, setting names a live unpaused key, the person's env lacks the variable | `ANTHROPIC_API_KEY` is injected |
| Key paused, removed or not set | nothing is injected; the subscription file rules |
| Browser sign-in finishes | the login lands in the vault and, with no Claude Code terminal running, is written as above |

The Add button opens a Claude Code dialog: subscription (browser), Console key ("Save and use"), and "Sign in from a terminal" as the fallback. Third-party platforms and Anthropic-compatible gateways follow in later slices.

## Consequences

A terminal-averse person can sign Claude Code in without a terminal, and a Console key saved in PiCode finally works. Choosing a key moves billing to API usage. The dialog says so, and the choice is explicit and reversible with Use.

PiCode now writes one field of `~/.claude.json`, keeping every other field. If Claude Code changes how it approves keys, the terminal asks once again. That fails safe.

The setting and the vault can disagree (a removed row). The launch re-checks the row every time, so a stale setting injects nothing.

## Alternatives considered

- **Inject the key whenever one exists.** Rejected: it would silently switch a subscriber to API billing, because the key outranks the file.
- **`apiKeyHelper` in `~/.claude/settings.json`.** Rejected for now: it adds a script PiCode must keep, and the env is already how launches carry credentials (ADR-0178).
- **Reuse pi's Add dialog.** Rejected by the owner: each CLI gets its own dialog.
