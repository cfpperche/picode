# 2026-09-22 — feat/packages-opencode: the transport declaration is per verb, and OpenCode's removal is PiCode's own write

Shipped: `pkgs.Caps.Lane Transport{Install, Remove, Update, Marketplace}`
(`internal/pkgs/pkgs.go`) replaces the single `Caps.Async` bool — true where the
CLI reaches a verb with its own command (`clipkgs.Commands`), false where its
only path is PiCode's edit of its own config file (`clipkgs.Writes`). OpenCode
declares `{install:true, remove:false}`: its removal runs through the writer
`clipkgs.Run` already had, hands back the empty line it ran, and
`POST /api/cli-packages/remove` answers the CLI's fresh list with 200 where it
answered 400. The pane reads the same declaration (`laneMutation`,
`web/shared/domain/cliPackages.js`) and draws the removal transcript Pi's direct
mutations draw. Docs: `docs/architecture/packages.md` (transport, open work),
`routes.md`, and the paid debt bullet in `docs/handoff/open/packages.md`.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[6], test-js, build — the
`docs/` Markdown is the metadata scope, so no site/Vale stage); the engine,
contract, route and JS tests the commit names; a scratch instance drew the
removal transcript over a two-module `~/.config/opencode/opencode.json`, and the
file came back with only the named module gone (`var/screenshots/`).
Blind spot: the transcript's title still says "package" where the vendor surface
says "plugin" — the word the direct path has always used — and the mobile pane
was not walked on a device.
visual-review: PASS (desktop pane, scratch instance, stopped afterwards)
Merge: fast-forward ready once main is merged in (main had moved).
