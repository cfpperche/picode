# Desktop v2 shell

## Next

- Policy UI for the desktop-v2 gates (ADR-0120). The daemon endpoint and the navigation gate are tracked in `work-browser-tabs.md`.

## Debts

- Named-window OAuth popups (`window.open(url, 'picode-mcp-auth')`) still use the native path in the shell's main window — needs its own auth-flow verification before routing out (the `_blank` bridge covers the rest since 2026-09-15).
