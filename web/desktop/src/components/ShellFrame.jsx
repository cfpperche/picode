import { useEffect, useState } from "react";

// Window controls for shell mode. The desktop shell (Tauri, ADR-0120) runs
// this UI undecorated — no native title bar — so the app renders its own:
// a small cluster pinned to the window's top-right corner, plus the drag
// regions the shell marks (the sidebar brand row). In a browser
// `window.__PICODE_SHELL__` is never set and this component renders nothing,
// so the web product is untouched.
const isShell = () => typeof window !== "undefined" && window.__PICODE_SHELL__ === true;

function currentWindow() {
  try {
    return window.__TAURI__?.window?.getCurrentWindow?.() ?? null;
  } catch {
    return null;
  }
}

export default function ShellFrame() {
  const [maximized, setMaximized] = useState(false);
  const [gone, setGone] = useState(!isShell());

  useEffect(() => {
    if (!isShell()) return undefined;
    const win = currentWindow();
    if (!win) {
      // Shell mode without the API bridge: keep the undecorated window
      // usable by saying so instead of showing dead buttons.
      setGone(true);
      return undefined;
    }
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

  if (gone) return null;
  const win = currentWindow();
  const call = (fn) => () => {
    try {
      fn(win);
    } catch {
      /* never let a window action break the app */
    }
  };

  return (
    <div className="shell-controls" data-tauri-drag-region aria-label="Window controls">
      <button
        type="button"
        className="shell-btn"
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
        className="shell-btn"
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
        className="shell-btn shell-btn-close"
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
