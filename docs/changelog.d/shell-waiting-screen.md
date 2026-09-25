### Fixed

- **Desktop: the window no longer gets stuck on "PiCode is not running yet".**
  The desktop window now opens right away on a waiting screen that says what
  is happening: Linux starting, PiCode starting or restarting, PiCode not
  answering, or a certificate Windows doesn't trust yet. Each stage has one
  action (Try again, Start PiCode, View logs, Trust certificate) and a
  running timer while something is in progress, and the window switches to
  PiCode by itself the moment it answers. Before, the page promised a retry
  it never made, and the window could take up to 30 seconds to appear. The
  screen has its own title bar, so it can be moved, minimized and closed.
