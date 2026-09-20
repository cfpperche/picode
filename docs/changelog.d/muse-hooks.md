### Fixed
- **Activity reports without a native runtime no longer 409.** Session
  reports (which carry a session id) used to require a live native
  runtime, so observers that never start one — the Antigravity title
  reporter today — had every report refused and their terminals read
  Open forever. With no runtime entry there is no identity fence to
  protect, so those reports now land as plain terminal states; terminals
  with a live runtime keep the strict conflict.
- **The server test suite no longer writes into your real home.** Booting
  a server seeds Activity on and syncs integration files, and the two
  CLIs installed through your own settings (Antigravity, Muse) resolved
  the developer's real home in tests that never isolated it. The whole
  suite now runs under a throwaway HOME (plus XDG), guarded by a test
  that fails if the sandbox ever goes missing.

### Changed
- Muse Code lifecycle docs: R3233 grew real hooks, but they cannot feed
  per-terminal activity (see the plan note), so Muse stays honestly Open.
