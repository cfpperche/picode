import { createContext, useContext } from "react";
import * as RadixDialog from "@radix-ui/react-dialog";
import * as RadixAlert from "@radix-ui/react-alert-dialog";
import { Drawer } from "vaul";
import { useMedia } from "../lib/media.js";

// The one modal primitive (ADR-0045). Same API as @radix-ui/react-dialog —
// Root / Portal / Overlay / Content / Title / Description / Close — so a
// file switches by changing its import line. At and above 720px it IS the
// Radix dialog (centred card); below, the same tree renders as a Vaul
// bottom sheet (handle, swipe to dismiss, safe-area padding) — the shape
// every mobile benchmark uses for a modal. `.dlg-sheet` in app.css does
// the visual switch, so the `dlg dlg-*` class names keep meaning "this
// dialog's content" on both.
//
// Raw imports of @radix-ui/react-dialog, react-alert-dialog or vaul
// outside this file are refused by web/src/lib/dialogPolicy.test.js;
// the two exceptions (Palette, Hotkeys) are desktop-only surfaces.
export const DIALOG_DESKTOP_QUERY = "(min-width: 720px)";

const Ctx = createContext({ desktop: true, alert: false });

function lib(desktop, alert) {
  if (!desktop) return Drawer;
  return alert ? RadixAlert : RadixDialog;
}

function makeRoot(alert) {
  return function Root({ children, ...props }) {
    const desktop = useMedia(DIALOG_DESKTOP_QUERY);
    const Base = lib(desktop, alert).Root;
    return (
      <Ctx.Provider value={{ desktop, alert }}>
        <Base {...props}>{children}</Base>
      </Ctx.Provider>
    );
  };
}

function Portal(props) {
  const { desktop, alert } = useContext(Ctx);
  const Base = lib(desktop, alert).Portal;
  return <Base {...props} />;
}

function Overlay({ className, ...props }) {
  const { desktop, alert } = useContext(Ctx);
  const Base = lib(desktop, alert).Overlay;
  return <Base className={[className, desktop ? "" : "dlg-overlay-sheet"].filter(Boolean).join(" ")} {...props} />;
}

function Content({ className, children, ...props }) {
  const { desktop, alert } = useContext(Ctx);
  const Base = lib(desktop, alert).Content;
  if (desktop) return <Base className={className} {...props}>{children}</Base>;
  return (
    <Base className={[className, "dlg-sheet"].filter(Boolean).join(" ")} {...props}>
      <div className="create-handle" aria-hidden="true" />
      {children}
    </Base>
  );
}

function Title(props) {
  const { desktop, alert } = useContext(Ctx);
  const Base = lib(desktop, alert).Title;
  return <Base {...props} />;
}

function Description(props) {
  const { desktop, alert } = useContext(Ctx);
  const Base = lib(desktop, alert).Description;
  return <Base {...props} />;
}

function Close(props) {
  const { desktop, alert } = useContext(Ctx);
  // AlertDialog has Cancel where Dialog has Close; both dismiss.
  const L = lib(desktop, alert);
  const Base = L.Close || L.Cancel;
  return <Base {...props} />;
}

const Root = makeRoot(false);
export { Root, Portal, Overlay, Content, Title, Description, Close };

// Alert flavour: Radix AlertDialog on the desktop (no dismiss on overlay
// click, focus trapped on the decision); the same sheet on the phone,
// where a swipe down is the cancel.
export const Alert = { Root: makeRoot(true), Portal, Overlay, Content, Title, Description, Close };
