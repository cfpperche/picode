### Fixed
- Connector gallery refreshes no longer race over the same cache file: two
  at once (the background timer and a manual refresh) could make one fail
  with "no such file or directory" and the gallery answer an error. The
  cache write is serialized and uses a private temp file.
