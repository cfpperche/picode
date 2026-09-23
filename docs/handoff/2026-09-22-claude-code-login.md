# 2026-09-22 — feat/claude-code-login: Claude Code signs in through the GUI

Shipped: ADR-0187, owner-approved, slice 1 of 3. Claude Code's Add opens `ClaudeCodeLoginDialog.jsx` (browser and mobile), shaped like its `/login`:
- **Claude subscription** in the browser. PiCode's OAuth uses Claude Code's own client. The login is written into `~/.claude/.credentials.json` when no Claude Code terminal runs, with scopes filled in on a fresh file.
- **Anthropic Console key** ("Save and use"). The setting `credentials.claude-code.key` records it, its last 20 characters are approved in `~/.claude.json`, and new launches get `ANTHROPIC_API_KEY`.
- **Sign in from a terminal**, as the fallback.

Use switches between the subscription and the key. `writeCLILogin` was pulled out of the activate handler so the browser sign-in can reuse it.

Fixed: a Console key saved for Claude Code never reached it before this branch.

Verified: `make ci-scoped` PASS. `TestClaudeCodeLoginInUse` covers every row of the decision table. The precedence was measured on Claude Code 2.1.280 with fake keys ("API-key auth precedence active"). The approval format (`trim().slice(-20)`) was read from the binary. Scratch QA ran with an isolated HOME: method, subscription and console steps, Save and use marking the key in use, Use back to the subscription, the terminal fallback, 390px, Omp and Codex unchanged, overlay audit ok. A recheck confirmed the key field's label and the link style.
visual-review: PASS (ccl-2-method.png, ccl-7-table-after.png, ccl2-1-console.png, ccl2-3-mobile-console.png; card 5/5)

Not verified: a real browser sign-in and a real key through to a live Claude Code session. That is for the owner to run.

Process slip, caught before the merge: renumbering ADR-0184 to 0187 with a global sed rewrote other sessions' ADR-0184 references across 32 files. The uncommitted ones were restored, and the committed ones were fixed with an amend (the commit changes no ADR-0184 line).

## Next up

- Claude Code slice 2: third-party platforms (Bedrock, Foundry, Vertex) in the same dialog (ADR-0187)
- Claude Code slice 3: an Anthropic-compatible gateway as its Custom provider

## Debts

- Claude Code's GUI sign-in and "Save and use" are not yet exercised live with real credentials (ADR-0187)

Merge: fast-forward ready.
