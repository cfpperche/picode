### Changed
- **Browsing history** now presents the data the way the reference does,
  inside the dialog: a search field, day groups that collapse (newest
  open, "Today"/"Yesterday" named), and a row per visit with its site
  icon, host, visit time and a per-row menu (Copy link, Remove from
  history). Rows can be selected and removed in one go, and "Clear
  browsing data" sits on the section header.
### Added
- `POST /api/browser/history/delete` removes a selection of visits in
  one transaction.
