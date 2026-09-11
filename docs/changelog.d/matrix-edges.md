### Added

- **Matrix edges — the model, the API and the contact union (ADR-0116).**
  Two panels of a matrix can be linked, and the link grants exactly one
  thing: the two sessions may exchange messages through the existing
  mailbox (ADR-0104). `GET/POST /api/matrices/{id}/edges` and
  `DELETE /api/matrices/{id}/edges/{edgeId}`; `GET /api/matrices/{id}` now
  carries `edges` as well as `panels`, so one read still opens a matrix.
  Contacts are now the union of *the same workspace* (unchanged) and
  *linked by a live edge*, which is how two sessions in **different project
  folders** can reach each other — per pair, drawn by hand, with no
  owner-wide rule change.
- **An edge never grants a transcript.** It is a mailbox contact and
  nothing else: no session file, no scrollback, no history. The grant is
  derived from the live edge on every read, so removing the edge, either
  panel, or the matrix removes it immediately — nothing is cached. Only the
  owner creates an edge, in the browser; no agent-facing tool can create
  one or discover that one could exist.

### Not yet

- **Nothing is drawn.** This release is the model, the routes and the
  grant. The canvas does not render edges, there is no way to draw or
  remove one from the UI, the enrolment prompt for an unenrolled or
  cross-folder pair is not built, and the Messages view has no audit list
  of live edges. All of that is the next session.
