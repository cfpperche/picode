### Fixed
- **`computer` typing came out as one repeated character.** The desktop
  app fired every Unicode key event back to back, and Windows 11 Notepad
  read each one as the last character of the text. The app now paces
  characters 5 ms apart and types at most 2 000 characters per call, so
  `type` writes what the agent sent.
