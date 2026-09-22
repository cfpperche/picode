# 2026-09-22 — feat/login-open-url: CLI browser opens go to the client the user is looking at
Branch feat/login-open-url (base a064994d, merged main before close; head 3c9295c2 —
'server: route CLI browser opens to the client the user is looking at' + 'docs: regenerate OpenAPI').
Shipped (ADR-0180, terminal browser hand-off): CLI /login used to resolve xdg-open/wslview/BROWSER
inside WSL → distro chromium. Now picode-open + xdg-open/wslview shadows in <dataDir>/bin
(ADR-0056 intercept), BROWSER=… in session env (terminals.go, cli_launch.go), boot-time refresh in server.go.
POST /api/terminals/{id}/open-url validates (osopen.ValidOpenURL) then either
Ephemeral("terminal.open_url") when the feed has subscribers (visible client opens the clicked link —
desktop integrated tab or browser tab; handler web/browser/src/lib/openUrlFeed.js with 1.5s dedupe +
visibilityState) or falls back to osopen.OpenURL (WSL → powershell.exe Start-Process with the URL in
PICODE_OPEN_URL env, never a shell command line). Wiring row 'open-url' defaults on (opt-out beside
the tmux guard); TmuxGuardRow.jsx generalized into SurfaceWrappers.jsx (desktop + mobile).
Verified: ci-scoped PASS; live smoke on scratch loginurl — wrappers installed at boot, SSE frame
terminal.open_url received by a real Chrome client, wrapper inside the pane HANDOFF_EXIT=0, osopen
host path spawned PowerShell, two wiring rows + opened web tab screenshots, overlayAudit ok.
Method's blind spot: not exercised from a CLI launched outside the WSL session env.
visual-review: PASS — var/screenshots/loginurl-wiring.png, var/screenshots/loginurl-opened.png; overlayAudit ok.
Merge: fast-forward ready

## Debts

- two visible clients both open the URL (dedupe is per-client) — belongs in docs/handoff/open/work-browser-tabs.md if it ever annoys
## Next up

- CLI login pages that pass URLs through mechanisms we don't shadow (hardcoded /usr/bin/xdg-open) keep the old behavior
