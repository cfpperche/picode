import { useLayoutEffect, useRef } from "react";
import { Drawer } from "vaul";

// ADR-0072: mobile owns its modal presentation. A sheet stays a sheet at
// every preview width. Closing a confirmation resolves as Cancel.
// side="bottom" (default) is the sheet; side="right" is a full-height
// drawer sliding from the right edge (the Inspector over a conversation).
function Root({ children, side = "bottom", ...props }) {
  return <Drawer.Root repositionInputs={false} direction={side} {...props}>{children}</Drawer.Root>;
}
function Overlay({ className, ...props }) {
  return <Drawer.Overlay className={[className, "dlg-overlay-sheet"].filter(Boolean).join(" ")} {...props} />;
}

// A field receives focus only after the user touches or types in the sheet.
function Content({ className, side = "bottom", children, onOpenAutoFocus, ...props }) {
  const ref = useRef(null);
  useLayoutEffect(() => {
    // Focus a field takes on its own — a React `autoFocus` during this
    // commit, a cmdk/Radix effect a tick later — is handed back to the
    // sheet, for the sheet's first 800ms or until the user touches
    // anything. Focus, not blur, so the dialog keeps a focus owner. Vaul
    // does not always forward the ref: fall back to the newest sheet.
    const sheets = document.querySelectorAll(side === "right" ? ".dlg-drawer-right" : ".dlg-sheet");
    const node = ref.current || sheets[sheets.length - 1] || null;
    if (!node) return undefined;
    const isField = (el) => el && el !== node && node.contains(el) && /^(INPUT|SELECT|TEXTAREA)$/.test(el.tagName);
    const take = () => { if (isField(document.activeElement)) node.focus({ preventScroll: true }); };
    take();
    let armed = true;
    const disarm = () => { armed = false; };
    const onFocus = (e) => { if (armed && isField(e.target)) node.focus({ preventScroll: true }); };
    document.addEventListener("focusin", onFocus, true);
    document.addEventListener("pointerdown", disarm, true);
    document.addEventListener("touchstart", disarm, true);
    document.addEventListener("keydown", disarm, true);
    const stop = setTimeout(disarm, 800);
    return () => {
      clearTimeout(stop);
      document.removeEventListener("focusin", onFocus, true);
      document.removeEventListener("pointerdown", disarm, true);
      document.removeEventListener("touchstart", disarm, true);
      document.removeEventListener("keydown", disarm, true);
    };
  }, [side]);
  return (
    <Drawer.Content
      ref={ref}
      data-side={side}
      className={[className, "dlg-sheet", side === "right" ? "dlg-drawer-right" : ""].filter(Boolean).join(" ")}
      {...props}
      onOpenAutoFocus={(e) => e.preventDefault()}
    >
      {side === "bottom" ? <div className="create-handle" aria-hidden="true" /> : null}
      {children}
    </Drawer.Content>
  );
}

const { Portal, Title, Description, Close } = Drawer;
export { Root, Portal, Overlay, Content, Title, Description, Close };
export const Alert = { Root, Portal, Overlay, Content, Title, Description, Close };
