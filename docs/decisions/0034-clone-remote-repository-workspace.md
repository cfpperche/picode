# ADR-0034: Clone a remote repository into a new workspace

- **Status**: accepted (amends ADR-0022's read-only stance with one narrow exception)
- **Date**: 2026-08-31

## Context

A workspace is born by pointing at a folder that already exists
(`store.AddWorkspace` stats the path). Starting from a remote repository
meant leaving the ADE, running `git clone` in a shell, and coming back —
exactly the round-trip an ADE exists to remove.

Benchmarks agree on the shape. VS Code (`Git: Clone`), GitHub Desktop and
JetBrains all put "paste a URL" and "pick a local folder" inside the same
entry point; the destination is derived from the repo name, pre-filled,
and — in the tools that get it right — editable (VS Code only lets you
pick the parent and has an open issue asking for an editable name).
Conductor learned publicly that demanding a GitHub App with broad
permissions causes backlash, and now runs the user's own git/ssh
credentials. Agent managers that skip a URL field entirely (Terragon,
Codespaces, claude.ai/code) do so because their clone lands in a cloud
sandbox — not our model (ADR-0007: personal machine / tailnet).

Two constraints in this codebase shaped the decision:

- **ADR-0022 declares git read-only**: "no git command that writes is
  reachable" from the GUI, because nothing interlocks a write against an
  agent mid-turn in a worktree. A clone into a directory that does not
  exist yet cannot collide with any agent — no agent can have a cwd in a
  folder that isn't there. The interlock argument does not apply, so this
  ADR carves that one narrow exception rather than weakening the rule.
- **No job framework**: the server has no SSE and no job queue. The
  long-request precedent is `handleAgentBash` (blocking handler, 10-minute
  timeout). Typical repos clone in seconds; streamed progress is a v2
  concern, not a reason to invent machinery now.

## Decision

The New-workspace dialog gains a source control: **Local folder | Clone
repository** — one form, two origins, defaulting to the existing local
flow. Remote mode asks for the repository URL, a name and a destination;
name and destination derive from the URL while untouched (the last-used
parent folder is remembered in `localStorage`, falling back to `~/code`),
and a manual edit stops the derivation. Submit says **Clone**, then
**Cloning…** while one blocking `POST /api/workspaces/clone` runs.

The server side lives in a new package, `internal/gitclone`, so
`gitgraph` keeps its read-only contract. `ParseRemote` validates the URL
the way `pipkg` validates package sources — shell metacharacters and a
leading dash are refused (the URL becomes argv; `git clone` also gets a
`--` separator), `https://`/`ssh://`/`git://`/scp-like are accepted,
`file://` and local paths are not remotes, and a `/tree/<branch>` suffix
pasted from a browser resolves to `--branch`. The clone runs with the
host's own credentials and every interactive prompt disabled
(`GIT_TERMINAL_PROMPT=0`, neutralized askpass, `ssh -oBatchMode=yes`), so
a missing credential fails in seconds with a classified, actionable
message instead of hanging the request.

The destination must not exist or must be empty — with one courtesy: a
non-empty destination whose `origin` already names the same repository
(normalized across https/ssh spellings) is **adopted** as the workspace
instead of failing. Anything else occupying the folder is a 409. On
success the folder goes through the same `store.AddWorkspace` as the
local flow: the workspace starts empty (ADR-0027), decorated by the same
`gitinfo` branch chip and dirty badge as any other.

## Refuse

| Asked for | Refused because |
| --- | --- |
| OAuth / GitHub App / account repo list | Trust model is the host's own git (`docs/architecture.md` security model); Conductor's permission backlash is the cautionary tale. The URL field is the escape hatch that always works. |
| Asking for a token in the UI | Never. Credentials belong to the host's git/ssh setup. |
| Shallow clone by default (`--depth`) | Breaks the git graph (ADR-0022) and every later operation that needs history. |
| `--recurse-submodules` | v1 clones the repository the URL names; submodules are the user's call afterwards. |
| Branch selector UI | A pasted `/tree/<branch>` URL is honored — that covers the real workflow without another field. |
| Streamed clone progress / live terminal | Needs SSE or a job frame the server doesn't have; blocking + honest "Cloning…" first. Candidate for v2 via a tmux session. |
| Cloning into a non-empty unrelated folder | Ambiguity between merge and overwrite; 409 with a clear message instead. |
| Cancel-on-close | Closing the dialog abandons the request, not the clone; git finishes or fails atomically and the workspace shows up on the next list. |

## Consequences

- The user's URL becomes a subprocess argument. `ParseRemote` plus the
  `--` separator is the whole defense; both are sabotage-tested.
- A private repo without credentials on the host fails with "run
  `gh auth login` or add an SSH key" — by design there is no prompt.
- The HTTP request can hold for up to 10 minutes on a huge repo with no
  visual progress beyond "Cloning…". Accepted for v1; v2 may stream.
- ADR-0022's "no reachable git writes" is now "no reachable git writes
  except `clone` into a fresh/empty directory" — the interlock rationale
  is untouched because no agent can occupy a directory that doesn't exist.

## Alternatives considered

- **Extend `POST /api/workspaces` with a `url` field** — different
  timeout, different failure modes, different success codes; overloading
  the 200ms registration endpoint with a 10-minute one helps nobody.
- **tmux session + terminal tab for progress** — the most PiCode-native
  streaming, but registering the workspace needs completion detection
  (watcher/poller) that doesn't exist; deferred to v2, not discarded.
- **GitHub account integration with a repo dropdown** — see Refuse.
