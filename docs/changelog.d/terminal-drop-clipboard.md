### Added
- **Paste files copied in Explorer straight into an agent terminal.**
  Screenshots already pasted as bytes, but Explorer file copies arrive as
  references the browser never exposes — the attach never opened. In the
  desktop app an empty paste now asks the Windows-native shell, which
  reads the clipboard files and stages them through the same drop door
  (4 files, 4 MB each). Real browsers are untouched.
