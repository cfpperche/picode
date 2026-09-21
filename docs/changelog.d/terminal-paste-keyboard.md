### Fixed
- **Ctrl+V / Ctrl+Shift+V now attaches images in terminal panes.**
  The keydown used to kill the browser's own paste (stopping xterm's ^V)
  and then read text only, so an image clipboard pasted nothing at all.
  It now re-reads the clipboard (files included) and fires an equivalent
  paste, routing through the same attach bar the native menu already
  opened. Text-only pastes behave exactly as before.
- **The PiCode menu's Paste row stages files.** It read text only and
  dropped images silently; with files on the clipboard it now opens the
  attach bar, like the keyboard does.
