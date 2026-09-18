### Added
- **Annotate mode matches the reference.** The strip replaces the URL bar
  while annotating (exit, discard-all, undo, crops toggle, hints, Send N);
  notes accumulate as numbered pins, each edited in an anchored card
  (Cancel/Save) and collapsed to a chip (text, options, remove); Send ships
  the whole set as one context to the agent terminal.

### Fixed
- Annotate mode no longer dies on the ACL (the command was missing from the
  shell manifest) or on Send (the page URL ref did not exist).
- The fallback preview no longer blanks the app when a second note is
  pinned (the event was read after React released it).
