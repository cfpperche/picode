# ADR-0095: Mobile files, editing and Git workflows

- **Status**: accepted
- **Date**: 2026-09-07
- **Supersedes**: only the file editor/tree and Git exclusions in ADR-0044 and ADR-0072; all other decisions remain in force.

## Context

The owner approved adding both the file editor/tree and complete current
desktop Git workflows to mobile after reviewing mobile v2. The original
supervision-only scope no longer meets that request. Copying a desktop rail
and tab strip would crowd the conversation and the software keyboard.

## Decision

Mobile owns full-screen Files and Git tools, opened from a workspace, agent
or terminal. Files moves between folders, search, preview and an editor;
Git moves between changes, history, commit details and pull requests. These
are independent mobile components with lazy loading, existing tokens and
MobileSheet dialogs. The [mobile v2 study](../benchmarks/2026-09-07-mobile-v2.md)
adapts workspace context and progressive disclosure from Paseo and Replit;
the [inspector study](../benchmarks/2026-09-05-inspector-rail.md) supplies the
owner/root contract, changes vocabulary and PR workflow, without its desktop
rail layout.

Reads remain authorized through their workspace, agent or terminal, with
`root` as an equality precondition. Workspace history and commit reads gain
the same contract as agent/terminal reads; all three check a supplied root.
Edits use the existing text API, mtime conflict response and explicit
Save/Discard/Cancel navigation guard. A moved terminal requires an explicit
Follow; a dirty editor cannot silently retarget. CodeMirror and its language
packages reuse the versions already used by desktop; mobile declares its own
dependencies. No presentation crosses the application boundary.

Git preserves the established Prepare, Run when idle and Ask agent channels
(ADR-0078), their root checks and their repository interlock. Complete current
workflows means status/diffs, branch and remote history filters, commits,
worktrees, PR status, Fetch, fast-forward Pull, Push, Commit, Commit and push,
and PR creation. It does not claim backend workflows that desktop does not
have, such as branch mutation, stash, merge/rebase pickers or force push.

## Consequences

Mobile users can inspect and edit a project and finish the existing Git
workflows without switching applications. Editor chunks load on demand.
Mobile owns additional UI and lifecycle tests; desktop contracts remain the
reference for file conflicts and Git execution. A wrong owner or stale folder
would target another project, so request races, root mismatches and navigation
while dirty are explicit acceptance rows in the implementation plan.

The existing file API is not a filesystem sandbox: lexical path validation
follows symlinks, and mtime comparison is not an atomic compare-and-swap.
This scope extension preserves that contract rather than claiming stronger
concurrency or isolation guarantees.

## Alternatives considered

- Keep the supervision-only scope: rejected by the owner's explicit approval.
- Mount desktop editor/inspector components: violates independent applications
  and does not fit a single phone screen.
- Add a second Git execution service: unnecessary; existing terminal and agent
  channels already provide credentials, output and interlocks.
