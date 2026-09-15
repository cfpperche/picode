### Fixed
- **Muse Code and Antigravity terminals wear their own mark in the sidebar,
  the tabs and the phone.** A terminal launched with a CLI that reports no
  activity (both of them today) fell back to the plain-shell icon, the
  *Shell session* subtitle and the *Terminal open* status. Every surface now
  resolves the identity from the runtime CLI, otherwise from the CLI the
  terminal was launched with, so the row reads **Antigravity · Open** or
  **Muse Code · Open** like any other CLI. It also lands in the dashboard's
  CLI bucket instead of counting as a plain shell.
- **A terminal created while the page is open stops looking like a shell.**
  The creation only announced the bare record, so a new Muse Code or
  Antigravity terminal kept the default icon until a reload; the launch view
  now travels on the feed with it.
