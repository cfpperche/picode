### Fixed
- Typing in the annotation card no longer triggers the page's own keyboard
  shortcuts: keystrokes are shielded at the card's shadow root, so a site
  hotkey (GitHub's "s" opened its search) can no longer steal focus and
  swallow the character mid-word.
- Enter saves the note again — it had never worked, for the same reason
  (the event target outside a shadow root is the host, not the input).
