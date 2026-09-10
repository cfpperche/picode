### Fixed

- **Reloading no longer halves fullscreen mode.** A reload always brought the mode's layout back but not the browser part — no gesture, so no fullscreen and no keyboard lock — and you had to toggle it off and on to get the keys back. Now the first click or keypress after the reload completes it on its own: the browser goes fullscreen again, the terminal keeps every key, nothing else changes. If the first gesture is Escape or the fullscreen chord, it does what it means (leaves the mode); synthetic input never triggers it.
