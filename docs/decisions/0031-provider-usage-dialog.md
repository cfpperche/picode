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

## Amendment (2026-08-31) — V2 meters and banked resets

Usage also covers **API-key plans** whose vendors publish a quota endpoint:
`zai`, `zai-coding-cn`, `opencode-go`. Catalog `quotaKind` is `api_key` for
those. Banked one-time **usage-limit resets** (Codex `wham/rate-limit-reset-credits`,
Grok `GetRemainingResets`) appear in `resets[]` when the vendor answers.
Redeem is `POST /api/providers/{id}/usage/reset` after a confirm. A Grok
reset fetch that needs grok.com cookies and fails is omitted — weekly windows
still show. No invented 0-reset badge.

## Alternatives considered

- **Vendor npm extensions in the GUI.** Refused: ADR-0003, extra runtime.
- **Quota % on the roster or statusbar.** Refused: "0" badges and invented
  numbers; progressive disclosure is the Cursor/Stripe bar.
- **Qwen Token Plan meter.** Refused for V3: the plan has no API-key quota
  JSON. Console usage needs a web session cookie (`GetSubscriptionSummary` /
  zelda), not `sk-sp-`. Hide Usage rather than invent a bar.
- **Chrome cookie dump for Grok resets.** Refused: the Grok CLI session
  (`~/.grok/auth.json`) is the fallback; optional `GROK_COOKIE` is explicit.
  Scraping the browser cookie jar is out of scope.

## Amendment (2026-08-31) — V3 per-account, Grok session, more keys

Usage is per vault row. `GET /api/providers/{id}/accounts/{aid}/usage`
(and `POST …/reset`) reads that cred without writing it into `auth.json`.
OAuth refresh writes the vault row; `auth.json` updates only if that row
is active. `#/providers` puts Usage on each account that `quotaKind`
matches. Group-level Usage is gone.

Grok banked resets try, in order: PiCode OAuth bearer, Grok CLI
`~/.grok/auth.json` `key` (refresh if `expires_at` is past), then
`GROK_COOKIE`. A miss still omits `resets[]`; weekly windows stay.

API-key meters: OpenRouter `GET /api/v1/key` (`limit_remaining`, then
`/credits` if the key has no cap), MiniMax / MiniMax CN
`/v1/token_plan/remains` (remaining-percent inverted to used), Kimi Code
API key on the same `/coding/v1/usages` as OAuth.
