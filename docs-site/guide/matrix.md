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

A matrix is laid out in one of two ways, and the **Grid | Canvas** switch
in the header says which. Every matrix starts as a grid.

- **Move** a panel by dragging its header.
- **Resize** it from any corner or edge.

In **grid** mode panels sit in twelve columns and settle upwards into the
free space above them, so the page has no gaps. In **canvas** mode they sit
wherever you put them, on a plane you pan and zoom.

The layout saves itself as you go, for everyone. If someone else changed
the same matrix while you were dragging, PiCode reloads it and says
*Matrix changed elsewhere — reloaded.*

## The canvas

Press **Canvas** in the header and the whole matrix moves onto one plane.
It is the same panels — same header, same status, same buttons — with the
twelve columns taken away.

| To | Do |
|---|---|
| move around | drag with the middle or right mouse button, or hold <kbd>Space</kbd> and drag |
| zoom | the mouse wheel, or the **−** and **+** buttons beside the minimap |
| see everything at once | **Fit**, or press <kbd>0</kbd> |
| move several panels together | drag a box around them, then drag any one of them |
| find your way | the minimap in the corner shows the whole plane and where you are on it |
| lay them out again | **Tidy panels** in the **⋯** menu |

**Tidy panels** puts them back in reading order — three across, left to
right — and keeps every panel the size you made it. Nothing on a canvas
moves on its own: that is what the plane is for, so tidying is something
you ask for rather than something that happens to you.

Going back to **Grid** packs the panels into twelve columns again. It asks
first, because where they sat on the plane is not kept.

### Where you are looking is yours

The place and the zoom you left the canvas at are remembered in **this
browser only**, and come back the next time you open that matrix. Two
people — or the same person on a laptop and a desktop — can look at
different corners of the same matrix without pulling each other around.
Where the panels *are* is shared, as it always was: moving a panel is an
edit, moving your view is not.

### Zoom, and why typing needs 100 %

How far out you are decides what a panel can do.

| Zoom | The panel |
|---|---|
| **100 %** | a live terminal you can read, type in **and click in** |
| 80 – 150 % | still live, still takes your typing, but clicks are switched off |
| below 75 % | the text of the last screen it had, frozen, with the header still live |
| below 40 % | a name-plate: face, name and status, sized to stay readable |

The one rule to know: **a click inside a terminal only lands where you
aimed it at 100 %**. Away from 100 % the terminal works out which
character you clicked from the wrong measurements and picks the wrong one —
and PiCode passes clicks through to the program running inside, so a
mis-aimed click reaches your editor or your CLI, not just the browser.
Nothing on screen would look wrong, so the panel simply stops taking
clicks: clicking it takes you back to 100 % first, and the click after that
lands where you meant. The percentage beside the minimap does the same in
one press.

Below 75 % a panel disconnects and shows its last screen instead. That is
what lets a matrix hold hundreds of panels: only the ones you are close
enough to read are connected. Zoom back in and the same terminals come
back exactly where they were.

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

On a canvas the same keys work, and three more join them once a panel is
selected:

| Key | Does |
|---|---|
| <kbd>+</kbd> / <kbd>-</kbd> | zoom in / out |
| <kbd>0</kbd> | fit every panel on screen |
| <kbd>Enter</kbd> | go to 100 % and start typing, in one press |

Arrow keys bring an off-screen panel into view; a panel already on screen
is left where it is, so the plane never jumps under you.

Inside a panel every other key belongs to the program running there — a
plain <kbd>Esc</kbd> reaches it, which is why leaving takes
<kbd>Shift</kbd>+<kbd>Esc</kbd>.

## One panel, full width

The **Maximize** button in a panel's header — the diagonal arrows, next to
the × — gives that panel the whole surface, in either mode. The rest of the
matrix keeps its layout underneath and the panel's slot says *Shown
maximized*. The terminal resizes to the bigger space, so more of it fits.
The same button, now **Restore**, puts it back; from the keyboard that is
<kbd>Shift</kbd>+<kbd>Esc</kbd> to leave the terminal, then <kbd>Esc</kbd>.

## Limits, and why

| | |
|---|---|
| 64 matrices, 500 panels each | a matrix is meant to feel unbounded; the cap keeps one page's load honest |
| smallest panel: 4 columns × 8 rows in a grid, 256 × 224 pixels on a canvas | anything narrower is too cramped for a terminal to be readable |
| zoom from 20 % to 150 % | further out than 20 % nothing resolves, even a name; further in the panel is bigger than the screen |
| only panels near what you are looking at stay connected | each live panel is a real connection to a real terminal; the rest show their last state in the header and wake up when you reach them |

Scrolling or panning past a panel does not connect it — it has to stay in
view for a moment first. That is why a fast trip across a hundred panels
costs nothing.

## On a phone

The Apps list shows the Matrix tile as **Desktop only**. A grid of live
terminals needs a pointer and a wide screen; on a phone, open the agent or
terminal you need from the sidebar instead.
