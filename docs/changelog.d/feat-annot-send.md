### Fixed
- Annotations reach the strip again: the page posts message objects (the
  host serializes once) and the chrome unwraps at any encoding depth, so a
  saved note always shows up as Send N instead of a silent Send 0.
