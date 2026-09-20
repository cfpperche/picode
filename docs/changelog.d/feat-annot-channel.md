### Fixed
- Annotation Send lights up again: the page's message channel is
  re-subscribed on every arm (a webview recreated under the same tab id
  used to inherit a stale subscription and go silent), the receiver is
  dropped when its tab closes, and a page that does not answer is retried
  once and then named out loud instead of leaving a dead Send.
- Re-arming annotate mode keeps the pins on the page and re-reports them.
