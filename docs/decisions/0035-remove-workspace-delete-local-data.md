# ADR-0035: Remove workspace can delete the local folder — opt-in, typed confirmation

- **Status**: accepted
- **Date**: 2026-08-31

## Context

Removing a workspace has always been registration-only: "The project
folder on disk is untouched" (store contract, ADR-0011/0026 era). Now
that a workspace can be *born* from a clone (ADR-0034), the symmetric
exit exists too: the owner clones a repo to try something and wants the
folder gone when the experiment ends — without dropping to a shell for
`rm -rf`. The reference UX is GitHub's repository deletion: a checkbox
is not enough for an irreversible delete; the user types the name.

## Decision

The Remove-workspace dialog gains a checkbox — "Also delete the project
folder on disk (<path>)" — which, when checked, reveals a typed
confirmation: the Remove button stays disabled until the user types the
folder's name exactly. The request then carries `?files=1&confirm=<name>`
and the server re-verifies: the typed name must match the folder's
basename, and the filesystem root and the home folder are refused no
matter what was typed. Validation runs before anything is stopped or
removed, so a refused delete leaves the workspace whole. After the
record is removed, `os.RemoveAll` deletes the folder; a failure reports
"workspace removed, but deleting the folder failed" instead of
pretending. A remote repository is never touched — this is a local
`rm` only, and the copy says so.

## Refuse

| Asked for | Refused because |
| --- | --- |
| Delete by checkbox alone | Irreversible; the typed name is the point (GitHub's pattern exists because checkboxes get clicked). |
| Trash / undo | A cross-platform trash dance (WSL included) for a power tool is more machinery than the risk warrants; the typed gate is the safety. |
| Touching the remote repository | Never. Out of scope by definition. |
| Client-side-only confirmation | The server re-checks `confirm` against the basename and refuses root/home — the browser is not the trust boundary. |
| Same option on agent removal | An agent's work folder already has the "Also delete the work folder" cleanup choice; project folders belong to workspaces. |

## Consequences

- The first real file deletion in the product. The blast radius is one
  workspace folder, gated twice (typed match client-side, re-verified
  server-side) and guarded against root/home.
- The record is removed before the files: a failed `RemoveAll` leaves an
  unregistered folder on disk (reported in the error), never a
  registered workspace pointing at nothing.
