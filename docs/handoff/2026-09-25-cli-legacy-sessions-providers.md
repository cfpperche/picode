# 2026-09-25 — feat/cli-legacy-sessions-providers: retire Pi-era Sessions and Providers addresses
Shipped (1989b844c): third of the series after feat/clis-settings-tab and
feat/cli-legacy-routes, owner asked. Retired #/sessions*, #/clis/sessions*,
#/providers*, mobile #/more/providers*, #/clis/providers[/<cli>].
cliProvidersLocation is deleted; sessionsLocation and its two rewrites in
cliLocation are deleted; routes.js providersNew and the mobile More sections
providers/sessions mapping are gone. #/providers/llama still opens llama.cpp
(not a Pi alias; kept).
Fixed on the way: #/clis/<cli>/providers/new/<extra> opened Add provider; it
is now an invalid link.
Checked and left: the provider OAuth returnTo (cliProvidersReturnTo) points at
Pi's providers pane — correct, that path is Pi-only (guests sign in their own way).
Verified: `make ci-scoped` and `make close` green. Scratch instance:
#/clis/codex/sessions (empty state) and #/clis/pi/providers render;
providers/new/extra shows "This provider link is invalid."; #/providers/llama
opens llama.cpp; #/sessions and #/providers land on home.
Blind spot: provider OAuth returnTo not exercised through a real sign-in.
visual-review: PASS (clisp-sessions.png, clisp-providers.png)
Not done (owner's call, still Pi-defaulting): palette go() opens Pi's panes
with no agent selected; sessionsHash(ws, cli="pi") defaults to Pi for
sessions without a cli field.
Merge: fast-forward ready.
