# ADR-0022: The git graph belongs to a repository, is read-only, and is opened from any cwd

- **Status**: accepted
- **Date**: 2026-08-29

## Context

PiCode shows a branch chip but never the shape of the history. The
question that decides the design is not "workspace or agent" — it is
"is this cwd a repository", and the code already answers it.

1. **Every agent has a cwd.** `store.AgentCwd` (`internal/store/workspaces.go:35`)
   returns `WorkPath` if set, else `~/.picode/work/<agent-id>` for a free
   agent (ADR-0011), else the workspace path. A free agent is not an agent
   without a folder; it is an agent in a folder that is rarely a repo.
2. **`gitinfo.Inspect` already exists** (55 lines) and is already called
   per agent (`internal/server/agents.go:466`) and per workspace
   (`internal/server/workspaces.go:72`). It returns nil when the directory
   is not a repository.
3. **A terminal's cwd is live.** `liveTermCwd` (`internal/server/terminals.go:175`)
   asks `tmux.PaneCwd` and falls back to the record. It moves on every `cd`.
4. **Agents run in worktrees, and worktrees share refs.** Measured in this
   repo: from `.worktrees/session-picker-ui`, `--git-dir` differs but
   `--git-common-dir` is the same `/home/goat/picode/.git`, and
   `for-each-ref refs/heads` lists both branches. Only HEAD differs.
5. **Reads are confined to the owner's cwd** by `relUnderCwd`
   (`internal/server/agent_files.go:222`). Any new surface must keep that.
6. **A unified-diff renderer already exists.** `hunksFromDiff`
   (`web/src/lib/diff.js:58`) treats `diff --git`, `index`, `@@` as meta
   and the rest as add/del/ctx. It is fed by pi tool events today, but
   nothing in it is specific to that.
7. **Study**: [mhutchie/vscode-git-graph](https://github.com/mhutchie/vscode-git-graph)
   (MIT). One runtime dependency, `iconv-lite`, for git output encoding —
   no framework, no charting library. `web/graph.ts` is 913 lines of DOM +
   SVG; the greedy column allocator (`determinePath`, `getAvailableColour`,
   `registerUnavailablePoint`) is 73 of them (58 + 9 + 6). It is the reference
   for layout, not for scope.

## Decision

One graph per **repository**, identified by `git rev-parse --git-common-dir`,
so every worktree of a repo collapses into a single tab that marks **each
worktree's HEAD and names the agents living there**. The graph is offered
wherever `gitinfo.Inspect(cwd)` is non-nil — agent bar, terminal bar,
workspace row — and nowhere else; a free agent gets no entry because
`Inspect` returns nil, not because a rule says so. It is **read-only**:
clicking a commit opens its diff through the existing renderer, and no
git command that writes is reachable from it.

Route by owner, identify the tab by repo:

```
hash    #/git/<t|a>/<id>      the owner that asked (this is what authorizes)
tab id  g:<git-common-dir>    what it is (this is what deduplicates)
```

Two agents in one repo request different hashes and land on the same tab.
The server never resolves a repository from a path in the URL, so the
`relUnderCwd` confinement is untouched.

The browser computes the layout; Go returns data. One endpoint per owner:

```
GET /api/{agents|terminals}/{id}/git?limit=250
  → { key, commits[{hash,parents,author,at,subject}], refs[],
      worktrees[{path,head,branch}], occupants{branch → [agent]} }
```

A graph opened from a terminal **pins** its repository at open time; a
later `cd` leaves the tab alone and offers the new repo as a separate one.

## Consequences

| Condition | Entry offered | Tab | Head marked |
|---|---|---|---|
| Workspace agent, cwd is the repo root | yes | `g:<key>` | its HEAD, labelled with the agent |
| Agent with `WorkPath` in a worktree | yes | `g:<key>` — **the same tab** | its own HEAD, labelled |
| Free agent, cwd `~/.picode/work/<id>` | no | — | — |
| Free agent whose picked folder is a repo | yes | `g:<key>` | yes |
| Terminal, live cwd is a repo | yes | `g:<key>`, pinned | **no** |
| Terminal that `cd`s out after opening | tab stays | unchanged | no |
| Terminal, cwd is not a repo | no | — | — |

- **Easier**: parallel agents stop being invisible to each other. Seeing
  `main` and `feat/x` in one graph, each labelled with the agent that is
  on it now, is information that only PiCode can show — VS Code does not
  know your sibling agents exist.
- **Easier**: the commit diff is nearly free. `git show <hash>` feeds
  `hunksFromDiff`, which feeds `SourceBlock`. Only the split on
  `diff --git` for multi-file commits is new.
- **Harder**: the occupant scan. Marking heads means `ListAllAgents`,
  resolving each cwd, and one git call per agent to find its common dir.
  Cost is unmeasured; `occupantsOf` (`internal/server/cleanup.go:88`)
  is the shape to generalise, cache included.
- **Harder**: two identities for one surface (owner in the hash, repo in
  the tab id). A reader who assumes hash and tab are the same thing will
  be wrong here.
- **Cost accepted**: porting the column allocator means **copying MIT
  source**, and PiCode has never done that. Every third-party thing here
  arrives through `web/package.json`; `NOTICE` carries one line, our own.
  So this needs an attribution shape that does not exist yet — mhutchie's
  copyright in the file header and a line in `NOTICE` — and whoever
  reviews it must treat it as vendored code, not as our own.
- **If wrong**: the entry points are three call sites and the tab is one
  route. Removing them leaves `gitinfo` exactly as it is today.

## Refuse

| Temptation | Why not |
|---|---|
| checkout / merge / rebase / cherry-pick / reset | nothing interlocks a write against an agent mid-turn in that worktree; the interlock is more work than the graph |
| A Git icon in the sidebar | an entry point is not a home; ADR-0015's track already refuses "explorer as the product" |
| A rule that hides the graph for free agents | `Inspect` returns nil; the absence is free, do not encode it twice |
| `#/git/<absolute-path>` | a new unscoped read path; the server would resolve a repo from an arbitrary URL |
| Following the terminal's live cwd | the tab would retarget unasked; the file tab already pins its path |
| Marking terminals as heads | a live cwd makes the label flicker on `cd` |
| Porting Git Graph wholesale | 913 lines for 8 styles and ~20 actions; we want the 73 that place columns |
| Computing the layout in Go | view logic in the backend; React is already the renderer (ADR-0008) |

## Alternatives considered

| Alternative | Why not |
|---|---|
| One graph per owner | the same graph N times; refs are shared, only HEAD differs (Context 4) |
| Key the tab by workspace | an agent's `WorkPath` can be a worktree outside the workspace path |
| Load the whole history | `--max-count` with a load-more, as Git Graph does; the SVG ceiling is unmeasured |
| Four endpoints (log, refs, worktrees, occupants) | the browser needs all four before it can draw one pixel |
| `@gitgraph/js` | it renders graphs you *author* by calling `commit`/`branch`/`merge` — for articles and docs. It does not lay out a DAG read from `git log`, which is the whole problem |
| D3 or another chart library | Git Graph proves it is unnecessary: hand-built `<path>` strings, zero dependencies, 913 lines total |
