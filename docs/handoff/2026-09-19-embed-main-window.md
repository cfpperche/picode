# 2026-09-19 — feat/embed-main-window

The embed spike's first morning: `computer_embed` answered "no main
window" on the resident the logon task starts with `--hidden`, where the
Tauri registry lookup returns None (the note on `MAIN_WINDOW` in
`main.rs`). `main_window()` now hands modules the kept handle, registry as
fallback; `embed.rs` uses it. Cross `cargo check` clean, exe rebuilt.

## Next up
- Owner: `make desktop-restart`, then the four-app embed matrix.
