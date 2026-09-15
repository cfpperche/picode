# 2026-09-15 — browser-permissions-shell

Second half of Browser permissions (the data half landed in
feat/browser-permission-store).

## Done

- Shell: `attach_permission_handler` on every tab — the platform's kind is
  named (the daemon's closed list), the policy decides
  (`btab_set_permission_policy`, allow/deny/default), and the outcome is
  emitted as `btat://permission` with the site URI. A kind with no policy
  keeps the platform's own default (deny) and is reported as such.
- The ACL's three edits, this time in one pass: `generate_handler!`,
  `build.rs`, `capabilities/default.json`, and the generated permission file.
  cargo xwin ✓.
- App.jsx records each report through `POST /api/browser/permissions`.

## Not built yet

- The **Site settings dialog** (the visible half): kinds with their policy,
  the per-site list with Reset/Allow/Block, and the **Ask prompt** — which
  is where the deferral comes in (`GetDeferral`, held until the user
  answers in the tab). Until then the policy is only settable from code and
  every unset kind denies, exactly as the platform did before.
- The note in `docs/handoff/open/work-browser-tabs.md` carries the measured
  API and the scope; the JSON field is `origin` and carries the page URI.
