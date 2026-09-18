# 2026-09-18 — webapp-clear-data: Clear data per installed web app

Shipped: the tile menu gains **Clear data** (desktop only, confirmed) —
the shell closes the app's webview and removes its partition folder
(`btab_clear_app_data`; containment by `browserlab::app_partition`, only
`app-<webappId>` ids name a folder; removal retries past the close
teardown, then fails honestly). Remove clears the folder too, best
effort from the shell (plain-browser removal leaves an orphan folder,
reclaimed on re-install — accepted). ADR-0153 amendment records it; the
public guide (`docs-site/guide/web-apps.md`, previous branch) gains the
section. Settings ▸ Browser clear-data keeps its work-profile scope.
Verified: `cargo xwin check` clean; 9 JS tests; build green; ci-scoped
PASS. visual-review: n/a from WSL (the command is shell-only) — the
confirm dialog follows the Remove dialog's proven pattern; owner
exercises it from the desktop.
Merge: fast-forward ready.

## Next up

- Owner on Windows: tile ⋯ → Clear data on a signed-in app — signs out that app only, others untouched.

## Debts

- docs/handoff/open/installed-webapps.md: standalone window stays refused (unchanged).
