import { forwardRef, memo, useCallback, useEffect, useRef } from "react";
import PiSpinner from "../PiSpinner.jsx";
import { IconCollapse, IconExpand, IconExternal, IconX } from "../Icons.jsx";
import PanelBody, { hasPane } from "./PanelBody.jsx";
import PanelFace from "./PanelFace.jsx";
import { PanelPlate, PanelStill } from "./PanelStill.jsx";

// Panel — one grid item, always rendered (react-grid-layout needs every
// child to resolve collisions), header live, body only when chunk loading
// says so (docs/plans/matrix-app.md §4.3–§4.5). The header is the drag
// handle (`.mx-head`); its buttons are the cancel zone (`.mx-actions`).
// The wrapper receives react-grid-layout's ref, className, style and the
// drag listeners and spreads them on its root; `children` are
// react-resizable's handles and must render inside the root too. The
// wrapper observes its own visibility through the surface's loader.
//
// Keyboard (plan §4.6 "Focus", docs/architecture/matrix.md): the wrapper
// is the roving tab stop of its matrix — tabIndex 0 on the focused panel
// (or the first one, until something is focused), -1 on the rest — and
// hands every key it gets to the surface (handlers.onKey), which owns the
// model: arrows move between panels, Enter enters the terminal, Shift+Esc
// leaves it, Delete removes. Clicking the chrome focuses the wrapper (a
// focusable div takes the click's focus); clicking the body focuses the
// xterm, which the surface records as "engaged".
//
// Two layers on purpose: the grid re-renders every wrapper on a drag (its
// clone carries a fresh style object), so the outer memo cannot hold —
// the inner one, keyed on the domain props only, does. An xterm body never
// remounts because the surface re-rendered.
//
// Canvas mode (docs/plans/matrix-canvas.md §4.3/§4.4) reuses this exact
// wrapper as a React Flow node, so a panel behaves the same in both hosts.
// It adds three things and changes nothing else: `bodyKind` (live · still ·
// plate — what the zoom says the body is), `note` (the still's age in the
// header) and the library's gates — `.mx-head` drags the node, `.mx-actions`
// and `.mx-panel-body` carry `nodrag`, the body also carries `nowheel` so
// the wheel scrolls the terminal instead of zooming the plane. The classes
// are inert in grid mode, which drags by the same header.

// PanelHead — face, name, hint, the sidebar's chip, Open · Maximize ·
// Remove. The wrapper's header is the drag handle; the maximize layer
// renders the same head fixed.
// What the Open action says per kind: the header's one external-link button
// takes the panel where it can be worked on, and it has to name that place —
// a note's is Pin Studio, not "its tab".
const OPEN_LABEL = { note: "Open in Pin Studio", file: "Open in its own tab", diff: "View this diff in a tab" };
const GONE = ["terminal-gone", "agent-gone", "note-gone", "file-gone", "diff-gone"];

export function PanelHead({ model, loaded, maximized, handlers, fixed, note }) {
  const gone = GONE.includes(model.state);
  const canOpen = !gone && !model.pending;
  const canMax = canOpen && model.state !== "agent-stopped" && model.state !== "agent-managed";
  const openLabel = OPEN_LABEL[model.kind] || "Open in its tab";
  return (
    <div className={"mx-head" + (fixed ? " is-fixed" : "")} title={model.name + (model.hint ? " — " + model.hint : "")}>
      <span className="mx-face" aria-hidden="true"><PanelFace model={model} /></span>
      <span className="mx-name">{model.name}</span>
      {model.hint ? <span className="mx-hint">{model.hint}</span> : null}
      {note ? <span className="mx-note">{note}</span> : null}
      <span className={"ws-status is-" + model.status}>
        {loaded && model.status === "working" ? <PiSpinner title="Working" /> : null}
        <span>{model.label}</span>
      </span>
      <span className="mx-actions nodrag">
        {canOpen ? (
          <button type="button" className="mx-action" title={openLabel} aria-label={openLabel} onClick={() => handlers.onOpen(model)}>
            <IconExternal size={13} />
          </button>
        ) : null}
        {canMax ? (
          <button type="button" className="mx-action" title={maximized ? "Restore (Esc)" : "Maximize"} aria-label={maximized ? "Restore" : "Maximize"} aria-pressed={!!maximized} onClick={() => handlers.onMaximize(model)}>
            {maximized ? <IconCollapse size={13} /> : <IconExpand size={13} />}
          </button>
        ) : null}
        {!model.pending ? (
          <button type="button" className="mx-action" title="Remove from matrix" aria-label="Remove from matrix" onClick={() => handlers.onRemove(model)}>
            <IconX size={13} />
          </button>
        ) : null}
      </span>
    </div>
  );
}

const PanelInner = memo(function PanelInner({ model, loaded, hidden, engaged, maximized, handlers, bodyKind, still, note }) {
  // Below 0.4 the plate replaces the whole panel: at that zoom the header
  // does not resolve either (C0), so drawing it is noise, not chrome.
  if (bodyKind === "plate") return <PanelPlate model={model} />;
  return (
    <>
      <PanelHead model={model} loaded={loaded} maximized={maximized} handlers={handlers} note={note} />
      <div className="mx-panel-body nodrag nowheel">
        {maximized ? (
          // The body lives in the surface's maximize layer meanwhile; the
          // wrapper keeps the slot and says so (plan §4.6).
          <div className="mx-placeholder mx-state" role="status">
            <span>Shown maximized.</span>
            <button type="button" className="btn btn-sm" onClick={() => handlers.onMaximize(model)}>Restore</button>
          </div>
        ) : bodyKind === "still" && hasPane(model) ? (
          <PanelStill model={model} still={still} />
        ) : (
          <PanelBody
            model={model}
            loaded={loaded}
            hidden={hidden}
            focused={engaged}
            onOpen={() => handlers.onOpen(model)}
            onRemove={() => handlers.onRemove(model)}
            onRun={() => handlers.onRun(model)}
            onOpenFile={(path) => handlers.onOpenFile(model, path)}
            onDirty={(dirty) => handlers.onDirty(model, dirty)}
          />
        )}
      </div>
    </>
  );
});

const Panel = memo(forwardRef(function Panel({ model, loaded, hidden, focused, engaged, maximized, tabStop, loader, handlers, bodyKind = "live", still, note, pointer = true, className, style, children, ...rest }, ref) {
  const rootRef = useRef(null);
  const setRoot = useCallback((el) => {
    rootRef.current = el;
    if (typeof ref === "function") ref(el);
    else if (ref) ref.current = el;
  }, [ref]);
  const id = model.id;
  useEffect(() => {
    if (!loader || !rootRef.current) return undefined;
    loader.observe(id, rootRef.current);
    return () => loader.unobserve(id);
  }, [loader, id]);
  // Whether this body holds an xterm is what decides its zoom row
  // (loadPolicy's `pane`): a body with no cell never goes still. It is told,
  // not guessed, because the loader answers before the body renders.
  const pane = hasPane(model);
  useEffect(() => {
    if (loader) loader.setPane(id, pane);
  }, [loader, id, pane]);
  // is-inert: a live pane away from zoom 1.0 renders and takes keys, but
  // its pointer lies (C0: the mapped cell is `cell × zoom`), so the body
  // takes no pointer at all and the canvas puts a snap-to-1 layer over it.
  // Only a body that *is* a terminal, though: the four rows that answer
  // with one line and one action have no cell to miss, and taking the
  // pointer off them would leave a visible Open or Remove that zooms
  // instead of doing what it says.
  const gated = pane && !maximized;
  const cls = ["mx-panel", className, focused ? "is-focused" : "", maximized ? "is-max" : "", model.pending ? "is-pending" : "",
    bodyKind === "plate" ? "is-plate" : "", gated && bodyKind === "still" ? "is-still" : "", gated && !pointer ? "is-inert" : ""].filter(Boolean).join(" ");
  return (
    <div
      ref={setRoot}
      className={cls}
      style={style}
      role="group"
      aria-label={model.name}
      tabIndex={tabStop ? 0 : -1}
      data-mx-panel={model.id}
      data-state={model.state}
      data-loaded={loaded ? "1" : "0"}
      onPointerDownCapture={(e) => handlers.onFocus(model, !!(e.target.closest && e.target.closest(".xterm")))}
      onFocus={(e) => { if (e.target === e.currentTarget) handlers.onFocus(model, false); }}
      onKeyDown={(e) => handlers.onKey(e, model)}
      {...rest}
    >
      <PanelInner model={model} loaded={loaded} hidden={hidden} engaged={!!(focused && engaged)} maximized={maximized} handlers={handlers} bodyKind={bodyKind} still={still} note={note} />
      {children}
    </div>
  );
}));

export default Panel;
