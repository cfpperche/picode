# ADR-0074: Files and Changes share the file tree's detail pane

- **Status**: accepted (owner approved the refactoring plan on 2026-09-05)
- **Date**: 2026-09-05
- **Amends**: ADR-0030's file-tab navigation and ADR-0032's Open file action

## Context

Files opened a global editor tab while Changes already displayed a diff beside
the tree. Reviewing several files repeatedly left the navigation surface.
The owner requested the same local detail layout for both lists.

The existing `FilePane` already supports CodeMirror and file previews. Its
path-change effect cleared the document without consulting the close prompt.
Embedding it with a changing path therefore also requires a document lifecycle
and guards on replacement and on the containing tab's close action.

The adaptation from the [Cursor benchmark](../benchmark-cursor.md#density-rules)
is a compact navigation rail with a resizable inspector. PiCode applies this
inside the folder tab opened from an agent, terminal or workspace.

## Decision

One canonical folder still identifies one tab. That tab owns a selection
`{path, mode: file|diff}` and one right-hand detail pane. Clicking a file
selects its content; clicking Changes selects its diff; Open file and View diff
switch the same pane. Selecting the same item is idempotent. Files/Changes
switches only the navigation list. Closing the detail returns the tree's width.

`FilePane` provides an embedded layout and uses the same document controller as
the existing file viewer. The controller retains editable text on failed saves,
serializes writes, rejects late reads, and compares edits against the last saved
text. Refresh preserves a dirty buffer, including one edited after the read
started. An unchanged read preserves CodeMirror's undo history and scroll.
Preview/Raw keeps the editor mounted. Hiding the tree retains the document;
reloading or leaving the browser invokes its native unsaved-change protection.

Replacing the document, switching to a diff, or closing the tree invokes one
Save/Discard/Cancel decision. The global tab close button uses the same guard
as the local panel. A failed or conflicting save keeps the current selection
and draft. Typing during a save leaves those later edits dirty.

Tree requests send the canonical `root` as an optional equality precondition
on the cwd resolved independently through the owner. Browse, text reads/writes,
blobs, status, working diffs, revision blobs and Reveal return 409 if it differs.
The parameter never supplies an alternate folder to read. Existing requests
without it retain their behavior. A background refresh cannot retarget a tree;
an explicit Refresh checks the dirty guard before accepting a new root.

The ADR-0073 Git Graph routes still resolve their optional `worktree` ref
first. On those routes, a supplied root must match that resolved checkout.
File Tree does not send a worktree ref; its requests remain scoped to its owner.

## Decision table and acceptance

| Conditions | Action | Evidence |
|---|---|---|
| Clean file, select another file or diff | Replace the local detail; no global tab | Browser QA |
| Same selection | Keep the current detail | Browser QA |
| Switch Files/Changes or hide/reveal the global tab | Retain selection, draft and scroll | Browser QA |
| Dirty replacement/close, Cancel | Stay in the document | `fileDocument.test.js`, browser QA |
| Dirty replacement/close, Discard | Leave without writing | `fileDocument.test.js`, browser QA |
| Dirty replacement/close, Save succeeds | Save then leave | `fileDocument.test.js`, browser QA |
| Save fails or conflicts | Keep the draft and selection; show retry/reload | `fileDocument.test.js`, browser QA |
| Write pending, request navigation | Await it; decide again only if still dirty | `fileDocument.test.js`, browser QA |
| Edit during a write | Preserve later edits as dirty | `fileDocument.test.js` |
| Refresh while dirty, or edit during a read | Keep the draft | `fileDocument.test.js`, browser QA |
| Clean refresh with unchanged bytes | Keep editor identity and scroll | `fileDocument.test.js`, browser QA |
| Older read returns last, or component was disposed | Ignore it; release unused blob URLs | `fileDocument.test.js`, browser QA |
| File disappears or explicit reload fails | Preserve any existing buffer; show state and action | `fileDocument.test.js`, browser QA |
| Matching root or legacy request, any owner | Read/write through the resolved owner cwd | `TestFileRootPreconditionAcrossOwners` |
| Mismatched root, any owner/route | 409; no write or Reveal | `TestFileRootPreconditionAcrossOwners` |
| Terminal cd with identical relative filenames | Reject stale reads/writes; refreshed root works | `TestFileRootRejectsTerminalCD`, browser QA |
| Git Graph worktree ref plus root precondition | Preserve sibling reads; reject a root naming a different checkout | `TestFileRootWithWorktreeScope` |
| No Git / empty folder / clean Changes | Show files or a compact empty state with an action | Browser QA |
| Unsupported, oversized or inaccessible file | Show one state and a recovery action | Browser QA |

Browser acceptance runs against an isolated fixture through
`scripts/qa-filetree-v2.mjs`. The recorded run passed all 15 groups; reviewed
screenshots and `filetree-v2-qa.json` are in `docs/screenshots/`. The runner
also checks that toolbar controls remain inside their pane at narrow widths.

## Consequences

Review and editing stay beside the navigation. No editor, rendering or state
management dependency is added. Keyboard navigation follows visible hierarchy;
the split supports both pointer and keyboard resizing. At very narrow desktop
surface widths the rail stacks above the detail. This does not add a mobile
shell route.

The document lifecycle is now shared code and needs regression coverage for
existing previews and the standalone file surface. Root preconditions add a
small server contract: a terminal that changes directory must be refreshed
deliberately. A buffer from its former cwd may require returning the terminal
there before saving, or an explicit discard before retargeting.

## Alternatives considered

| Alternative | Reason not selected |
|---|---|
| Only replace the file-click callback | Loses drafts on the next selection and bypasses the outer tab's close action |
| Implement another viewer inside the tree | Duplicates CodeMirror, previews and conflict handling |
| Read from the root supplied by the browser | Would replace owner authorization with a caller-controlled filesystem address |
| Save automatically before navigating | The user must choose whether edits reach the filesystem |
