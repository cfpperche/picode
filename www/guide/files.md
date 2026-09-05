---
description: Browse project files, review changes and edit beside the file tree.
---

# Files and changes

Click the folder path below an agent or terminal, or the folder button on a
workspace, to open its File Tree. The desktop keeps one tab per folder.

Select a file in **Files** to display its content in the panel on the right.
Selecting another file replaces that panel. Images and supported documents
have previews; text files use the editor. Markdown and SVG offer **Preview**
and **Raw**, and **Save** writes your edits. You can also press **Ctrl+S**
(**Command+S** on macOS).

**Changes** lists files changed since the latest commit. Select an item to
review its diff, then choose **Open file** to inspect or edit its current
contents in the same panel. **View diff** returns to the comparison. Deleted
files have a diff but no current file to open.

Switching **Files / Changes** keeps the detail open. Drag the divider to adjust
the tree's width, or focus it and use the arrow keys. In the tree, Up/Down moves
focus, Right expands a folder, Left collapses it or focuses its parent, and
Enter opens the focused item. Closing the detail gives the tree its full width.

If you have unsaved edits, replacing the file, opening a diff or closing the
File Tree asks you to **Save**, **Discard** or **Cancel**. A failed save keeps
your draft visible. If another process changed the file, **Reload** asks before
discarding your edits and reading the new contents. Switching to another
PiCode tab keeps your draft and reading position.

**Refresh** updates the file list and changes. Automatic refreshes preserve
unsaved text. If a terminal moves to another folder, the open File Tree asks
for an explicit refresh before using that folder. Return the terminal to the
original folder to save an existing draft there, or discard it before moving on.

**Reveal** opens the folder in the file manager on the machine running PiCode.
