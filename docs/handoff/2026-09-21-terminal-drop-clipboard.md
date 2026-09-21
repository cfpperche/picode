# 2026-09-21 — feat/terminal-drop-clipboard: paste Explorer file copies via the shell
Shipped: `clipboard_files` Tauri command (CF_HDROP → bytes, 4 files /
4 MB, permission + capability + regenerated schemas); `shellClipboard.js`
bridge (self-gating, unit-tested); empty pastes in the shell ask the
native clipboard in TermSurface capture, menu Paste, and the keydown
re-dispatch (now always fires so keyboard reaches the shell path).
Server untouched (it runs in WSL2 and cannot see the Windows clipboard).
Verified: `make ci-scoped` PASS; node incl. 5 bridge tests; `make web`
ok; scratch live with a stubbed invoke: menu-Paste stages 3 files in the
bar (screenshot read), keydown reaches the shell attempt, text still
pastes exactly once, overlayAudit ok.
Blind spots: Rust compiles only on Windows (no toolchain here) —
signatures checked against the registry, rustfmt clean; real Explorer
copies need the owner's desktop run.
visual-review: PASS (drop-clip-3files read; card 5/5; audit ok)
Not done: Linux/macOS clipboard files (endpoint answers empty there by
construction — shell is Windows-only).
Merge: fast-forward ready. Deploy: owner's order (not run).
