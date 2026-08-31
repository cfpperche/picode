# ADR-0031: Live provider usage on `#/providers`

- **Status**: accepted
- **Date**: 2026-08-31

## Context

Signed-in providers that bill against a subscription (Claude Pro/Max, Codex,
Copilot, Kimi, SuperGrok) expose 5-hour / 7-day / weekly / extra windows.
Pi's TUI footer and PiCode's composer statusbar only show **session** tokens
and cost — on purpose: those numbers are in the JSONL. Plan quota lives on
undocumented vendor endpoints (Anthropic `/api/oauth/usage`, Codex `wham/usage`,
Copilot `copilot_internal/user`, Kimi `/coding/v1/usages`, xAI
`cli-chat-proxy.grok.com/v1/billing`). Community Pi extensions already call
them. PiCode stores the same OAuth tokens in `auth.json` and must not invent
percentages.

The owner asked for a Usage control on `#/providers` that opens a dialog with
those windows. Cursor Settings → Usage is the product bar (progress + reset,
opened on purpose). Stripe progressive disclosure: hide the button when the
credential cannot fetch. Tokens must never reach the browser (ADR-0013).

## Decision

PiCode fetches live plan windows in Go (`internal/usage`) using the **active**
`auth.json` slot. `GET /api/providers/{id}/usage` returns a normalized report
(`status`, `windows[]` with `usedPercent` or `remaining`, `resetsAt`). The
catalog includes `quotaKind` so the UI can hide Usage when the method cannot
fetch (API-key Grok, unsigned, llama.cpp). The dialog is Radix, fetch-on-open,
skeleton then bars. The composer statusbar still does not invent quotas.

Vendor URLs are accepted cost. A parse miss or HTTP 5xx is `status=error` with
a one-line message, never a guessed 0%. OAuth refresh happens in-process when
`expires` is past or the usage call returns 401.

## Consequences

- Easier: users see 5h/7d/week remaining without a Pi TUI extension.
- Harder: undocumented endpoints will drift; each adapter is isolated so one
  vendor break is one error line.
- Wrong: a guessed bar would violate the statusbar rule we already shipped.
  Tests pin the decision table (hide vs fetch vs empty vs 401/429/5xx).
- V2 can add API-key meters (`zai`, `opencode-go`) on the same interface.

## Alternatives considered

- **Vendor npm extensions in the GUI.** Refused: ADR-0003, extra runtime.
- **Quota % on the roster or statusbar.** Refused: "0" badges and invented
  numbers; progressive disclosure is the Cursor/Stripe bar.
- **Per-vault-account fetch without Use.** Deferred: pi still has one slot;
  Use then Usage is the path.
