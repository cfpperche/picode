### Fixed
- **The Antigravity sign-in works on macOS.** It runs under a pty through
  `script(1)`, and the flags it used are util-linux's — macOS's BSD script
  answers them with its own usage line, so the sign-in answered 502 there. The
  invocation is platform-aware now.
- **A home behind a symlink shows as `~/…` again, and Grok's trusted folder is
  found.** On macOS `/var` points at `/private/var`, so the paths the
  instructions report (resolved) and the home they were compared against (as
  typed) disagreed; the abbreviation missed and a trusted folder read as
  untrusted. Both sides are resolved now.
