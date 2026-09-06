# Inspector rail implementation and acceptance

Owner-approved direction (2026-09-05): ADR-0078 accepted with Phase 0 (docs),
Phase 1 (the read-only rail with Changes and Files) and Phase 2 (the PR tab
through the host's `gh`). The rail opens content in the center and follows
the selected tab's owner. Commit is designed in the ADR, with three options
compared against the benchmarks, and waits for the owner's decision.

## Implementation

| Area | Implementation |
|---|---|
| Server | `gitgraph.StatusWithStats` (per-file add/del/binary via one `--numstat -z`, untracked counted in Go, branch, worktree); `gitstatus` pages carry `branch`, `worktree`, `totals` over the re-anchored slice |
| Pure logic | `web/desktop/src/lib/inspector.js` (anchor, layout, grouping, scope, prefs) and `lib/resizeEdge.js` (edge drag math + hook), both under `node --test` |
| Rail | `components/Inspector.jsx` (frame, header, tabs, load/409/feed idioms from FileTreeSurface), `InspectorChanges.jsx`, `InspectorFiles.jsx` |
| Center | `FileSurface` gains a per-tab Diff view (`WorkingDiff`); `FileTree` gains `trailing` and `ariaLabel`; `AgentTabs` gains `endSlot` for the toggle |
| Shell | Mounted after `<main>` in `App.jsx`; `app.inspector.toggle` (`Ctrl+.` / `Cmd+.`); palette action "Toggle inspector"; `#inspector` in the overlay audit |
| Docs pipeline | Fixture seeds a dirty repository under the picode workspace; surface `app-inspector` with profile `desktop-inspector` |
| PR tab | `internal/server/pr.go`: `GET …/pr` for the three owners (states in 200, minute cache, gh's login as the credential) and `POST /api/terminals/{id}/type` (literal keystrokes, no Enter); `InspectorPR.jsx` + `usePullRequest`; `tmux.TypeText` |
| Git actions (stage 1) | `gitstatus` adds `upstream`, `ahead`, `behind`, `detached`; the chip shows `↑ ↓` / unpublished / detached; a Git menu prepares Fetch, Pull, Push, Commit, Commit and push, Create PR in an idle terminal of the folder (`gitActionCommand`, `shellQuote`, `commitMessageSchema`, `InspectorCommitDialog.jsx`) |
| Run when idle (stage 2) | `internal/server/git_run.go`: `POST /api/terminals/{id}/run {text, root}` types and submits behind an interlock (`repoBusy`: agents mid-turn, TUIs working, automations, terminals working or holding a program, target pane at a shell); 409 `moved` / `foreground` / `busy` naming who; the `type` route gains the same root and foreground guards; opt-in checkbox in the Git menu (`picode-inspector-run`); the app falls back to preparing with a note |

Benchmark adaptation: Paseo's folder-grouped Changes with counts and total,
Orca's project tree and click-to-center, t3code's text tabs; PiCode adds
owner-scoped reads with a pinned root, the This-agent scope and the
shrink-before-hide layout rule. See the
[study](../benchmarks/2026-09-05-inspector-rail.md).

## Decision table

The rows and their evidence live in ADR-0078 ("Decision table and
acceptance"). Automated coverage: `inspector.test.js` (anchor, layout,
grouping, scope, counts, filter, prefs), `resizeEdge.test.js`,
`internal/gitgraph/status_test.go` (`TestStatusWithStats*`,
`TestCountNewFileCapsAndFinalLine`), `internal/server/filetree_test.go`
(`TestGitStatusReAnchorsToTheOwnerCwd` with counts and totals),
`internal/server/pr_test.go` (fake gh on PATH: unauth, none, no remote, ok
with folded checks, cache and refresh; missing gh; plain folder; owner routes
with the root precondition; type-text refusals),
`TestStatusWithStatsUpstreamAheadBehind` (bare remote, ahead/behind,
detached), `gitActionCommand`/`gitActions`/`branchChip`/`shellQuote` and
`commitMessageSchema` node tests, `TestTerminalRunRefusals` (moved,
foreground, busy agent, busy terminals by state and by program, the type
route's guards) and `TestTerminalRunTypesAndSubmits` (a real tmux shell runs
the submitted command).

## Live acceptance scope

Run against a disposable `picode-docs-fixture` on a private port; never a
real workspace path. States to capture and read: Changes with counts (dark),
Files (light), no anchor, blocked terminal (409 + Follow), non-git folder,
the More menu open (overlay audit `ok`), the squeezed window, the ≤767 px
shell, the This-agent scope, and a file opened in Diff and Editor views.

## Observed results

Filled in by the acceptance run; see `docs/screenshots/inspector-*` and
`docs/handoff.md`.
