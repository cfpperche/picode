### Fixed
- **`computer` typing under load.** Characters the keyboard layout has are
  now typed as keystrokes (with Shift when needed), which carry their own
  character and survive a busy app; only characters the layout lacks, AltGr
  characters and dead keys still travel as Unicode packets. Pacing alone
  (the earlier fix) still garbled text when the app fell behind.
