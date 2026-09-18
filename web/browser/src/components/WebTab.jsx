import { useEffect, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconChevronRight, IconCollapse, IconEnter, IconExpand, IconGlobe, IconMonitor, IconReload, IconSettings, IconX } from "./Icons.jsx";
import { toast } from "../lib/toast.js";
import { isLoopbackUrl } from "@picode/shared/client/devservers.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { askTitle } from "../lib/browserPermissions.js";
import { subscribeFloatingLayers, overlapsLayers, rectOf } from "../lib/floatingLayers.js";
import { requestBrowserDialog } from "../lib/browserDialogs.js";
import { previewUrl, verifyPreviewUrl } from "../lib/previewStill.js";
import { createPortal } from "react-dom";
import { cropRect, parsePick, pickLabel, pickScript, stillToViewport, stylesToCSS } from "../lib/annotate.js";

// Work browser tab surface (Phase 3 slice 1): the React side renders the
// toolbar and an empty region; the actual page is a native WebView2 child
// positioned over that region (bounds pushed on every resize). Desktop
// shell only — a web page cannot host a WebView2, so /browser/ never shows
// this surface.
const invoke = typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

export default function WebTabSurface({ tabId, url = "", active, hidden, className = "", expanded = false, chromeless = false, asks = [], onAnswerAsk, onClose, onMeta, onNew, onBrowserSettings, onToggleExpand }) {
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

  const [menuOpen, setMenuOpen] = useState(false);
  const [still, setStill] = useState(""); // the page, frozen while the menu is over it
  const [previewFailed, setPreviewFailed] = useState(false); // the capture proved worthless
  const [zoom, setZoom] = useState(1);
  const [findOpen, setFindOpen] = useState(false);
  const [findQuery, setFindQuery] = useState("");
  const [findState, setFindState] = useState({ count: 0, active: 0 });
  const [overlayCover, setOverlayCover] = useState(false);

  // Any floating layer that reaches the page — the tab-strip menu, the
  // command palette, a toast, a dropdown — needs the native view out of the
  // way, and the rule is geometry, not a list of remembered overlays
  // (2026-09-16: the owner's editor tab menu came up invisible under the
  // page). The same list drives the clipping audit, so a new floating
  // surface is seen by both or by neither.
  useEffect(() => subscribeFloatingLayers((layers) => {
    const host = hostRef.current;
    setOverlayCover(overlapsLayers(host ? rectOf(host) : null, layers));
  }), []);

  // The menu opens over the page, and a native WebView2 paints over HTML —
  // so the tab freezes the page into a still and hides the webview while the
  // menu is up. The capture is prefetched on hover. A blob URL is not a
  // still until it decodes to real pixels (an empty capture resolves the
  // IPC call fine and makes a truthy URL — hiding the live page behind
  // that is the uniform gray the owner saw on x.com, 2026-09-16).
  //
  // covered =
  // | menu | verified still | capture in flight | capture failed | dialog open | webview  |
  // | ---- | -------------- | ----------------- | -------------- | ----------- | -------- |
  // | open | yes            | —                 | —              | —           | hidden (the frozen page sits behind the menu) |
  // | open | no             | yes               | —              | —           | **visible** (the menu waits for this row to resolve — see below) |
  // | open | no             | no                | yes            | —           | hidden (host gray; the menu still works) |
  // | shut | —              | —                 | —              | no          | visible |
  // | —    | —              | —                 | —              | yes         | hidden (the dialog dims the window) |
  //
  // The waiting row is why `menuOpen` is only set once there is something to
  // sit behind the menu: hiding the live page before its still exists showed
  // the host's gray for the length of a capture, and the owner read that as
  // the page blinking on every menu click (2026-09-17).
  const previewRef = useRef({ url: "", at: 0, inflight: null });
  useEffect(() => () => {
    if (previewRef.current.url) URL.revokeObjectURL(previewRef.current.url);
  }, []);
  const grabPreview = (maxAge = 1200) => {
    if (!invoke) return Promise.resolve("");
    const cur = previewRef.current;
    if (cur.url && Date.now() - cur.at < maxAge) return Promise.resolve(cur.url);
    if (cur.inflight) return cur.inflight;
    const inflight = invoke("btab_preview", { id })
      .then((bytes) => previewUrl(bytes))
      .then((url) => (url ? verifyPreviewUrl(url).catch(() => "") : ""))
      .then((url) => {
        if (!url) throw new Error("preview: unusable capture");
        if (previewRef.current.url) URL.revokeObjectURL(previewRef.current.url);
        previewRef.current = { url, at: Date.now(), inflight: null };
        return url;
      })
      .catch(() => {
        previewRef.current.inflight = null;
        return "";
      });
    previewRef.current.inflight = inflight;
    return inflight;
  };

  // ---- annotate mode (v2c step 1): point at an element and send it on ----
  // The native page is hidden (btab_visibility) and its frozen still is shown
  // instead, because HTML can never paint over a WebView2 child. Clicks land
  // on the still, resolve to a viewport point, and come back as one element
  // through the same CDP door the agent's browser verbs use.
  const [annotOn, setAnnotOn] = useState(false);
  const [annotStill, setAnnotStill] = useState("");
  const [annotPick, setAnnotPick] = useState(null);
  const [annotComment, setAnnotComment] = useState("");
  const [annotWantShot, setAnnotWantShot] = useState(true);
  const [annotShots, setAnnotShots] = useState("ask"); // always | ask | never
  const [annotTerminals, setAnnotTerminals] = useState([]);
  const [annotTerminal, setAnnotTerminal] = useState("");
  const [annotBusy, setAnnotBusy] = useState(false);
  const stillRef = useRef(null);

  useEffect(() => {
    if (!annotOn) return undefined;
    let alive = true;
    fetch("/api/browser/prefs")
      .then((r) => (r.ok ? r.json() : null))
      .then((p) => { if (alive && p && (p.annotationShots === "always" || p.annotationShots === "never")) setAnnotShots(p.annotationShots); })
      .catch(() => {});
    fetch("/api/terminals")
      .then((r) => (r.ok ? r.json() : null))
      .then((body) => {
        if (!alive) return;
        const list = Array.isArray(body) ? body : body?.terminals || [];
        const live = list.filter((t) => t && t.running);
        setAnnotTerminals(live);
        setAnnotTerminal((cur) => cur || (live[0]?.id ?? ""));
      })
      .catch(() => {});
    return () => { alive = false; };
  }, [annotOn]);

  // Leaving the mode (or unmounting with it on) must always give the live page
  // back: a hidden view with no overlay would look like a dead tab.
  useEffect(() => {
    if (!annotOn) return undefined;
    return () => { if (invoke) invoke("btab_visibility", { id, visible: true }).catch(() => {}); };
  }, [annotOn, id]);

  const startAnnotate = async () => {
    if (!invoke) return;
    const still = await grabPreview(0);
    if (!still) { toast("Could not freeze the page for annotating."); return; }
    setAnnotStill(still);
    setAnnotPick(null);
    setAnnotComment("");
    setAnnotWantShot(true);
    setAnnotOn(true);
    invoke("btab_visibility", { id, visible: false }).catch(() => {});
  };
  const stopAnnotate = () => {
    setAnnotOn(false);
    setAnnotPick(null);
  };

  const pickAt = async (event) => {
    const img = event.currentTarget;
    const box = img.getBoundingClientRect();
    const point = stillToViewport({
      clickX: event.clientX - box.left,
      clickY: event.clientY - box.top,
      naturalWidth: img.naturalWidth,
      naturalHeight: img.naturalHeight,
      displayWidth: box.width,
      displayHeight: box.height,
    });
    if (!point) { toast("That point is not on the frozen page."); return; }
    const scale = { sx: box.width / (img.naturalWidth || 1), sy: box.height / (img.naturalHeight || 1) };
    let res;
    try {
      res = await invoke("btab_cdp_call", {
        id,
        method: "Runtime.evaluate",
        params_json: JSON.stringify({ expression: pickScript(point.x, point.y), returnByValue: true }),
        tier: "read",
        raw: false,
      });
    } catch (e) {
      toast("Could not read the page element: " + (e?.message || e));
      return;
    }
    const pick = parsePick(res);
    if (!pick) { toast("No element answered at that point."); return; }
    setAnnotPick({ ...pick, scale });
  };

  const croppedShot = () => {
    const img = stillRef.current;
    if (!img || !annotPick) return "";
    const box = img.getBoundingClientRect();
    const rect = cropRect({
      rect: annotPick.rect,
      naturalWidth: img.naturalWidth,
      naturalHeight: img.naturalHeight,
      displayWidth: box.width,
      displayHeight: box.height,
    });
    if (!rect) return "";
    const canvas = document.createElement("canvas");
    canvas.width = rect.width;
    canvas.height = rect.height;
    const ctx = canvas.getContext("2d");
    if (!ctx) return "";
    ctx.drawImage(img, rect.x, rect.y, rect.width, rect.height, 0, 0, rect.width, rect.height);
    const dataUrl = canvas.toDataURL("image/png");
    return dataUrl.startsWith("data:image/png;base64,") ? dataUrl.slice("data:image/png;base64,".length) : "";
  };

  const saveAnnotation = async () => {
    if (!annotPick) { toast("Pick an element on the page first."); return; }
    setAnnotBusy(true);
    const wantImage = annotShots === "always" || (annotShots === "ask" && annotWantShot);
    try {
      const res = await fetch("/api/browser/annotations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          terminalId: annotTerminal,
          url: liveUrlRef.current || url,
          title: "",
          selector: annotPick.selector,
          comment: annotComment,
          dom: annotPick.html,
          css: stylesToCSS(annotPick.styles),
          image: wantImage ? croppedShot() : "",
        }),
      });
      if (!res.ok) {
        const text = await res.text().catch(() => "");
        toast("The annotation was not saved: " + (text || res.status));
        return;
      }
      const body = await res.json().catch(() => ({}));
      const paths = Array.isArray(body.paths) ? body.paths : [];
      const note = annotComment.trim();
      if (note && annotTerminal && paths.length) {
        const drop = await fetch(`/api/terminals/${encodeURIComponent(annotTerminal)}/drop`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ message: note, paths }),
        }).catch(() => null);
        if (drop && drop.ok) toast.ok("Annotation sent to the agent.");
        else toast.ok("Annotation saved; nothing was sent (the terminal is not running an agent CLI).");
      } else {
        toast.ok("Annotation saved.");
      }
      stopAnnotate();
    } catch (e) {
      toast("The annotation was not saved: " + (e?.message || e));
    } finally {
      setAnnotBusy(false);
    }
  };
  // A menu open has to arrive with its backdrop: the still (the frozen page
  // the HTML sits on) is prefetched on hover and focus, and when it is not
  // there yet the menu waits for the capture instead of parking the live page
  // behind a strip of gray. A failed capture still opens it — over the host
  // gray, which is the honest "we could not freeze this page".
  const onMenuOpenChange = (open) => {
    if (!open) {
      setMenuOpen(false);
      setStill("");
      setPreviewFailed(false);
      return;
    }
    // A plain browser has no native page to cover: nothing to wait for.
    if (!invoke) {
      setMenuOpen(true);
      return;
    }
    const opened = () => {
      setMenuOpen(true);
      if (started) invoke("btab_zoom", { id }).then((z) => { if (typeof z === "number") setZoom(z); }).catch(() => {});
    };
    if (previewRef.current.url) {
      setStill(previewRef.current.url);
      setPreviewFailed(false);
      opened();
      return;
    }
    // The capture is a shell round-trip; if it stalls, the menu opens anyway
    // (over the host gray) instead of a click that appears to do nothing.
    const timed = new Promise((r) => setTimeout(() => r(""), 600));
    Promise.race([grabPreview(), timed]).then((url) => {
      if (url) {
        setStill(url);
        setPreviewFailed(false);
      } else {
        setPreviewFailed(true);
      }
      opened();
    });
  };
  const covered = overlayCover || (menuOpen && (!!still || previewFailed));
  // Whatever the reason, a covered page shows the frozen frame instead of
  // the host's empty background. The menu prefetches on hover; every other
  // layer takes the capture when it first covers the page (cached, so the
  // second time is instant).
  useEffect(() => {
    if (!covered) {
      // The still belongs to the moment it was taken: keeping it would show a
      // stale frame the next time anything covers the page.
      setStill("");
      return undefined;
    }
    if (still) return undefined;
    let live = true;
    grabPreview().then((url) => {
      if (live && url) setStill(url);
    });
    return () => {
      live = false;
    };
  }, [covered, still]);
  // A tab switch hides without unmounting, and Radix never fires
  // onOpenChange(false) for a menu that unmounts open: park the cover
  // state here or the webview stays hidden behind a menu that is gone.
  useEffect(() => {
    if (!hidden) return;
    setMenuOpen(false);
    setStill("");
    setPreviewFailed(false);
  }, [hidden]);

  // Bounds sync: the region's viewport-relative rect drives the native
  // webview. ResizeObserver + window resize cover sidebar, inspector and
  // window moves; switching editor tabs hides instead of resizing.
  useEffect(() => {
    if (!invoke || hidden) return undefined;
    const el = hostRef.current;
    if (!el) return undefined;
    const push = () => {
      const r = el.getBoundingClientRect();
      invoke("btab_bounds", { id, x: r.left, y: r.top, w: r.width, h: r.height }).catch(fail);
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
    // hidden covers tab selection AND the route: the settings views paint
    // over the pane, so App parks the tab (onPane) while the route is
    // elsewhere — otherwise the page keeps its bounds and covers them
    // (2026-09-16: Browser settings opened behind the live x.com view).
    invoke("btab_visibility", { id, visible: !hidden && !covered }).catch(fail);
    return () => {
      // Unmount parks the native view. The split pane mounts only for the
      // active tab, so switching tabs unmounts it — without this the
      // WebView2 keeps its last bounds and paints over the next tab
      // (owner report 2026-09-14). The view and its page state survive;
      // remounting shows it again.
      invoke("btab_visibility", { id, visible: false }).catch(() => {});
    };
  }, [id, hidden, covered]);

  // App mode keeps no toolbar: Ctrl+F summons the find bar over the page
  // (WebView2 ships no native find UI); reload and zoom stay engine
  // accelerators (F5, Ctrl+R, Ctrl +/-) and the page's right-click menu is
  // the engine's own. Passwords/downloads/history live in Settings ▸
  // Browser, where the toolbar menu only ever linked.
  useEffect(() => {
    if (!invoke || !chromeless || !active || hidden) return undefined;
    const onKey = (e) => {
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === "f") {
        e.preventDefault();
        setFindOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [invoke, chromeless, active, hidden]);

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

  // Zoom steps: the reference's own ladder, 25% to 500%.
  const ZOOM_STEPS = [0.25, 0.33, 0.5, 0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4, 5];

  const applyZoom = (factor) => {
    if (!invoke) return;
    invoke("btab_set_zoom", { id, factor })
      .then((next) => setZoom(typeof next === "number" ? next : factor))
      .catch((e) => toast("Zoom failed: " + (e?.message || e)));
  };
  const zoomStep = (dir) => {
    const at = ZOOM_STEPS.findIndex((z) => z >= zoom - 0.001);
    const next = ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, (at < 0 ? ZOOM_STEPS.indexOf(1) : at) + dir))];
    if (next && next !== zoom) applyZoom(next);
  };

  const printPage = () => invoke && invoke("btab_print", { id })
    .catch((e) => toast("Printing failed: " + (e?.message || e)));

  // Find in page: the runtime's find session does the highlighting, the bar
  // above the page shows where it landed. A null `forward` starts a fresh
  // search (the input changed); an empty query stops and clears.
  const runFind = (query, forward) => {
    if (!invoke) return;
    invoke("btab_find", { id, query: query ?? "", forward: forward ?? null })
      .then((state) => setFindState({ count: state?.count || 0, active: state?.active || 0 }))
      .catch((e) => toast("Find failed: " + (e?.message || e)));
  };
  const closeFind = () => {
    setFindOpen(false);
    setFindQuery("");
    setFindState({ count: 0, active: 0 });
    runFind("", null);
  };

  // The input searches as you type; 250ms of quiet is enough to ask.
  useEffect(() => {
    if (!invoke || !findOpen) return undefined;
    const t = setTimeout(() => runFind(findQuery, null), 250);
    return () => clearTimeout(t);
  }, [findQuery, findOpen]);

  // The menu's History/Downloads/Clear data/Passwords items: the dialogs
  // live in Settings ▸ Browser (option A, owner 2026-09-15), so the request
  // rides the route change and the settings page opens it.
  const openBrowserDialog = (name) => {
    requestBrowserDialog(name);
    onBrowserSettings?.();
  };

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
      {!chromeless ? (
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
          <button
          type="button"
          className={"web-tab-annot-btn" + (annotOn ? " on" : "")}
          title={annotOn ? "Stop annotating" : "Point at an element and send it to an agent"}
          aria-label="Annotate an element"
          aria-pressed={annotOn}
          onClick={() => (annotOn ? stopAnnotate() : startAnnotate())}
        >✎</button>
        {annotOn && annotStill
          ? createPortal(
              <div className="annot-veil" role="dialog" aria-label="Annotate an element">
                <div className="annot-head">
                  <strong>{annotPick ? "Element picked" : "Click the element to annotate"}</strong>
                  {annotPick ? <span className="annot-sub">{pickLabel(annotPick)}</span> : null}
                  <button type="button" className="set-btn" onClick={stopAnnotate}>Cancel</button>
                </div>
                <div className="annot-stage">
                  <div className="annot-img-wrap">
                    <img
                      ref={stillRef}
                      src={annotStill}
                      alt="Frozen page"
                      onClick={pickAt}
                      draggable={false}
                    />
                    {annotPick ? (
                      <span
                        className="annot-outline"
                        style={{
                          left: annotPick.rect.x * annotPick.scale.sx,
                          top: annotPick.rect.y * annotPick.scale.sy,
                          width: annotPick.rect.width * annotPick.scale.sx,
                          height: annotPick.rect.height * annotPick.scale.sy,
                        }}
                      />
                    ) : null}
                  </div>
                </div>
                <div className="annot-foot">
                  <input
                    className="annot-note"
                    value={annotComment}
                    onChange={(e) => setAnnotComment(e.target.value)}
                    placeholder="What should change here?"
                    aria-label="Annotation comment"
                  />
                  {annotShots === "always" ? <span className="annot-sub">Screenshot: always</span> : null}
                  {annotShots === "never" ? <span className="annot-sub">Screenshot: never</span> : null}
                  {annotShots === "ask" ? (
                    <label className="annot-check">
                      <input type="checkbox" checked={annotWantShot} onChange={(e) => setAnnotWantShot(e.target.checked)} />
                      Include the screenshot
                    </label>
                  ) : null}
                  <select
                    className="set-select"
                    value={annotTerminal}
                    onChange={(e) => setAnnotTerminal(e.target.value)}
                    aria-label="Send the annotation to"
                  >
                    {annotTerminals.length === 0 ? <option value="">No agent terminal running</option> : null}
                    {annotTerminals.map((t) => (
                      <option key={t.id} value={t.id}>{t.name || t.id}</option>
                    ))}
                  </select>
                  <button type="button" className="set-btn" onClick={saveAnnotation} disabled={annotBusy || !annotPick}>
                    {annotBusy ? "Saving…" : "Save and send"}
                  </button>
                </div>
              </div>,
              document.body,
            )
          : null}
        <DropdownMenu.Root open={menuOpen} onOpenChange={onMenuOpenChange}>
            <DropdownMenu.Trigger asChild>
              <button
                type="button"
                className="web-tab-menu"
                title="Browser options"
                aria-label="Browser options"
                onPointerEnter={() => grabPreview(1500)}
                onPointerDown={() => grabPreview(1500)}
                onFocus={() => grabPreview(1500)}
              >⋮</button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content align="end" sideOffset={6} collisionPadding={8} className="web-tab-menu-list" onCloseAutoFocus={(e) => e.preventDefault()}>
                <DropdownMenu.Item className="um-item" onSelect={() => setFindOpen(true)}>Find in page</DropdownMenu.Item>
                <DropdownMenu.Item className="um-item" onSelect={printPage}>Print</DropdownMenu.Item>
                <DropdownMenu.Separator className="web-tab-menu-sep" />
                <div className="web-tab-zoom" role="group" aria-label="Zoom">
                  <span>Zoom</span>
                  <span className="web-tab-zoom-ctl">
                    <button type="button" onClick={() => zoomStep(-1)} aria-label="Zoom out" title="Zoom out">−</button>
                    <span className="web-tab-zoom-val">{Math.round(zoom * 100)}%</span>
                    <button type="button" onClick={() => zoomStep(1)} aria-label="Zoom in" title="Zoom in">+</button>
                    <button type="button" onClick={() => applyZoom(1)} aria-label="Reset zoom" title="Reset zoom"><IconReload /></button>
                  </span>
                </div>
                <DropdownMenu.Item className="um-item" onSelect={shot}><span className="um-item-name"><IconMonitor />Take a screenshot</span></DropdownMenu.Item>
                <DropdownMenu.Separator className="web-tab-menu-sep" />
                <DropdownMenu.Sub>
                  <DropdownMenu.SubTrigger className="um-item">
                    <span className="um-item-name">Passwords and autofill<IconChevronRight className="web-tab-sub-arrow" /></span>
                  </DropdownMenu.SubTrigger>
                  <DropdownMenu.Portal>
                    <DropdownMenu.SubContent className="web-tab-menu-list web-tab-submenu" sideOffset={4} alignOffset={-6} collisionPadding={8}>
                      <DropdownMenu.Item className="um-item" onSelect={() => openBrowserDialog("passwords")}>Password manager</DropdownMenu.Item>
                      <DropdownMenu.Item className="um-item" onSelect={() => openBrowserDialog("contact")}>Contact info</DropdownMenu.Item>
                    </DropdownMenu.SubContent>
                  </DropdownMenu.Portal>
                </DropdownMenu.Sub>
                <DropdownMenu.Item className="um-item" onSelect={() => openBrowserDialog("downloads")}>Downloads</DropdownMenu.Item>
                <DropdownMenu.Item className="um-item" onSelect={() => openBrowserDialog("history")}>History</DropdownMenu.Item>
                <DropdownMenu.Item className="um-item" onSelect={() => openBrowserDialog("wipe")}>Clear browsing data</DropdownMenu.Item>
                <DropdownMenu.Separator className="web-tab-menu-sep" />
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
      ) : null}
      {findOpen ? (
        <div className="web-tab-find" role="search" aria-label="Find in page">
          <input
            autoFocus
            value={findQuery}
            onChange={(e) => setFindQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") { e.preventDefault(); runFind(findQuery, !e.shiftKey); }
              if (e.key === "Escape") { e.preventDefault(); closeFind(); }
            }}
            placeholder="Find in page"
            aria-label="Find in page"
            spellCheck={false}
          />
          <span className="web-tab-find-count" aria-live="polite">{findState.count ? `${findState.active + 1}/${findState.count}` : "0/0"}</span>
          <button type="button" onClick={() => runFind(findQuery, false)} aria-label="Previous match" title="Previous (Shift+Enter)">↑</button>
          <button type="button" onClick={() => runFind(findQuery, true)} aria-label="Next match" title="Next (Enter)">↓</button>
          <button type="button" onClick={closeFind} aria-label="Close find bar" title="Close (Esc)"><IconX /></button>
        </div>
      ) : null}
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
      <div className="web-tab-host" ref={hostRef} hidden={!started} data-covered={covered ? "1" : undefined}>
        {still ? <img className="web-tab-still" src={still} alt="" aria-hidden="true" onError={() => { setStill(""); setPreviewFailed(true); }} /> : null}
      </div>
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
