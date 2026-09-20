# Embedding Windows apps in a PiCode pane — spike (2026-09-18/19)

Question (owner, 2026-09-18): can Windows applications open inside PiCode's
desktop window, in a pane like the work browser? Three routes exist: a DWM
thumbnail mirror (view only), cross-process `SetParent` reparenting (native
in-pane experience, unsupported by Microsoft), and Windows Graphics Capture
with forwarded input (universal, remote-desktop feel). The owner accepted a
one-day spike on reparenting, the only route that could give a native pane.
The spike is `desktop-shell/src/embed.rs` (`computer_embed`/`computer_unembed`,
reachable from the Computer lab only; 6fb3f90b, 156462d9): strip the caption
and frame styles, set `WS_CHILD`, `SetParent` under the main window with
mixed DPI hosting, place in the right half; restore on unembed.

## Matrix

Measured on the owner's desktop (Windows 11 26200, 2560×1600 at 150 %), the
owner driving the lab, screenshots read back by a subagent.

| App | Renders inside | Keyboard after a click | Dialogs / menus | Follows a resize of PiCode | Unembed |
|---|---|---|---|---|---|
| Windows 11 Notepad (WinUI 3) | **No** — tab strip, toolbar and status bar paint; the editor stays unpainted, PiCode shows through | — | — | — | back on the desktop, intact |
| File Explorer | **No** — tab strip, address bar and command bar paint (semi-transparent); no navigation pane, no file list | — | — | — | — |
| Windows Terminal (XAML island in a Win32 window) | **Yes**, complete | yes (`dir` ran) | Settings opened as a tab inside the app | no: the rectangle is set once; the app's own maximize button still works and covered PiCode | back on the desktop, intact, own taskbar entry |
| VS Code (Electron) | **Yes**, complete (activity bar, tree, editor, status bar) | not measured | — | no: fixed rectangle, smaller than the half | — |

## Reading

`SetParent` works for apps whose content is drawn into an ordinary Win32
window: Windows Terminal, Electron/Chromium, classic Win32. It fails for
Microsoft's own modern apps (WinUI 3 Notepad, Explorer's body), whose
content is composed per process and does not follow a change of parent: the
chrome paints, the body does not. Microsoft is moving its apps to that
stack. Add the known costs — unsupported cross-process parenting, attached
input queues, the app's own caption buttons staying live, the child dying
with the parent — and reparenting is not a general mechanism. At most it is
a named feature for Terminal and VS Code.

Spike defects, ours and fixable, not part of the verdict: the rectangle was
computed on a thread whose DPI context differs from the window's (stopped
230–550 px short of the right edge); a real pane recomputes it in the
parent's DPI on every `WM_SIZE`, and hides or intercepts the app's caption
buttons. The Tauri registry does not see the main window on a resident the
logon task starts with `--hidden`; the kept handle does (156462d9).

## Recommendation

Do not build an app pane now. If the goal is to watch the agent work without
leaving PiCode, the DWM thumbnail mirror is cheap, read-only and universal,
with the `computer` tool acting on the real window. If interaction inside
the pane is wanted later, Windows Graphics Capture plus forwarded input is
the evolution and covers Notepad and Explorer too. Reparenting stays a
possible named feature for Terminal and VS Code, behind a compatibility
list, if demand appears. The lab buttons stay as a lab tool.
