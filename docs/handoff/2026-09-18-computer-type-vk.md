# 2026-09-18 — feat/computer-type-vk

Second round on `type` (after `feat/computer-type-pacing`): with the
machine loaded by other sessions, 5 ms pacing still produced runs of one
repeated character in Windows 11 Notepad ("do PiCode" → "dddddddde"):
whenever the app falls behind, every queued `VK_PACKET` translates to the
newest packet's character, so no fixed pace is safe.

Fix in `desktop-shell/src/input.rs`: `layout_key` — a character the
current layout produces with at most Shift is typed as its virtual key
(Shift down, key down/up, Shift up), which carries its character in the
message. Unicode packets remain for characters the layout lacks, AltGr
characters and dead keys (`MAPVK_VK_TO_CHAR` high bit). Pace and cap unchanged.

Proof: cross `cargo check` clean; `make desktop-shell` builds. Live, after
`make desktop-restart`, under load: the sentence lands intact in Windows
Terminal (cmd), but Windows 11 Notepad (WinUI) still reads shifted keys
unshifted and drops some — that app, not the injection. Layout: ENG.

## Next up
- Decide on `mode: "paste"` for WinUI receivers (Notepad, Settings).

## Debts
- WinUI apps garble synthetic keystrokes when behind: offer `mode: "paste"` (clipboard + Ctrl+V, clipboard restored).
- Keys resolve against the shell's own layout; an app on another layout gets that mapping.
