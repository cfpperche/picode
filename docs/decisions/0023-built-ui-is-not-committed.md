# ADR-0023: The built UI is not committed; embedding moves behind a build tag

- **Status**: accepted
- **Date**: 2026-08-30

## Context

`internal/web/public/` holds Vite's output and is tracked in git. It has been
since `9d268755`, the bootstrap commit, before React, Vite or CI existed. No
ADR ever decided it — ADR-0001 and ADR-0008 say the UI ships inside the binary
via `go:embed`, which is about the *artefact*, not about what git carries.

Measured on 2026-08-30:

| Fact | Value |
|---|---|
| Tracked files under `internal/web/public/` | 330 |
| Size in the working tree | 33 MB |
| Commits touching them | 336 of 439 — 77% of the repo's history |
| Files rewritten per UI commit (last 10) | 133 on average |
| Conflicts in two merges that day | 335 and 334, every one a `rename/rename` |

Every one of those conflicts resolves the same way — delete both sides and
rebuild — because the content is derived. Since AGENTS.md #5 puts each agent in
its own worktree, parallel work makes this the dominant cost of merging.

The constraint that kept it: `go:embed public` fails to compile when the
directory is missing *or* empty (verified: `pattern public: no matching files
found`, and `cannot embed directory public: contains no embeddable files`). So
dropping the files from git would break `go build`, `go vet` and `go test` on a
fresh clone, and `make ci` runs `test` before `build`.

**What the field does.** Six comparable Go projects that embed a frontend, and
none of them commits the build output:

| Project | Receipt |
|---|---|
| Grafana | `.gitignore`: `/public/build` |
| Prometheus | `.gitignore`: `/web/ui/static` |
| HashiCorp Vault | `.gitignore`: `ui/dist` |
| Coder | `.gitignore`: `dist/`, `node_modules/` |
| Gitea | `public/assets/js/index.js` → 404 at HEAD |
| Syncthing | `lib/auto/gui.files.go` → 404 at HEAD |

Prometheus also answers the constraint, and answers it better than working
around it. `web/ui/ui.go` carries `//go:build !builtinassets` and serves the UI
**from disk**; `assets_embed.go` carries `//go:build builtinassets` and does
the embedding. The default build does not embed at all.

## Decision

Stop tracking Vite's output, and make embedding a property of release builds
rather than of every compile, following Prometheus.

1. `internal/web/public/` is ignored by git.
2. `internal/web` gains two files: the default one (`//go:build !embedui`)
   serves the UI from `internal/web/public` on disk; the tagged one
   (`//go:build embedui`) keeps today's `go:embed`.
3. `make build` and the release workflow build with `-tags embedui`. The
   shipped binary is unchanged: one file, UI inside (ADR-0001 stands).
4. `make dev`, `go build`, `go vet` and `go test` need no tag and no Node. On a
   clone with no UI built yet the server says so in one line instead of serving
   404s.

## Consequences

- **Easier**: a UI change is a diff of the files a human wrote. Merges between
  parallel agents stop producing hundreds of conflicts that mean nothing. The
  repository stops carrying 33 MB of derived bytes.
- **Easier**: a contributor who only touches Go no longer needs Node to run the
  tests — today `go test` cannot compile without the built UI present.
- **Harder**: two build modes exist, and the tag has to be right. A binary
  built without it serves from disk and is not portable; `make build` is the
  contract, as ADR-0008 already said.
- **Harder**: `go install github.com/cfpperche/picode/cmd/picode@latest` stops
  producing a working UI. It works today by accident, is not in the README, and
  does not work for Prometheus, Grafana or Vault either. Releases are the
  supported path (`release.yml`).
- **If wrong**: `git add -f internal/web/public` and drop the tag. Nothing in
  the serving path changes shape — only where the files are read from.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep committing, add `make resolve-ui` to rebuild on conflict | treats the symptom; the 133-file diffs and the 33 MB stay |
| Keep committing, drop the content hash from filenames | the service worker caches **only** hashed `/assets/`; this would break that contract for a merge convenience |
| Untrack and require `make web` before any Go command | what six peers avoid: it forces Node on contributors who only touch Go |
| Untrack and commit a placeholder so `go:embed` still compiles | works (verified: `all:public` plus `.gitkeep` compiles), but keeps a fiction in the tree and still embeds in dev. The build tag is the pattern the field converged on |

## Open question

Whether the service worker behaves identically when assets are served from disk
rather than from the embedded FS. It caches only hashed `/assets/`, and the
paths do not change — but this has not been exercised. Recorded in
`docs/handoff.md`; it gates calling this done, not calling it decided.
