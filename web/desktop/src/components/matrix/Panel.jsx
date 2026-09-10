import { forwardRef, memo, useCallback, useEffect, useRef } from "react";
import { ProviderFace } from "../ProviderFaces.jsx";
import TerminalCliBadge from "../TerminalCliBadge.jsx";
import PiSpinner from "../PiSpinner.jsx";
import { IconCollapse, IconExpand, IconExternal, IconX } from "../Icons.jsx";
import PanelBody from "./PanelBody.jsx";

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

// PanelHead — face, name, hint, the sidebar's chip, Open · Maximize ·
// Remove. The wrapper's header is the drag handle; the maximize layer
// renders the same head fixed.
export function PanelHead({ model, loaded, maximized, handlers, fixed }) {
  const gone = model.state === "terminal-gone" || model.state === "agent-gone";
  const canOpen = !gone && !model.pending;
  const canMax = canOpen && model.state !== "agent-stopped" && model.state !== "agent-managed";
  return (
    <div className={"mx-head" + (fixed ? " is-fixed" : "")} title={model.name + (model.hint ? " — " + model.hint : "")}>
      <span className="mx-face" aria-hidden="true">
        {model.kind === "agent"
          ? (model.target ? <ProviderFace agent={model.target} /> : <span className="ws-face">?</span>)
          : <TerminalCliBadge term={model.target || {}} decorative />}
      </span>
      <span className="mx-name">{model.name}</span>
      {model.hint ? <span className="mx-hint">{model.hint}</span> : null}
      <span className={"ws-status is-" + model.status}>
        {loaded && model.status === "working" ? <PiSpinner title="Working" /> : null}
        <span>{model.label}</span>
      </span>
      <span className="mx-actions">
        {canOpen ? (
          <button type="button" className="mx-action" title="Open in its tab" aria-label="Open in its tab" onClick={() => handlers.onOpen(model)}>
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

const PanelInner = memo(function PanelInner({ model, loaded, hidden, engaged, maximized, handlers }) {
  return (
    <>
      <PanelHead model={model} loaded={loaded} maximized={maximized} handlers={handlers} />
      <div className="mx-panel-body">
        {maximized ? (
          // The body lives in the surface's maximize layer meanwhile; the
          // wrapper keeps the slot and says so (plan §4.6).
          <div className="mx-placeholder mx-state" role="status">
            <span>Shown maximized.</span>
            <button type="button" className="btn btn-sm" onClick={() => handlers.onMaximize(model)}>Restore</button>
          </div>
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
          />
        )}
      </div>
    </>
  );
});

const Panel = memo(forwardRef(function Panel({ model, loaded, hidden, focused, engaged, maximized, tabStop, loader, handlers, className, style, children, ...rest }, ref) {
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
  const cls = ["mx-panel", className, focused ? "is-focused" : "", maximized ? "is-max" : "", model.pending ? "is-pending" : ""].filter(Boolean).join(" ");
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
      <PanelInner model={model} loaded={loaded} hidden={hidden} engaged={!!(focused && engaged)} maximized={maximized} handlers={handlers} />
      {children}
    </div>
  );
}));

export default Panel;
