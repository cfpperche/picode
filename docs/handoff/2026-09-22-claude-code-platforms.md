# 2026-09-22 — feat/claude-code-platforms: Claude Code on Bedrock, Foundry or Vertex from the GUI

Shipped: ADR-0189 (owner-approved, slice 2 of ADR-0187). The Claude Code sign-in dialog gains "3rd-party platform", which offers Amazon Bedrock, Microsoft Foundry or Google Vertex AI. Each form offers the sign-in methods Claude Code's own wizard lists (read from 2.1.280). `PUT`/`DELETE /api/claude-code/platform` writes or removes the `env` block the wizard itself writes in `~/.claude/settings.json`: the platform's keys set, every other platform's keys and the model pins removed, everything else kept. A file that does not decode is refused. The roster's `platform` field carries no secret. The pane line "Claude Code uses …, instead of the accounts below" has Edit (secrets never shown) and Stop using (with a confirm). A platform outranks the subscription and a Console key: saving one clears the key, and Use on either removes the platform.

Verified: `make ci-scoped` PASS, and `TestClaudeCodePlatform` covers the decision table plus the refusals. Precedence was measured with fake values ("dispatching to bedrock" while `ANTHROPIC_API_KEY` was set; a block that exists only in settings is honoured). Scratch QA with an isolated HOME: the first pass was FAIL (an error lingered after switching method, and the pane line failed the row audit). Both were fixed and rechecked, and the audit was ok on desktop and mobile.
visual-review: PASS (ccp2-2a-desktop-line.png, ccp2-2b-mobile-line.png, ccp2-1a-accesskey-cleared.png; card 5/5)

Not verified: a real Bedrock, Foundry or Vertex account end to end. Known nits: no gap between the description and the first label in the platform form, and the field with the error is not highlighted.

## Next up

- Claude Code slice 3: an Anthropic-compatible gateway (ANTHROPIC_BASE_URL + ANTHROPIC_AUTH_TOKEN) as its Custom provider

Merge: fast-forward ready.
