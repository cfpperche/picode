### Added
- **Site permissions, the data half** (Settings ▸ Browser): the daemon now
  keeps one standing per site and kind — camera, microphone, location,
  notifications, clipboard, autoplay, sensors, MIDI, fonts, file system —
  with `GET/POST /api/browser/permissions`, `DELETE
  /api/browser/permissions/{id}` and `POST /api/browser/permissions/clear`
  (optionally one kind), all tested, on migration 050 and the events
  invariant. The shell feeds it and the Site settings dialog reads it in the
  next slice.
