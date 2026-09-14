# 2026-09-13 — snippets-mobile: the phone editor catches up with the desk (v2)

Shipped: mobile `SnippetEdit` now has the studio's authoring surface — the
placeholder table (per-card default, Optional, enum choices; a reserved name
says where it comes from), live validation with the error named under the body
and Save off, the address check (`?except=` when editing), enums riding the
draft into `placeholders[]`, and the last-good-parse fallback dimming the
table while the body is broken. Drafts stay under the shared keys: a phone
draft is the draft the desk sees.

Two phone decisions, in `docs/architecture/snippets.md`: a card per
placeholder instead of a `<table>` (four columns at 390px scroll or squeeze
the name), and a required placeholder showing a disabled default field reading
"Required — the agent asks" — the sentence is inside the field, so the dimmed
control is not a mystery.

Verified live at a real 390×844 viewport: table with 2 editable rows and
`{{branch}}` reserved; optional toggle rewriting the body to `{{repo=}}`;
enum `dev, prod` reaching `placeholders[]` on save; expand refusing `staging`
(400) and accepting `prod`; taken address (red line, both saves off, role
alert); invalid body (error named, table dimmed to the last good parse).
Three screenshots read. The shared logic carries the 16 `snipDraft` tests; the
new mobile code is UI-only, so the live QA is its verification. Docs: the
phone section, plus a stale "Command kind has no control yet" removed.
Merge: fast-forward ready.
