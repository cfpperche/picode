# 2026-09-07 — feat/opencode-defaults: Activity on by default + start banner

Shipped: new catalog CLIs import with Activity on (a missing
`enabled.json` key is not false). One-time seed flips empty
default-off rows (OpenCode frozen on first deploy). `launch.sh` and
the OpenCode TUI wrapper print `Starting <CLI>...`; maintenance skips
the banner.

Verified: store decision table; `TestEveryMutationAppendsAnEvent`;
`TestNewWiresOpenCodeActivityByDefault`; launch HUP + banner;
disable-all intercept PATH; `go test` store+server.

visual-review: n/a (terminal banner, no React)

Not done: live Working/Needs you needs deploy + dogfood.

Merge: merged as 06da6771.
