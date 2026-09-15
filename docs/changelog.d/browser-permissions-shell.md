### Added
- **Site permissions, the shell half** (Settings ▸ Browser): every tab now
  answers permission requests — camera, microphone, location, notifications,
  clipboard, autoplay, sensors, MIDI, fonts, file system — from the policy
  the user set (`btab_set_permission_policy`), and reports each outcome as
  `btat://permission`. The app records the standing through the daemon's
  API, so the Site settings dialog has real data to list and edit.
