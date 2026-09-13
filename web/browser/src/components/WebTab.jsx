import { useEffect, useRef, useState } from "react";
import { IconGlobe } from "./Icons.jsx";

// Work browser tab surface (Phase 3 slice 1): the React side renders the
// toolbar and an empty region; the actual page is a native WebView2 child
// positioned over that region (bounds pushed on every resize). Desktop
// shell only — a web page cannot host a WebView2, so /browser/ never shows
// this surface.
const invoke = typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

export default function WebTabSurface({ tabId, active, hidden, onMeta, onNew }) {
  const id = tabId.slice(2);
  const [urlDraft, setUrlDraft] = useState("");
  const [started, setStarted] = useState(false);
  const hostRef = useRef(null);

  // Bounds sync: the region's viewport-relative rect drives the native
  // webview. ResizeObserver + window resize cover sidebar, inspector and
  // window moves; switching editor tabs hides instead of resizing.
  useEffect(() => {
    if (!invoke || hidden) return undefined;
    const el = hostRef.current;
    if (!el) return undefined;
    const push = () => {
      const r = el.getBoundingClientRect();
      invoke("btab_bounds", { id, x: r.left, y: r.top, w: r.width, h: r.height }).catch(() => {});
    };
    const ro = new ResizeObserver(push);
    ro.observe(el);
    push();
    window.addEventListener("resize", push);
    return () => {
      ro.disconnect();
      window.removeEventListener("resize", push);
    };
  }, [id, hidden, started]);

  useEffect(() => {
    if (!invoke) return undefined;
    invoke("btab_visibility", { id, visible: !hidden }).catch(() => {});
    return undefined;
  }, [id, hidden]);

  // Mirror the page's location into the toolbar (active tab only).
  useEffect(() => {
    if (!invoke || hidden) return undefined;
    const tick = () =>
      invoke("btab_meta", { id })
        .then((m) => {
          if (m.url) {
            setStarted(true);
            setUrlDraft((cur) => (document.activeElement === urlRef.current ? cur : m.url));
          }
          onMeta?.(m);
        })
        .catch(() => {});
    const t = setInterval(tick, 800);
    tick();
    return () => clearInterval(t);
  }, [id, hidden, started, onMeta]);

  const urlRef = useRef(null);

  function go(u) {
    const target = (u ?? urlDraft).trim();
    if (!target) return;
    const r = hostRef.current?.getBoundingClientRect();
    invoke("btab_navigate", {
      id,
      url: target,
      x: r?.left,
      y: r?.top,
      w: r?.width,
      h: r?.height,
    })
      .then(() => setStarted(true))
      .catch(() => {});
  }

  if (!invoke) {
    return (
      <section className="web-tab-surface" hidden={hidden} aria-label="Work browser">
        <p className="file-pane-msg">The work browser runs in the PiCode desktop app.</p>
      </section>
    );
  }

  return (
    <section className="web-tab-surface" hidden={hidden} aria-label="Work browser">
      <div className="web-tab-toolbar">
        <button type="button" title="Back" onClick={() => invoke("btab_back", { id }).catch(() => {})}>←</button>
        <button type="button" title="Forward" onClick={() => invoke("btab_forward", { id }).catch(() => {})}>→</button>
        <button type="button" title="Reload" onClick={() => invoke("btab_reload", { id }).catch(() => {})}>⟳</button>
        <input
          ref={urlRef}
          value={urlDraft}
          onChange={(e) => setUrlDraft(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") go(); }}
          placeholder="Search or enter a URL"
          spellCheck={false}
        />
        <button type="button" title="Go" onClick={() => go()}>Go</button>
        <button type="button" title="New browser tab" onClick={onNew}>+</button>
      </div>
      <div className="web-tab-host" ref={hostRef} hidden={!started} />
      {!started ? (
        <div className="web-tab-empty">
          <IconGlobe size={28} />
          <h3>Start browsing</h3>
          <p>Enter a URL to open a page</p>
        </div>
      ) : null}
    </section>
  );
}
