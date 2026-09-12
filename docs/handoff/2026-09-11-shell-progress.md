# 2026-09-11 — feat/shell-progress: live compact progress (Phase 2 session B)

Shipped (fast-forward ready, installed live): the Give back flow streams its
steps into the WSL Disk window. In --json mode the Go tool prints one
progress object per line (progressLine, %q-quoted, pinned by
compact_json_test.go); the shell's disk_compact spawns the tool, reads the
pipe line by line, forwards each step as a `disk-progress` event, and
returns only the final outcome. The window listens, shows the stage in the
state line, locks the buttons (re-entry guarded), and treats a stream that
ends without an outcome as a failure.

Verified: Go tests (progress format incl. quoted steps), cargo xwin build;
installed to %LOCALAPPDATA%\PicodeShell (+ Go tray updated via
desktop-restart) and relaunched, PID confirmed. The full live click-through
(compact kills sessions) is the owner's, with the in-window confirm as the
gate.

visual-review: the window is owner-visual; progress renders in the state
line, outcome in the outcome block.

Debts: no cancel button (safe only before the terminate — noted); no
measurement cache; toast shortcut still pending.

Merge: fast-forward ready.
