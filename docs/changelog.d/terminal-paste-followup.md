### Fixed
- **Terminal paste now answers the same door as the Attach menu.**
  An interactive agent pane whose bound record carries no launch fields
  no longer refuses a files-paste the menu would accept.
- **A double-paste stages both pastes.** Rapid Ctrl+V,V used to keep only
  the second files; concurrent stages now append functionally under the
  4-file cap.
- **Text pasted alongside files seeds the message.** It used to be dropped
  (or land stray in the terminal when clipboard-read was granted); the
  keydown path's late text is now claimed away for that gesture.
