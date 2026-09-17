# 2026-09-17 — engine-permission-state

The owner's live Ask test exposed this: after answering **Allow**, reloading
never asked again. Half of that is normal, half was ours.

## What the test proved (all live, production store)

`browser_permissions` held exactly the three rows the decision table
promises: the `*` camera **ask** standing (policy), a **deny** from the 60 s
watchdog (log, `standing=0`) and the owner's **allow** (log) — an Allow-once
never became permanent. The page reported `granted`, then the watchdog
denial as `NotAllowedError`.

## The gap, and the fix

"Once allowed, a reload does not re-prompt" is Chromium's behaviour, not a
bug. But **Reset** was: `btab.rs` never called
`ICoreWebView2Profile4::SetPermissionState`, so the engine's per-origin
memory outlived the row the dialog deleted — "forget this decision" until
the app restarted.

- `apply_engine_state(app, kind, origin, state)` — a concrete origin is told
  to the engine (`allow`/`deny`/`default`); `*` is skipped (the engine has no
  per-kind concept) and an unknown kind is ignored. The profile is shared, so
  one call covers every tab; the main window's profile is reachable with no
  browser tab open.
- Called from `btab_set_permission_policy` (the dialog's write, including
  Reset, which already passes the site) and from `btab_permission_answer`
  when `remember` ("Always").
- Compiles (`cargo xwin build`, 1m42s); ACL unchanged (no new command).

## Not verified

The live run needs the swapped exe (`make desktop-restart`): Reset a site's
camera decision, reload the page, and the bar must appear again. Nobody has
seen that yet — the code path is compile-verified only.
