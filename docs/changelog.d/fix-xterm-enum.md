### Fixed

- web: a TUI booting inside a terminal (OpenCode boots straight into it) could
  freeze the pane at its first painted line until a reload. esbuild's minifier
  dropped the `let` declaration of xterm 6's `requestMode` enum while keeping
  the `(n = {})` assignment, so the first DECRQM query threw
  `ReferenceError: n is not defined` inside the parser and stalled xterm's
  write pipeline. The desktop and mobile builds now pin `@xterm/xterm` to its
  UMD build, which ships the enum pre-compiled.
