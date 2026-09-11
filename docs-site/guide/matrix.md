# Matrix

A **matrix** is one page that shows many agents and terminals side by side,
live. Each tile is a **panel**: the real screen of that agent or terminal —
the same one its own tab shows — not a picture of it. A panel can also hold
a **pinned note**, a **file**, or the **changes** to a file — so the plan you
are working from and the code you are changing sit beside the work.

Open it from the **Apps** tab → **Matrix**. Matrices are saved on the
server, so every browser signed in to your PiCode sees the same ones.

## Make one

1. **New matrix**, give it a name.
2. **Add panel**, then pick from the list — it is grouped into **Agents**,
   **Terminals**, **Pins**, **Open files** and **Changes to an open file**.
   Anything already on this matrix is not offered again — though one file
   can be on it twice, once as the file and once as its changes.
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
| a pin | the note, formatted and read-only |
| a file | the editor, with **Save** — the same one its tab has |
| the changes to a file | the diff, kept up to date as the file changes |
| a stopped CLI terminal | *This CLI terminal is stopped.* — Resume last session |
| an agent working in its terminal | its live screen |
| a managed agent | its conversation, live and read-only — the same turns, tools and answers its tab shows |
| a stopped agent | *Agent is stopped.* — Run |
| something that was deleted | *That terminal is gone.* — Remove |

**Open** in a panel's header sends you to that agent or terminal's own tab.
The screen follows you there and comes back when you return to the matrix —
there is only ever one of it. On a note the same button opens **Pin
Studio**, which is where a note is written; the panel picks the change up
as soon as you save it.

### A managed agent's conversation

A panel bound to a managed agent shows what it is saying, as it says it —
the same turns, tool cards and answers you would read in its tab, scrolling
itself as the agent writes. You read here; you **answer in the tab**. The
panel has no message box, no queue buttons and no way to reply to a
question, so nothing you click by accident can reach the agent.

**When the agent is waiting on you**, the chip turns to **Needs you** and
the panel shows the question with the choices it is offering, above a line
across the bottom of the body that names it and gives you **Open**. Open
takes you to the agent's tab, where you answer. The chip is the same one the
sidebar shows, so a panel says *Needs you* even when its body is asleep or
the matrix is zoomed too far out to read — which is the point: you can see
which of twenty agents is blocked from across the whole board.

A conversation costs a live connection, so the matrix only keeps the ones
you can actually read: panels far from what you are looking at are put to
sleep, panels below 40 % zoom become name-plates, and if more than twelve
conversations are in view at once the ones you scrolled past longest ago say
*Paused* until you come back to them. Nothing is lost — the conversation is
read again from the start when the panel wakes up.

Notes, files and diffs are not terminals, so the rules that exist for
terminals do not apply to them: they take your clicks and your scrolling at any zoom, and
stay readable down to 40 %, where — like everything else — they become
name-plates.

**A file or diff panel only shows a file you already have open in a tab.**
The picker lists those and nothing else: it is not a file browser, and it
never goes looking through your folders. Open the file the way you always
do, then add it here — as the file, as its changes, or both. **Unsaved changes are safe**: the panel says *Unsaved*, it is
never put to sleep while it holds them, and maximizing it or switching the
matrix to the other layout keeps your text. Removing the panel asks first.

## Link two panels

A **link** between two panels is you saying: *these two sessions may send
each other messages.* Drawing one is the whole gesture — hover a panel on
the canvas, and a small round connector appears at the right of its header;
drag it onto another panel.

**What a link does, exactly.** The two sessions gain each other as a
**contact** in PiCode's messaging (the same thing the **Messages** page sets
up for a folder). One can send the other a message and read the replies,
and that is all.

**What a link never does.** It does not let one session read the other's
history — not its transcript, not its scrollback, not its session file, not
what you typed into it. Those often hold file contents, command output and
credentials, and reading is copying: once a session has copied another
one's log, no amount of removing the line could take it back. If one agent
needs to know what another found, it asks — and the other answers in its own
words, with its own judgment about what to share.

**PiCode will always ask first.** Drawing a line never connects anything
quietly:

- If one of the two is not connected yet, you are offered the connection in
  one line, with what it grants. Cancel and nothing at all is written — no
  link, no connection.
- If the two work in **different project folders**, you are asked again,
  separately, and the question names both folders. Two sessions in the same
  folder can already message each other with no link at all; a link across
  folders is the one thing that is genuinely new, so it gets its own yes.

**Removing a link takes the permission with it.** Click the small chip on
the line (or select the line and press <kbd>Delete</kbd>) and confirm. There
is nothing left over: PiCode works out who may message whom from the lines
that exist right now, so the moment a line is gone, so is the permission.
Deleting either panel, or the whole matrix, does the same.

**A link that is not working says so.** It turns amber, is marked **Broken**
and tells you why when you point at it — *"Cleo's connection was revoked;
the link grants nothing"*, *"Delta's session changed; the link grants
nothing"*. A broken link grants nothing at all; it is not a faded version of
a working one.

Only **agents** and **Agent CLI terminals** can be linked. A note, a file or
a diff has no mailbox, so those panels have no connector.

In **grid** mode there is no plane to draw on, so each panel's header shows
a small chip with how many links it has — amber if any of them is broken.

### Read every link in one place

**Agent CLIs → Messages** ends with **Matrix links**: every link you have
drawn, anywhere, with both ends, which matrix it lives on, whether it is
working right now, and a **Remove** that revokes it exactly as the canvas
does. It is not filtered by the folder picker above it — links across two
folders are precisely the ones you want to see — so this page is the
complete answer to *"which of my sessions can reach which?"*

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
| 1000 links per matrix | a link is a permission you drew by hand; a thousand of them is already more than anyone can audit |
| only panels near what you are looking at stay connected | each live panel is a real connection to a real terminal; the rest show their last state in the header and wake up when you reach them |

Scrolling or panning past a panel does not connect it — it has to stay in
view for a moment first. That is why a fast trip across a hundred panels
costs nothing.

## On a phone

The Apps list shows the Matrix tile as **Desktop only**. A grid of live
terminals needs a pointer and a wide screen; on a phone, open the agent or
terminal you need from the sidebar instead.
