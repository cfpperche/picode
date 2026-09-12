import { useEffect, useState } from "react";
import { isShell } from "../lib/shellFrame.js";

// Window controls for shell mode (ADR-0123). The desktop shell runs this
// UI undecorated; when the shell flag is set the UI claims the frame and
// draws its own minimize/maximize/close in a reserved top-right slot, and
// every view's top bar pads itself clear of them (styles keyed off
// `data-picode-frame` on the root element). The shell's injected fallback
// frame sees the claim and retires. In a browser this renders nothing.
function currentWindow() {
  try {
    return window.__TAURI__?.window?.getCurrentWindow?.() ?? null;
  } catch {
    return null;
  }
}

export default function WindowControls() {
  const [maximized, setMaximized] = useState(false);

  useEffect(() => {
    if (!isShell()) return undefined;
    const win = currentWindow();
    if (!win) return undefined;
    let unlisten;
    const sync = async () => {
      try {
        setMaximized(await win.isMaximized());
      } catch {
        /* the window went away */
      }
    };
    sync();
    win
      .onResized(sync)
      .then((off) => {
        unlisten = off;
      })
      .catch(() => {});
    return () => {
      if (unlisten) unlisten();
    };
  }, []);

  if (!isShell()) return null;
  const win = currentWindow();
  if (!win) return null;
  const call = (fn) => () => {
    fn(win)?.catch?.((e) => console.error("picode window controls:", e));
  };

  return (
    <div className="win-controls" role="group" aria-label="Window controls">
      <button
        type="button"
        className="win-btn"
        title="Minimize"
        aria-label="Minimize"
        onClick={call((w) => w.minimize())}
      >
        <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
          <path d="M2 6h8" stroke="currentColor" strokeWidth="1.2" fill="none" />
        </svg>
      </button>
      <button
        type="button"
        className="win-btn"
        title={maximized ? "Restore" : "Maximize"}
        aria-label={maximized ? "Restore" : "Maximize"}
        onClick={call((w) => w.toggleMaximize())}
      >
        {maximized ? (
          <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
            <rect x="1.5" y="3.5" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="1.2" fill="none" />
            <path d="M3.8 2.2V1.5h7v7h-.7" stroke="currentColor" strokeWidth="1.2" fill="none" />
          </svg>
        ) : (
          <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
            <rect x="2" y="2" width="8" height="8" rx="1" stroke="currentColor" strokeWidth="1.2" fill="none" />
          </svg>
        )}
      </button>
      <button
        type="button"
        className="win-btn win-btn-close"
        title="Close"
        aria-label="Close"
        onClick={call((w) => w.close())}
      >
        <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
          <path d="M2.5 2.5l7 7m0-7l-7 7" stroke="currentColor" strokeWidth="1.2" fill="none" />
        </svg>
      </button>
    </div>
  );
}
