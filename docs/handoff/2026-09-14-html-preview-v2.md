# 2026-09-14 — feat/html-preview-v2: study for the second preview origin (docs only)

No code. `docs/plans/html-preview-v2.md` is the study the owner asked for:
what v2 buys (own storage, workers, own cookie), what changes in the threat
model, four origin options with a recommendation, and five decisions waiting
for the owner (D1–D5) that become ADR-0137.
Grounded in the code — `auth.setCookie` (host-only `picode_session`,
`SameSite=Strict`), `auth.originAllowed` (`Origin == r.Host`, and
`Sec-Fetch-Site: cross-site` refused first), no CORS on any route, `/preview`
outside the auth gate — and measured in the QA Chromium against a throwaway
probe server: `*.localhost` resolves for the browser only (`getent`/`curl`
do not), and the split that decides the design is `<label>.localhost:P` →
`Sec-Fetch-Site: cross-site`, **no cookie**, versus `localhost:P2` →
same-site, **cookie sent**.
Recommendation: **B** — per-ticket `<label>.localhost` on the existing port
(Host routing, storage fresh per preview, sandbox fallback when the host does
not resolve), v2 local-only.
`docs/plans/html-preview.md` v2 bullet now points at the study.
Verified: docs only — `make close` gates PASS; no Go/UI change, no screenshots.
