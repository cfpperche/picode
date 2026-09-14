### Added
- **The dashboard now says what is waiting on you.** One line above the
  numbers — “3 need you · 2 questions in Inbox · 1 terminal waiting” — with a
  single action that goes to the Inbox, or to the waiting terminal when the
  Inbox is empty. It appears only while something is blocked.

### Fixed
- **Quota readings no longer outlive their window.** `Limits` now marks a
  reading whose window has already reset as `stale · reset 2d ago` instead of
  counting down to a past instant, and every row carries how old the reading
  is (hover). A panel fed by one CLI also says so: `Covers Codex only.`
