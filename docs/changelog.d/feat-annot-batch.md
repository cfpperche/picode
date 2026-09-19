### Added
- Sending annotations has no limit on how many: the whole set travels as one
  note (every pin's sentence, element, HTML, styles and screenshot) plus one
  crop per pin, and the crops that do not fit the paste are staged beside the
  note, which names every one of them.

### Fixed
- A Send with three or more annotations no longer fails: it used to hit the
  prompt door's four-file limit and deliver nothing while keeping the notes in
  the store. A Send that cannot be staged whole now stages nothing at all
  instead of half a package.
