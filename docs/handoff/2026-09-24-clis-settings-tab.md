# 2026-09-24 — feat/clis-settings-tab: Agent CLIs gets a Settings tab
Shipped (e7fc93c46): Agent CLIs has a third tab, Settings (#/clis/settings,
desktop + mobile), holding the tmux guard and browser hand-off switches
(SurfaceWrappers); they left the top of the CLI catalog. Owner asked for it.
Loading and empty states added; toast copy says "tmux guard" (matches the
switch label); the "How it works" link uses a non-breaking space.
Owner approved retiring the Pi-era settings addresses that defaulted to Pi:
#/settings, #/more/settings, #/clis/settings/<cli> (#/clis/settings is now the
tab) and the ?tab=keys redirect. cliSettingsLocation parses only
#/clis/<cli>/settings. The dead mobile More section "settings" is gone.
Docs: architecture cli-settings/routes/terminal-bridge, docs-site guide pages,
changelog.d fragment.
Verified: `make ci-scoped` and `make close` green. Scratch instance, desktop +
mobile widths, switch toggled. Blind spot: not run inside the Windows shell.
visual-review: PASS (clisset-desktop, clisset-clis, clisset-mobile,
clisset-toggled; overlayAudit ok; card 5/5)
Not done: the same Pi-defaulting legacy pattern remains for packages
(#/packages, #/more/packages, #/clis/packages) and connectors (#/mcps,
#/integrations, #/clis/connectors); palette go("settings"/"packages"/...) falls
back to Pi with no agent selected — the owner's call. The user-menu search for
"tmux guard" still lands on #/clis, not #/clis/settings.
Merge: fast-forward ready.
