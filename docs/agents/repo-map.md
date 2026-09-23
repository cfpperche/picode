# Repo map

> Reference for [AGENTS.md](../../AGENTS.md).

```
AGENTS.md          this contract
docs/              living documentation (handoff.md = generated view, `make handoff`;
                   handoff/ = session notes; handoff/open/ = durable next/debts per topic;
                   architecture/ = one file per subsystem; changelog.d/ = changelog fragments)
docs-site/         public docs (VitePress Markdown → GitHub Pages)
docs/decisions/    ADRs — one decision per file, immutable once accepted
docs/screenshots/  frozen visual history (ADR-0086); new evidence stays in var/screenshots/
.pi/               Pi harness: skills, project settings, roles
cmd/picode/        entrypoint
cmd/picode-desktop/  Windows provisioning tool (headless, one command at a time)
ext/               Chrome MV3 extension, sideload (ADR-0043)
internal/browserhost/  native-messaging host + Chrome install
internal/server/   HTTP server + API
desktop-shell/     Tauri 2 desktop shell for Windows (ADR-0120)
connectors/        portable MCP connector definitions (JSON reference configs)
packages/          optional Pi packages (pi-browser, pi-inbox, pi-connector-*, …)
internal/web/      UI loader: from disk by default, embedded with `-tags embedui` (ADR-0023).
                   public/ is Vite output and is NOT committed
web/               Independent desktop/mobile apps + shared contracts/tokens (ADR-0072)
scripts/           gates, close, worktree, changelog assembly, ADR seed, QA scratch instance
.github/           CI
```
