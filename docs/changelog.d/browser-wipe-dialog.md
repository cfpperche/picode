### Changed
- **Clear browsing data** now opens a dialog in the reference shape: a
  time-range picker (hour / 24 hours / 7 days / 4 weeks / all time) over
  a checklist of what to clear — browsing history, cookies and site
  data, cached images and files, download history, autofill form data
  and site settings — each showing what it will remove.
### Fixed
- Clearing site data no longer touches saved passwords or autofill; the
  earlier one-click version included general autofill in its mask and
  missed WebView2's own browsing history.
