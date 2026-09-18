# pi-computer

The `computer` tool for pi (ADR-0148): an agent uses the Windows desktop
through PiCode's desktop app — screenshots of a monitor or a window, the
accessibility tree of a window, mouse, keyboard, clipboard, opening programs.

| Action | Answer |
|---|---|
| `screenshot`, `zoom`, `wait` | an image the model sees; every coordinate it sends afterwards is a pixel of that image |
| `snapshot` | the window's accessibility tree, one line per element with its centre in the last image |
| `windows`, `focus` | what is open, and bringing one window forward |
| clicks, `left_click_drag`, `mouse_move`, `scroll`, `type`, `key`, `hold_key` | the action, then a fresh image |
| `clipboard_read`, `clipboard_write`, `open` | one line |

One grant is the whole policy: the human switches it on per agent or per
terminal in Settings ▸ Computer. With it, the agent acts with the human's
own permissions on the human's desktop — no sandbox, no tiers. Without it
(or for a `pi` started outside PiCode, which has no identity here), every
call is refused with the path to the switch.

Install it like any pi package (it finds PiCode through `server.json` and
the install token, exactly as `pi-browser` does). Nothing here runs without
the desktop app: with no shell connected the daemon answers that the desktop
app is not connected, and the tool says so.
