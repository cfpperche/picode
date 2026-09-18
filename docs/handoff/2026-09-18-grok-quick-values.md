# 2026-09-18 — grok-quick-values

The installed grok is the official xAI "Grok Build" CLI — an earlier study
mis-attributed it to the community superagent-ai/grok-cli. The owner's
pointer to docs.x.ai/build/cli/reference settled it.

## Done
- Grok quick settings expanded: Model (`--model/-m`), Reasoning effort
  (`--effort/--reasoning-effort`, text — values are model-dependent and not
  enumerated), Sandbox select with the built-in profiles (off, workspace,
  devbox, read-only, strict — custom names render "as typed"), plus the
  existing Approvals and Auto-approve (--always-approve, alias --yolo).
- grok docs links (catalog + lifecycle) point at
  docs.x.ai/build/cli/reference; grok.com/build was an empty JS shell.
- Vendor receipts: reference table + sandbox page + settings reference
  (GROK_SANDBOX values), all fetched 2026-09-18.
- Lifecycle plan unchanged — `grok update` matches the official reference.

## Debts
- Grok effort values stay free-text until xAI publishes a fixed list.
