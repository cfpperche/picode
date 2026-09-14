### Added

- **Archiving and deleting snippets now work on the phone.** A snippet's page
  has **Archive** / **Unarchive** and **Delete** (with a confirm naming the
  snippet), and the list has an **Archived** view next to **Active** — so an
  archived snippet is visible and can come back instead of disappearing with no
  handle on the other side.

### Changed

- **A snippet Command no longer runs into a repository another agent is
  writing in.** The confirm step still shows the exact command; running it is
  refused with a message naming who is at work there, the same rule the
  terminal's **Run command…** already followed.

### Fixed

- Destructive buttons on the phone (Delete) read destructive **before** the
  tap. They only had a hover style, and a phone has no hover.
