import PanelFace from "./PanelFace.jsx";

// The two bodies a canvas panel has instead of a live pane
// (docs/plans/matrix-canvas.md §4.3). Both are inert: no xterm, no socket,
// no pointer — the whole saving of a big plane is that a panel you are not
// reading costs nothing (C0: nine live `top` panes 4–14 % of a core, the
// same page at zoom 0.2 with 500 stills 0.0 %).

// PanelStill — the terminal's last screen as text, captured from
// `term.buffer.active` before the body flipped (stills.js). It is a
// picture of a moment, so it says so: the header carries its age when the
// feed has seen the panel move since, and the text never claims to be live.
// A panel that was never live down here has nothing to show yet — one
// muted line, the same the unloaded body shows.
export function PanelStill({ model, still }) {
  // Nothing captured yet: this panel has never been live in this tab (a
  // camera restored below the band, a page opened straight into one). Say so
  // and name the way out — clicking the body is exactly that.
  if (!still || !still.text) {
    return (
      <div className="cv-placeholder">
        <span>{model.label} — zoom in to read it.</span>
      </div>
    );
  }
  return (
    <pre className="cv-still" aria-label={model.name + " — last screen"}>{still.text}</pre>
  );
}

// PanelPlate — the body below zoom 0.4, where C0 measured that a text
// still never resolves into characters however far you magnify it: the
// face, the name and the status colour, sized to the panel. It is for
// finding a panel on the plane, not for reading it, so everything scales
// by 1 / zoom (--cv-zoom, set once on the canvas root) and stays about the
// same size on screen however far out the viewer is.
export function PanelPlate({ model }) {
  return (
    <div className="cv-plate" role="group" aria-label={model.name + " — " + model.label}>
      <span className="cv-plate-face" aria-hidden="true"><PanelFace model={model} /></span>
      <span className="cv-plate-name">{model.name}</span>
      <span className={"cv-plate-status ws-status is-" + model.status}>{model.label}</span>
    </div>
  );
}
