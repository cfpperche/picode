# ADR-0204: The Instructions tab proposes instruction-file fixes and writes them only on confirmation

- **Status**: accepted (owner, 2026-09-23)
- **Date**: 2026-09-23
- **Boundary**: persistence — PiCode authors content in the user's repository (a line in `CLAUDE.md`, a new personal file, a `.gitignore` entry), where until now it wrote only text a person typed in the editor

## Context

The Instructions tab (`docs/architecture/cli-instructions.md`) reads what each
agent CLI loads and reports findings, and it never writes. The most common
finding has a one-line fix that a person must still know how to write: on this
machine, 3 of the 5 repositories that carry both files have a `CLAUDE.md` that
tells Claude Code in prose to read `AGENTS.md`. Claude Code does not load
`AGENTS.md` then, and its own docs name the fix: an `@AGENTS.md` import line.
The second is a personal file: `CLAUDE.local.md` or `AGENTS.override.md`,
which belongs in `.gitignore`.

PiCode already writes into repositories, but only text a person typed: the
file editor's save (`PUT /api/workspaces/{id}/text`, `agent_files.go`) refuses
a stale write by modification time and never creates a file. What is new is
PiCode proposing the content itself, and creating files. That changes what a
person can trust about their tree, which is why the owner asked for this
decision before any write (2026-09-23).

The study behind the tab (`docs/benchmarks/2026-09-23-agents-md.md` §9)
already refused three temptations, and they hold here: a second editor, a
PiCode-owned rules store fanned out into copies, and generating `AGENTS.md`
with PiCode's own prompt (the ETH study found no general gain and over 20%
added cost).

## Decision

A finding may carry a **proposed fix**. The fix is a small, fixed edit
computed by the server from the files on disk. The tab shows the exact diff
per file and writes it only when the person confirms. The first set is two
fixes:

- **Bridge.** In a `CLAUDE.md` that points to `AGENTS.md` in prose, add
  `@AGENTS.md` as the first line; the prose stays. If the `CLAUDE.md` holds
  nothing but that sentence, the diff replaces the sentence with the line.
- **Personal file.** Create `CLAUDE.local.md` or `AGENTS.override.md` with no
  text, and add its name to the repository's `.gitignore`, creating that file
  when it is missing.

Every write follows the same rules:

- It is confined to the workspace folder.
- The diff shows what will be written, byte for byte.
- The write is refused when any file changed since the diff was computed,
  compared by content hash, not modification time.
- It never overwrites an existing personal file.
- It never runs git: the change shows up as an ordinary unstaged change in
  the Git tab, for the person to review and commit.

No fix is applied without that confirmation, one fix at a time.

## Consequences

- The most common finding becomes one click plus a review, with the change
  visible in the Git tab like any other edit.
- The fixes are code with tests: each one is a pure function from the files
  on disk to a diff, pinned by fixtures, and the write path is shared.
- A new write endpoint joins the file editor's.
  `POST /api/workspaces/{id}/instructions/fix` takes a finding id and the
  hashes the diff was computed from. It answers 409 when the tree moved, and
  the tab offers the fresh diff.
- If we are wrong about a fix's content, the cost is one unwanted line in a
  file the person reviews before committing, and the Git tab can discard it.
  Because git is never run, nothing reaches history without the person.
- Harder: every future fix needs the same care (a pure diff, a fixture, a
  refusal on drift). A fix that cannot be expressed as a small, exact diff —
  merging two long files, trimming a file under a limit — is not offered
  here. It stays a finding with "Open file".

## Alternatives considered

- **Write without a diff (one-click apply).** Faster, but the person learns
  what changed only from the Git tab afterwards. That conflicts with "a
  person approves before behaviour changes" (ADR-0194's study) and with the
  honesty bar.
- **Symbolic link `CLAUDE.md` → `AGENTS.md`.** Claude's docs accept it, but a
  symlink needs Administrator rights or Developer Mode on Windows, and git
  checks it out as a one-line text file there. PiCode ships a Windows desktop
  shell.
- **Run the CLI's own `/init` as an agent.** It writes a whole file from a
  model. Useful as a draft, but the evidence says generated context files do
  not help and cost more, and it is not a small, exact diff. It is left to a
  later decision.
- **Commit the fix.** Saves a step, but PiCode would author commits in the
  user's repository on its own. Every other PiCode write action goes through
  the person (ADR-0096).
