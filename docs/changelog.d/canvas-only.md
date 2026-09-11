### Changed

- **The Matrix app is now called Canvas.** Same boards, same panels, same
  links; the name says what it is. Old bookmarks keep working —
  `#/app/matrix` and `#/app/matrix/<id>` land on the canvas — and a tab
  you had open reopens as the Canvas tab, with the canvas and the view
  you left it at.
- **Every board is the canvas plane.** The 12-column grid layout is gone,
  so there is one place a panel can be and one way to move it. A board
  still in grid mode was converted once, keeping the arrangement it had —
  a panel that was smaller than the canvas minimum comes out slightly
  larger.
- **The API and the feed were renamed with it**: `/api/matrices…` is now
  `/api/canvases…` and the `matrix.*` events are `canvas.*`. PiCode's own
  clients ship in the same binary and were changed together.
- **PiCode loads lighter for everyone.** Canvas is fetched the first time
  you open it instead of riding in every page load, and the two layout
  libraries the grid needed are gone: 18 KB gzip off the first load of the
  desktop, whether or not you use the app.
