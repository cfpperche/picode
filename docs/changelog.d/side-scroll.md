### Fixed
- **Sidebar scrollbar alignment and track.** The sidebar's `.side-scroll`
  scroller previously lived inside `.side-section` with an 8px horizontal
  padding, causing the scrollbar to float 8px away from the right border and
  look disconnected when scrolling down past the pinned header.
  The container now spans full width with `padding: 0 8px 10px` on
  `.side-scroll`, placing the scrollbar flush against the sidebar's right
  edge while maintaining consistent 8px breathing room for item rows and headers.
