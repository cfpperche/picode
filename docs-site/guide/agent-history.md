---
description: Bring back an agent you removed, while its conversation is still on disk.
---

# Agent history

Removing an agent does not have to be the end of its conversation. While the
CLI still keeps the conversation file, the agent waits in **Agent history**
and you can bring it back, with its setup and its conversation.

- **Where:** the user menu ▸ **Agent history**, the command palette
  (<kbd>Ctrl</kbd>+<kbd>K</kbd> ▸ "Agent history"), or `#/history`. Desktop
  only for now.
- **How long:** as long as the conversation exists. When the CLI (or you)
  deletes it, the agent leaves the history on its own. Its record stays in
  [Outcomes](./outcomes).

## What each row shows

| Column | Means |
|---|---|
| Agent | The name it had |
| CLI | Which CLI it ran, and its model |
| Workspace | Where it was — marked **removed** when that workspace is gone |
| Conversation | The conversation's title or first message |
| Removed | When you removed it |

Open a row to see **where it lives**: the conversation file and the folder
it worked in, each with **Copy**. A folder that no longer exists is marked
**missing**.

## Bring it back

Pick where it comes back (**Bring back to**) and press **Bring back**.

- A Pi agent comes back with its model, thinking level, tools, checklist
  and extra prompt, attached to the same conversation.
- Any other CLI comes back in a terminal that resumes the same
  conversation, with the launch settings it had. Environment variables
  are the exception: PiCode never kept their values, so the message after
  the restore names the ones to set again.
- It works in the same folder as before, even when you bring it back into
  another workspace: most CLIs can only resume a conversation from the
  folder it started in.

It cannot come back when:

| Situation | What to do |
|---|---|
| Its workspace was removed | Choose another workspace in **Bring back to** |
| The folder it worked in is gone | Recreate the folder, then try again |
| Its CLI is no longer installed | Install it from **Agent CLIs** |

A brought-back agent is the same agent: automations and pins that pointed
at it work again. **Undo** on the removal message does the same thing, right
after you remove it.

## Remove from history

**Remove from history** takes the row out; the agent can no longer come back
from here, and its record stays in Outcomes. For a Pi agent you can also
delete the conversation file. Other CLIs' files are theirs, so PiCode never
deletes them. Use the CLI itself if you want them gone.
