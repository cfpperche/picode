# 2026-09-14 — feat/custom-endpoint-v2

Three items off the custom-endpoint topic, in one branch.

**P2 — URL hints.** The base-URL field carries a hint and example that follow
the API type (pi appends the route itself; OpenAI roots end in /v1, Google's in
/v1beta, Anthropic-compatible endpoints differ). A test pins that every offered
type has one.

**P3 — the real thinking-format union.** The picker now offers pi's own values
(`openai`, not the undocumented `reasoning_effort` this form used to write —
both hit the same pi branch and the read path maps the old value to `openai`),
plus `chat-template`, `qwen-chat-template`, `baseten`, `string-thinking` and
`ant-ling`. `chat-template` and `baseten` reveal the JSON editor they read
(chatTemplateKwargs / chatTemplateArgs), validated in the browser and again in
Go, with pi's `$var` references allowed and anything else refused.

**P4 — Verify spends one real request (owner-authorised).** A custom endpoint's
Verify used to answer from credential presence, so a wrong key read green. It
now sends one minimal completion (one word in, the smallest ceiling each API
accepts, one retry for `max_completion_tokens`), and reports model, milliseconds
and token counts. The row's action says "Verify with the endpoint (1 request)"
before anything is spent; built-ins keep pi's free answer. The answer body is
discarded (a test pins that no completion text leaves).

Two defects found while verifying, both fixed here: the status line under the
two buttons collapsed into a column of three words (buttons now own their row),
and the line never updated because the state objects used `line` while the JSX
read `text`. Also: redaction now needs a key long enough to be one — a
two-character key turned "max_tokens" into "max_to***ens" in the message a
person has to read.

Verified on a scratch instance with a fake gateway serving every state:
screenshots read for the hint, the kwargs editor, the bad-JSON refusal, verify
success (desktop and mobile), quota and refused-key failures, and the roster
row's verdict; `__picodeOverlayAudit()` ok each time, no clip, no horizontal
overflow. Evidence in `var/screenshots/cx-*.png` (not committed).

visual-card: 1 yes · 2 yes · 3 yes · 4 no · 5 yes (5/5)

## Next up

- Other CLIs' custom-endpoint files (its own topic note).

## Debts

- Built-in Verify still answers from credential presence.
- `string-thinking` on a non-OpenAI transport is listed but not verified.
