### Fixed
- Annotation screenshots actually arrive: the crop asked the shell for the
  page preview with the React tab id, which the shell could not resolve, so
  every Send staged the note alone and the picture failed in silence. The
  shell now resolves either id shape and the call site passes the native one.

### Added
- `find_webview`: one resolver for "the webview of this tab", used by the
  preview, the annotate mode, the trash and the state pull.
