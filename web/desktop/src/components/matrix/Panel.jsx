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
// Two layers on purpose: the grid re-renders every wrapper on a drag (its
// clone carries a fresh style object), so the outer memo cannot hold —
// the inner one, keyed on the domain props only, does. An xterm body never
// remounts because the surface re-rendered.
const PanelInner = memo(function PanelInner({ model, loaded, hidden, focused, maximized, handlers }) {
  const gone = model.state === "terminal-gone" || model.state === "agent-gone";
  const canOpen = !gone && !model.pending;
  const canMax = canOpen && model.state !== "agent-stopped" && model.state !== "agent-managed";
  return (
    <>
      <div className="mx-head" title={model.name + (model.hint ? " — " + model.hint : "")}>
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
            <button type="button" className="mx-action" title={maximized ? "Restore" : "Maximize"} aria-label={maximized ? "Restore" : "Maximize"} aria-pressed={!!maximized} onClick={() => handlers.onMaximize(model)}>
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
      <div className="mx-panel-body">
        <PanelBody
          model={model}
          loaded={loaded}
          hidden={hidden}
          focused={focused}
          onOpen={() => handlers.onOpen(model)}
          onRemove={() => handlers.onRemove(model)}
          onRun={() => handlers.onRun(model)}
          onOpenFile={(path) => handlers.onOpenFile(model, path)}
        />
      </div>
    </>
  );
});

const Panel = memo(forwardRef(function Panel({ model, loaded, hidden, focused, maximized, loader, handlers, className, style, children, ...rest }, ref) {
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
      data-state={model.state}
      data-loaded={loaded ? "1" : "0"}
      onPointerDownCapture={() => handlers.onFocus(model)}
      {...rest}
    >
      <PanelInner model={model} loaded={loaded} hidden={hidden} focused={focused} maximized={maximized} handlers={handlers} />
      {children}
    </div>
  );
}));

export default Panel;
