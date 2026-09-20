# 2026-09-15 — x-oauth-popup: a sized popup is a window, not a tab

Shipped: `on_new_window` (btab.rs) allows a `window.open` with window
features to the runtime — a real WebView2 popup on the same profile, with
`window.opener` intact, which is what OAuth needs. Requests without features
(`target=_blank`, plain `window.open`) still adopt as editor tabs. Plan,
topic trap and two comments updated.
Verified: `cargo xwin build` ✓ (the COM closure has no host test);
`make close` (Go/JS comments only outside the shell).
visual-review: n/a (no HTML surface changed).
Not done: the live x.com + Google login — needs the new shell on Windows.
Merge: fast-forward ready.

## Next up

- Owner: after a shell restart, sign in to x.com with Google — a small
  separate window should open and the login should land in the tab.

## Debts

- The popup rule (feature-sized → window, else tab) has no automated test:
  the decision lives in the Windows-only `on_new_window` closure.
