# 2026-09-18 — webapp-clear-retry: the clear survives the webview teardown

Fix: owner run on Windows — Clear data failed with the generic fallback.
Two causes stacked: the partition folder stays locked for a while after
the webview closes (three 300 ms retries lost that race), and Tauri
rejects `Err(String)` as a raw string, which `webappError` masked behind
"Could not save the web app". Now the removal has a ten-second budget
(20 × 500 ms) and shell error strings pass through `webappError`
untouched (test added).
Verified: 10 JS tests; `cargo xwin check --release` clean; ci-scoped
PASS. visual-review: n/a (same dialog; message text only).
Merge: fast-forward ready.

## Next up

- Owner on Windows: Clear data on a signed-in app — now expected to succeed within ~a second of teardown; if the ten-second budget ever runs out, the dialog shows the real reason.

## Debts

- docs/handoff/open/installed-webapps.md: standalone window stays refused (unchanged).
