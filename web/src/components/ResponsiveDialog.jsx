import { createContext, useContext, useLayoutEffect, useRef } from "react";
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
    // Sheet only: Vaul's own input repositioning fights the viewport
    // resize the mobile shell already asks for (interactive-widget=
    // resizes-content) — a focused Name or Provider field left the sheet
    // halfway off the screen. The browser moves the sheet; Vaul must not.
    const extra = desktop ? {} : { repositionInputs: false };
    return (
      <Ctx.Provider value={{ desktop, alert }}>
        <Base {...extra} {...props}>{children}</Base>
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
  return <SheetContent Base={Base} className={className} {...props}>{children}</SheetContent>;
}

// The sheet never focuses a field on its own: on a phone, focus is the
// keyboard, and the keyboard covers half the sheet the user has not read
// yet. Radix's open-autofocus is cancelled (whatever the caller wanted on
// the desktop), and a React `autoFocus` — which fires during commit,
// before any handler — is undone right after mount. A tap on a field is
// the only way the keyboard comes up.
function SheetContent({ Base, className, children, onOpenAutoFocus, ...props }) {
  const ref = useRef(null);
  useLayoutEffect(() => {
    // Focus a field takes on its own — a React `autoFocus` during this
    // commit, a cmdk/Radix effect a tick later — is handed back to the
    // sheet, for the sheet's first 800ms or until the user touches
    // anything. Focus, not blur, so the dialog keeps a focus owner. Vaul
    // does not always forward the ref: fall back to the newest sheet.
    const sheets = document.querySelectorAll(".dlg-sheet");
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
  }, []);
  return (
    <Base
      ref={ref}
      className={[className, "dlg-sheet"].filter(Boolean).join(" ")}
      {...props}
      onOpenAutoFocus={(e) => e.preventDefault()}
    >
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
