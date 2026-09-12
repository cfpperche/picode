import { useEffect, useState } from "react";

// Windows caption buttons for shell mode (ADR-0122): the last slot of the
// merged top row. They call the window the page lives in through Tauri;
// the daemon origin is trusted for these permissions in the shell's
// capability file. In a browser this renders nothing.
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

  const win = currentWindow();
  if (!win) return null;
  const call = (fn) => () => {
    fn(win)?.catch?.((e) => console.error("picode window controls:", e));
  };

  return (
    <div className="shell-caps" role="group" aria-label="Window controls">
      <button type="button" className="cap-btn" title="Minimize" aria-label="Minimize" onClick={call((w) => w.minimize())}>
        <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><path d="M0 5h10" stroke="currentColor" strokeWidth="1" /></svg>
      </button>
      <button type="button" className="cap-btn" title={maximized ? "Restore" : "Maximize"} aria-label={maximized ? "Restore" : "Maximize"} onClick={call((w) => w.toggleMaximize())}>
        {maximized ? (
          <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><rect x="0" y="2.5" width="7" height="7" stroke="currentColor" fill="none" /><path d="M2.5 2.5V0h7v7H7" stroke="currentColor" fill="none" /></svg>
        ) : (
          <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><rect x="0.5" y="0.5" width="9" height="9" stroke="currentColor" fill="none" /></svg>
        )}
      </button>
      <button type="button" className="cap-btn cap-close" title="Close" aria-label="Close" onClick={call((w) => w.close())}>
        <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" strokeWidth="1" /></svg>
      </button>
    </div>
  );
}
