# Desktop v2 shell

## Next

- Policy UI for the desktop-v2 gates (ADR-0120). The daemon endpoint and the navigation gate are tracked in `work-browser-tabs.md`.

## Debts

- External links are dead in the shell's main window: no `on_new_window` handler and no opener plugin, so every `target="_blank"` click (Setup guide buttons, changelog/GitHub links, OAuth popups) is dropped. Owner deferred the fix 2026-09-15; candidates are a Rust `on_new_window` routing http(s) out (fixes all links) or a Tauri-only web click interceptor invoking `btab_open_external`.
