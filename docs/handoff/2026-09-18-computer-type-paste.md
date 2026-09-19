# 2026-09-18 — feat/computer-type-paste

`type` gains `mode: "paste"`: the text goes through the clipboard and one
Ctrl+V (`paste_text` in `desktop-shell/src/computer.rs`, cap 100 000
characters), and the clipboard's previous text is written back. Default
stays `keys`. Reason, measured 2026-09-18: Windows 11 Notepad (WinUI)
garbles synthetic keystrokes whenever it falls behind, while Windows
Terminal takes them whole; a paste lands whole everywhere it is allowed.
Schema and guideline updated in `packages/pi-computer` and
`internal/mcptool` (the test lists `mode`); guide row added; changelog.

First attempt of this branch was lost: the commit chain stopped on a grep
with no match and the worktree was removed with the edits uncommitted.
This one was verified with `git show --stat` before landing.

Proof: `go test ./internal/mcptool`, pi-computer tests, cross `cargo check`,
`make desktop-shell`. Live after `make desktop-restart`: the long sentence
with `mode: "paste"` into Notepad under load, read back with `snapshot`.

## Next up
- Owner: `make desktop-restart`, then paste into Notepad from the Claude Code terminal.

## Debts
- Non-text clipboard content (image, files) is replaced by the paste and not restored.
