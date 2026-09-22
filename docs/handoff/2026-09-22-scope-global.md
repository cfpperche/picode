# 2026-09-22 — feat/scope-global: the user scope label is "Global"

Shipped: the user-level scope label reads "Global" everywhere it renders — the scope chip, package/MCP card tags, fallback buttons and the pi-settings 400
message; scopes now read `Global / This workspace / This agent`. Backend label specs in `internal/{clipkgs,clisettings,climemory,mcp,pkgs,server}`, web JSX in
`web/{browser,mobile}/src/components/{Packages,Mcps,GuestPackages,PiSettings,PiKeys}.jsx`, shared-domain fixtures `web/shared/domain/{cliNative,piRows}.test.js`,
the six docs-site guides (`browser-tool`, `configure`, `integrations`, `mcp`, `packages`, `settings`) + `docs/architecture/cli-settings.md`, fragment
`docs/changelog.d/scope-global.md`. Unchanged on purpose: pairing "This machine is trusted", server-bind "This machine only", the UserMenu hostname subtitle,
vault/tmux prose.
Verified: `go build ./...` ok; `go test` on clipkgs/clisettings/climemory/pkgs/mcp/server ok; web `npm test` 962/962 pass; `make ci-scoped` PASS
(fmt,vet,hooks,go[13],test-js,build,docs; 30 paths, GO=1 WEB=1 DOCS=1 METADATA=1). Visual on the scratch instance (qa-scratch scope-label, port 8471):
Codex → Packages chip reads "Global"; Claude Code family reads `Global / This workspace / This workspace (local)`; screenshot
`var/screenshots/scope-global-codex.png` (not committed). Blind spot: one scratch instance in Chromium only — no Firefox/Safari, no phone-width pass.
Adversarial review (independent subagent): PASS after one defect fixed — the pi-settings 400 still named the old layer; its guide-prose RISK items fixed too.
visual-review: PASS — chip and scope rows read correctly on the scratch instance screenshot above.
Not done / debts: none known.
Merge: one commit `clis: the user scope chip is "Global", not "This machine"`, merged `main` as bffc87ec; landing loop from the root still owed — merge
`main` again, rerun `make close`, land fast-forward.
