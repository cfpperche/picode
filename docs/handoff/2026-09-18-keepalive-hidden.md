# 2026-09-18 — keepalive-hidden: the keepalive task shows no console window

Follow-up to distro-keepalive (ADR-0155), same day: the owner found a
wsl.exe console window on the desktop — a scheduled task whose action is
a plain console app allocates a visible window in the user's session.
The shell's own child never showed one (CREATE_NO_WINDOW); the task did.
S4U (non-interactive principal) would be the config-only fix but needs
admin to register, and the task must stay creatable non-elevated by the
shell and desktop-swap.sh — refused. Fix: the task action wraps wsl.exe
in `conhost.exe --headless`. Verified live before shipping: task
Running, all wsl.exe MainWindowHandle 0, sleep infinity alive in the
distro, the old visible instance ended with Stop-ScheduledTask.

## Debts

- docs/handoff/open/wsl-keepalive.md unchanged: sparseVhd attribution and
  Windows-side test coverage are still open.
