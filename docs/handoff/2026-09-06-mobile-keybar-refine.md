# 2026-09-06 — mobile extra keys, owner screenshots after deploy

Branch: `feat/mobile-keybar-refine`.

Owner: black strip with the keyboard closed; Safari undo/Done pill;
prompt covered; TUI ink visible through the extra-keys row.

Fixes: pin `#m-app` only while the IME covers pixels (`inset: 0` at
rest); opaque extra-keys row above xterm; clip + refit the pane.
Safari's undo/Done pill is system chrome — a PWA cannot hide it.

visual-review: UNVERIFIED on a real iPhone IME (same constraint as the
first landing).
