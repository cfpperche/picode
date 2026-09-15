import { useEffect, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconCollapse, IconEnter, IconExpand, IconGlobe, IconMonitor, IconSettings, IconX } from "./Icons.jsx";
import { toast } from "../lib/toast.js";
import { isLoopbackUrl } from "@picode/shared/client/devservers.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { askTitle } from "../lib/browserPermissions.js";

// Work browser tab surface (Phase 3 slice 1): the React side renders the
// toolbar and an empty region; the actual page is a native WebView2 child
// positioned over that region (bounds pushed on every resize). Desktop
// shell only — a web page cannot host a WebView2, so /browser/ never shows
// this surface.
const invoke = typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

export default function WebTabSurface({ tabId, url = "", active, hidden, className = "", expanded = false, asks = [], onAnswerAsk, onClose, onMeta, onNew, onBrowserSettings, onToggleExpand }) {
  const id = tabId.slice(2);
  const [urlDraft, setUrlDraft] = useState("");
  const [started, setStarted] = useState(false);
  // The address-bar display pref. It is read by the meta-poll effect's deps
  // below, so it must be declared before them: a `const` read earlier in the
  // same render is a ReferenceError that takes the whole app down (main,
  // commit 4f1a68a7 — a blank shell whenever a work-browser tab rendered).
  const [showFullUrl, setShowFullUrlState] = useState(true);
  const [err, setErr] = useState("");
  const fail = (e) => { setErr(String(e?.message || e)); console.error("btab:", e); };
  const hostRef = useRef(null);
  const pushRef = useRef(null);
  // The parent re-creates onMeta on every render. Keeping it in a ref (and
  // out of the meta effect's deps) is what stops the 800ms poll from
  // re-arming on every state update it just caused: the effect ran tick(),
  // tick() wrote the parent's state, the parent handed back a new onMeta,
  // the effect re-ran — a render loop that pinned a core while a work
  // browser tab was open (found 2026-09-15 in the forced-render scratch).
  const onMetaRef = useRef(onMeta);
  onMetaRef.current = onMeta;

  // Bounds sync: the region's viewport-relative rect drives the native
  // webview. ResizeObserver + window resize cover sidebar, inspector and
  // window moves; switching editor tabs hides instead of resizing.
  useEffect(() => {
    if (!invoke || hidden) return undefined;
    const el = hostRef.current;
    if (!el) return undefined;
    const push = () => {
      const r = el.getBoundingClientRect();
      const off = menuOpenRef.current ? MENU_H : 0;
      invoke("btab_bounds", { id, x: r.left, y: r.top + off, w: r.width, h: r.height - off }).catch(fail);
    };
    const ro = new ResizeObserver(push);
    ro.observe(el);
    push();
    pushRef.current = push;
    window.addEventListener("resize", push);
    return () => {
      ro.disconnect();
      window.removeEventListener("resize", push);
    };
  }, [id, hidden, started]);

  // The Ask bar (slice 3, Browser permissions) takes room above the page, so
  // the host's rect moves when it appears or goes. The native webview has to
  // follow: the ResizeObserver above only fires when the host's size changes,
  // and this is the belt for the cases it misses.
  const ask = asks[0] || null;
  useEffect(() => {
    pushRef.current?.();
  }, [ask?.id]);

  useEffect(() => {
    if (!invoke) return undefined;
    invoke("btab_visibility", { id, visible: !hidden }).catch(fail);
    return () => {
      // Unmount parks the native view. The split pane mounts only for the
      // active tab, so switching tabs unmounts it — without this the
      // WebView2 keeps its last bounds and paints over the next tab
      // (owner report 2026-09-14). The view and its page state survive;
      // remounting shows it again.
      invoke("btab_visibility", { id, visible: false }).catch(() => {});
    };
  }, [id, hidden]);

  // Mirror the page's location into the toolbar (active tab only).
  useEffect(() => {
    if (!invoke || hidden) return undefined;
    const tick = () =>
      invoke("btab_meta", { id })
        .then((m) => {
          if (m.url) {
            setStarted(true);
            setUrlDraft((cur) => (document.activeElement === urlRef.current ? cur : trimUrl(m.url)));
            record("history", m.url, false, m.title);
          }
          onMetaRef.current?.(m);
        })
        .catch(fail);
    const t = setInterval(tick, 800);
    tick();
    return () => clearInterval(t);
  }, [id, hidden, started, showFullUrl]);

  const urlRef = useRef(null);
  const menuOpenRef = useRef(false);
  // Frame fallback (browser shell): no native webview exists, so a local
  // server renders in a frame instead. The address lives here because the
  // desktop shell is the only thing that can read a webview's URL back.
  const [frameUrl, setFrameUrl] = useState(url);
  const [frameNonce, setFrameNonce] = useState(0);
  const local = isLoopbackUrl(frameUrl || url);
  useEffect(() => {
    if (invoke) return;
    setFrameUrl(url || "");
  }, [url]);
  const historySeenRef = useRef("");

  // Address-bar display pref (slice 3): refetch when its setting changes.
  // The state itself is declared with the other useState calls above, because
  // the meta effect names it before this point.
  useEffect(() => {
    fetch("/api/browser/prefs").then((r) => r.json()).then((p) => setShowFullUrlState(p.showFullUrl !== false)).catch(() => {});
    return subscribeFeed((ev) => {
      if (ev.type === "setting.updated") fetch("/api/browser/prefs").then((r) => r.json()).then((p) => setShowFullUrlState(p.showFullUrl !== false)).catch(() => {});
    });
  }, []);

  // Origin-only display: scheme + host (+ port when unusual), no path.
  const trimUrl = (raw) => {
    if (showFullUrl || !raw) return raw;
    try {
      const u = new URL(raw);
      return u.origin === "null" ? raw : u.origin;
    } catch {
      return raw;
    }
  };

  // History recording (slice 3): a URL new to this tab lands as a visit —
  // typed when the user drove the bar, page-driven otherwise. The store
  // updates the newest row in place for re-reports, so the 800 ms poll
  // costs one cheap UPDATE, not rows.
  const record = (which, url, typed, title) => {
    if (which === "history") {
      if (historySeenRef.current === url) return;
      historySeenRef.current = url;
      fetch("/api/browser/history", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url, title: title || "", typed }),
      }).catch(() => {});
    }
  };

  // While the options menu is open, the page webview slides down below the
  // menu's rect — an HTML popover can't paint over a native WebView2
  // sibling, so the page makes room instead. Bounds stay in sync with
  // resize: push() applies the offset whenever it fires.
  const MENU_H = 96;

  const shot = () => invoke && invoke("btab_screenshot", { id })
    .then((path) => { toast.ok(`Screenshot saved to ${path}`); setErr(""); })
    .catch(fail);

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
      .then(() => {
        setStarted(true);
        setErr("");
        record("history", target, true);
      })
      .catch(fail);
  }

  if (!invoke) {
    const typed = (frameUrl || "").trim();
    return (
      <section className="web-tab-surface" hidden={hidden} aria-label="Work browser">
        <div className="web-tab-toolbar">
          <button type="button" title="Reload" onClick={() => setFrameNonce((n) => n + 1)}>⟳</button>
          <div className="web-tab-urlbar">
            <input
              ref={urlRef}
              value={frameUrl}
              onChange={(e) => setFrameUrl(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") setFrameNonce((n) => n + 1); }}
              placeholder="Search or enter a URL"
              spellCheck={false}
            />
            <button type="button" className="web-tab-go" title="Open (Enter)" aria-label="Open" onClick={() => setFrameNonce((n) => n + 1)}><IconEnter /></button>
          </div>
          {onClose ? <button type="button" className="web-tab-menu" title="Close browser pane" aria-label="Close browser pane" onClick={onClose}><IconX /></button> : null}
        </div>
        {local ? (
          <iframe key={frameNonce} className="web-tab-frame" title="Work browser" src={frameUrl || url} />
        ) : (
          <>
            <div className="web-tab-empty">
              <IconGlobe size={28} />
              <h3>Start browsing</h3>
              <p>Enter a URL to open a page</p>
            </div>
            {typed ? (
              <p className="file-pane-msg">Only servers on this machine open without the desktop app — anything else opens in your browser.</p>
            ) : null}
          </>
        )}
      </section>
    );
  }

  return (
    <section className={"web-tab-surface" + (className ? " " + className : "")} hidden={hidden} aria-label="Work browser">
      <div className="web-tab-toolbar">
        <button type="button" title="Back" onClick={() => invoke("btab_back", { id }).catch(() => {})}>←</button>
        <button type="button" title="Forward" onClick={() => invoke("btab_forward", { id }).catch(() => {})}>→</button>
        <button type="button" title="Reload" onClick={() => invoke("btab_reload", { id }).catch(() => {})}>⟳</button>
        <div className="web-tab-urlbar">
          <input
            ref={urlRef}
            value={urlDraft}
            onChange={(e) => setUrlDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") go(); }}
            placeholder="Search or enter a URL"
            spellCheck={false}
          />
          <button type="button" className="web-tab-go" title="Open (Enter)" aria-label="Open" onClick={() => go()}><IconEnter /></button>
        </div>
        <DropdownMenu.Root onOpenChange={(o) => { menuOpenRef.current = o; pushRef.current?.(); }}>
          <DropdownMenu.Trigger asChild>
            <button type="button" className="web-tab-menu" title="Browser options" aria-label="Browser options">⋮</button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content align="end" sideOffset={6} collisionPadding={8} className="web-tab-menu-list" onCloseAutoFocus={(e) => e.preventDefault()}>
              <DropdownMenu.Item className="um-item" onSelect={shot}><span className="um-item-name"><IconMonitor />Take a screenshot</span></DropdownMenu.Item>
              <DropdownMenu.Item className="um-item" onSelect={() => onBrowserSettings?.()}><span className="um-item-name"><IconSettings />Browser settings</span></DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
        {err ? <span className="web-tab-err" title={err}>{err}</span> : null}
        {onToggleExpand ? (
          <button type="button" className="web-tab-menu" title={expanded ? "Back to split" : "Expand browser"} aria-label={expanded ? "Back to split" : "Expand browser"} onClick={onToggleExpand}>
            {expanded ? <IconCollapse /> : <IconExpand />}
          </button>
        ) : null}
        {onClose ? <button type="button" className="web-tab-menu" title="Close browser pane" aria-label="Close browser pane" onClick={onClose}><IconX /></button> : null}
      </div>
      {ask ? (
        <div className="web-tab-ask" role="alert" aria-label="Site permission request">
          <div className="web-tab-ask-main">
            <span className="web-tab-ask-t">{askTitle(ask)}</span>
            <span className="web-tab-ask-d">
              Allow or block this request. Always allow remembers it for this site.
              {asks.length > 1 ? ` ${asks.length - 1} more waiting.` : ""}
            </span>
          </div>
          <div className="web-tab-ask-actions">
            <button type="button" className="btn btn-primary" onClick={() => onAnswerAsk(ask, "allow", false)}>Allow</button>
            <button type="button" className="btn" onClick={() => onAnswerAsk(ask, "deny", false)}>Block</button>
            <button type="button" className="btn btn-ghost" onClick={() => onAnswerAsk(ask, "allow", true)}>Always allow</button>
          </div>
        </div>
      ) : null}
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
