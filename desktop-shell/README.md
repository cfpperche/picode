# picode-shell — Desktop v2 (ADR-0120)

The Windows work-application window: Tauri 2 + WebView2 rendering the PiCode
UI served by the daemon in WSL. Phase 1 skeleton (docs/plans/desktop-v2.md).

## Build (cross from WSL)

```bash
rustup target add x86_64-pc-windows-msvc
cargo install cargo-xwin        # once
make desktop-shell              # from the repo root; produces:
# desktop-shell/target/x86_64-pc-windows-msvc/release/picode-shell.exe
```

Run the exe on Windows. It discovers the daemon address through `wsl.exe`
(`<data>/server.json`), falls back to a local offline page when nothing
answers, lives in the tray (left-click opens the window), and stays
single-instance.

## Coexistence

The Go tray (`picode-desktop.exe`, ADR-0020) keeps owning keepalive and the
disk actions during Phase 1 — both can run; quit the old tray while testing
this one. Phase 2 migrates the duties (docs/plans/desktop-v2.md).
