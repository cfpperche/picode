### Added

- **The key-map API is documented for every agent CLI**: `GET /api/cli-keys?cli=`
  answers one shape — the map's file and catalog, the host's platform, when the
  CLI picks an edit up — and `PUT` writes it. Pi answers today; the other eight
  answer with their state (and a refusal that names why: *"Grok does not allow
  its keys to be remapped"* is not the same fact as *"PiCode cannot write Codex's
  key map yet"*). See it in [the API reference](/api/).
