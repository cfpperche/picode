# 2026-09-10 — browser-input: consent-gated control for the browser surface

Shipped: ADR-0115 (amends ADR-0114's read-only constraint); `/browser-input`
flag+command in pi-browser-capture (branch-persisted, mirrors live consent to
`<session>.capture/input.json`); `BrowserInputConsent` (500 ms TTL) +
`WriteInputConsent` in internal/rpc; `POST /api/agents/{id}/browser-input`
flips consent; `/ws/browser` forwards `input_mouse`/`input_keyboard`/
`input_touch` only while consented (else `watch-only` refusal envelope);
the surface's Watch-only chip is the toggle; pointer/keyboard/touch map
through the letterbox into viewport coordinates
(`web/shared/domain/browserInput.js`).

Verified: ci-scoped green; sidecar tests 5/5; shared helpers 5/5; proxy tests
cover consent-on forwarding, watch-only refusal, unknown-type refusal; scratch
E2E with a logging fake engine: chip toggle flipped the mirror (`on`→`off`),
a canvas click arrived as mousePressed/Released at exactly (640, 400), typed
keys arrived as `char`/`keyUp`. Screenshots read (control-on state); overlay
audit ok.

visual-review: PASS (browser-control-on.png; card 5/5)
Not done / debts: the live prompt test also revealed managed-mode prompts do
not run slash commands (recorded in ADR-0115); `keyboard type` via the QA CLI
uses insertText (no keydown) — real keys are trusted browser input; no remote
scroll (engine protocol has no wheel); one unreproduced rpc test FAIL in the
first ci-scoped run (clean on 3 reruns, -race, and shards).
Merge: main merged in; ff-ready after close rerun.
