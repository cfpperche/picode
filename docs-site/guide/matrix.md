# Matrix

A **matrix** is one page that shows many agents and terminals side by side,
live. Each tile is a **panel**: the real screen of that agent or terminal —
the same one its own tab shows — not a picture of it.

Open it from the **Apps** tab → **Matrix**. Matrices are saved on the
server, so every browser signed in to your PiCode sees the same ones.

## Make one

1. **New matrix**, give it a name.
2. **Add panel**, then pick an agent or a terminal from the list. Anything
   already on this matrix is not offered again.
3. Repeat. Panels land in the first free slot.

The switcher at the top left of the surface moves between your matrices;
the **⋯** menu renames or deletes the one you are looking at. Deleting a
matrix never touches the agents and terminals on it.

## Arrange it

- **Move** a panel by dragging its header.
- **Resize** it from the bottom-right corner, or from its bottom or right
  edge.
- Panels settle upwards into the free space above them, so the page has no
  gaps.

The layout saves itself as you go, for everyone. If someone else changed
the same matrix while you were dragging, PiCode reloads it and says
*Matrix changed elsewhere — reloaded.*

## What a panel shows

The header always carries the name, the workspace or the CLI, and the same
word the sidebar uses — **Working**, **Needs you**, **Ready** — even when
the panel's body is asleep.

| The panel is bound to | The body shows |
|---|---|
| a running terminal | the live terminal, ready to type in |
| a stopped CLI terminal | *This CLI terminal is stopped.* — Resume last session |
| an agent working in its terminal | its live screen |
| a managed agent | *Managed agent — open to read.* — Open |
| a stopped agent | *Agent is stopped.* — Run |
| something that was deleted | *That terminal is gone.* — Remove |

**Open** in a panel's header sends you to that agent or terminal's own tab.
The screen follows you there and comes back when you return to the matrix —
there is only ever one of it.

## Keyboard

Click a panel's header once (or press <kbd>Tab</kbd> into the matrix) to
select it. Then:

| Key | Does |
|---|---|
| <kbd>←</kbd> <kbd>→</kbd> <kbd>↑</kbd> <kbd>↓</kbd> | move to the panel next door, scrolling it into view |
| <kbd>Home</kbd> / <kbd>End</kbd> | first / last panel |
| <kbd>Enter</kbd> | start typing in this panel's terminal |
| <kbd>Shift</kbd>+<kbd>Esc</kbd> | stop typing and come back to the panel |
| <kbd>Delete</kbd> | remove the panel from the matrix (with **Undo**) |

Inside a panel every other key belongs to the program running there — a
plain <kbd>Esc</kbd> reaches it, which is why leaving takes
<kbd>Shift</kbd>+<kbd>Esc</kbd>.

## One panel, full width

The **Maximize** button in a panel's header — the diagonal arrows, next to
the × — gives that panel the whole surface. The rest of the matrix keeps
its layout underneath and the panel's slot says *Shown maximized*. The
terminal resizes to the bigger space, so more of it fits. The same button,
now **Restore**, puts it back; from the keyboard that is
<kbd>Shift</kbd>+<kbd>Esc</kbd> to leave the terminal, then <kbd>Esc</kbd>.

## Limits, and why

| | |
|---|---|
| 64 matrices, 500 panels each | a matrix is meant to feel unbounded; the cap keeps one page's load honest |
| smallest panel: 4 columns × 8 rows | anything narrower is too cramped for a terminal to be readable |
| only panels near what you are looking at stay connected | each live panel is a real connection to a real terminal; the rest show their last state in the header and wake up when you scroll to them |

Scrolling past a panel does not connect it — it has to stay in view for a
moment first. That is why a fast scroll through a hundred panels costs
nothing.

## On a phone

The Apps list shows the Matrix tile as **Desktop only**. A grid of live
terminals needs a pointer and a wide screen; on a phone, open the agent or
terminal you need from the sidebar instead.
