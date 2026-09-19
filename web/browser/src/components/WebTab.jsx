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
import { cropRect, stylesToCSS, stateItems, batchMessage, parseAnnotMessage, parseStatePayload, resolveSendTarget } from "../lib/annotate.js";
import AnnotateStrip from "./AnnotateStrip.jsx";
import AnnotatePreview from "./AnnotatePreview.jsx";

// Work browser tab surface (Phase 3 slice 1): the React side renders the
// toolbar and an empty region; the actual page is a native WebView2 child
// positioned over that region (bounds pushed on every resize). Desktop
// shell only — a web page cannot host a WebView2, so /browser/ never shows
// this surface.
const invoke = typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

export default function WebTabSurface({ tabId, url = "", active, hidden, className = "", expanded = false, chromeless = false, asks = [], onAnswerAsk, onClose, onMeta, onNew, onBrowserSettings, onToggleExpand, boundSession = "" }) {
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

  // ---- annotate mode (v2c): the page stays LIVE; the UI lives inside it ----
  // Nothing is drawn over the native view: the shell injects a script into the
  // page (shadow DOM), and the page talks back on WebView2's message channel.
  // Annotations accumulate as numbered pins; the strip's Send ships the whole
  // set as ONE context through the prompt door (ADR-0152: one row per pin,
  // one message carrying every path). Nothing leaves the page until Send.
  const [annotOn, setAnnotOn] = useState(false);
  const [annotItems, setAnnotItems] = useState([]);
  const [annotShots, setAnnotShots] = useState(true);
  const [annotSending, setAnnotSending] = useState(false);
  const annotOnRef = useRef(false);
  // Did this tab's page answer the arm? Any message (enter/state/pick) proves
  // the channel; the arm's watchdog reads it once.
  const annotSeenRef = useRef(false);
  const annotItemsRef = useRef([]);
  annotOnRef.current = annotOn;
  annotItemsRef.current = annotItems;
  // The page's current URL, mirrored by the meta poll below: annotations are
  // filed under what the tab shows, not what it was opened with. (This ref
  // was missing once — every Send died on a ReferenceError.)
  const liveUrlRef = useRef("");
  // Leaving clears both sides at once: the page drops its overlay (or is
  // already gone) and the chrome drops the count. Idempotent, so the strip's
  // ✕ and the page's own Esc converge instead of double-announcing.
  const leaveAnnotate = (announce) => {
    if (!annotOnRef.current) return;
    const pending = annotItemsRef.current.length;
    annotOnRef.current = false;
    setAnnotOn(false);
    setAnnotItems([]);
    setAnnotSending(false);
    if (announce && pending > 0) toast(`Annotate mode off — ${pending} unsent annotation${pending === 1 ? "" : "s"} discarded.`);
  };
  const exitAnnotate = () => {
    if (invoke) invoke("btab_annotate_mode", { id: tabId, on: false }).catch(() => {});
    leaveAnnotate(true);
  };
  const clearAnnotate = () => {
    const n = annotItemsRef.current.length;
    if (invoke) invoke("btab_annotate_clear", { id: tabId }).catch((e) => toast("Discard failed: " + (e?.message || e)));
    setAnnotItems([]);
    if (n > 0) toast.ok(`All ${n} annotation${n === 1 ? "" : "s"} discarded.`);
  };
  const undoAnnotate = () => {
    if (!annotItemsRef.current.length || !invoke) return;
    invoke("btab_annotate_clear", { id: tabId, last: true }).catch((e) => toast("Undo failed: " + (e?.message || e)));
  };
  useEffect(() => {
    const listen = typeof window !== "undefined" ? window.__TAURI__?.event?.listen : null;
    if (typeof listen !== "function") return undefined;
    let unlisten = null;
    listen("btab://annotate", (event) => {
      const parsed = parseAnnotMessage(event?.payload);
      if (!parsed || parsed.id !== tabId) return;
      annotSeenRef.current = true;
      const inner = parsed.inner;
      if (inner.kind === "pick") toast.ok("Picked " + (inner.selector || inner.tag || "element"));
      else if (inner.kind === "state") setAnnotItems(stateItems(inner));
      else if (inner.kind === "note") toast.ok(String(inner.message || ""));
      else if (inner.kind === "warn") toast(String(inner.message || ""));
      else if (inner.kind === "exit") leaveAnnotate(true);
      // saved / removed / enter ride the state sync that follows them.
    }).then((u) => { unlisten = u; }).catch(() => {});
    return () => { try { if (unlisten) unlisten(); } catch { /* already gone */ } };
  }, [id]);
  // The crop comes from a fresh viewport capture, cut with the element's rect
  // scaled from CSS pixels to the capture's own size (the page reports vw/vh).
  const cropFromPreview = async (msg) => {
    if (!invoke || !msg || !msg.vw || !msg.vh || !msg.rect) return "";
    try {
      // The native tab id, not the React tab id: this surface's other native
      // calls (bounds, zoom, find) all pass the tail, and the crop came back
      // empty for every Send until this matched (2026-09-19 — the shell
      // resolves either shape now, but the call sites stay consistent).
      const bytes = await invoke("btab_preview", { id });
      const url = await previewUrl(bytes);
      if (!url) return "";
      const img = await new Promise((resolve, reject) => {
        const i = new Image();
        i.onload = () => resolve(i);
        i.onerror = reject;
        i.src = url;
      });
      const rect = cropRect({
        rect: msg.rect,
        naturalWidth: img.naturalWidth,
        naturalHeight: img.naturalHeight,
        displayWidth: msg.vw,
        displayHeight: msg.vh,
      });
      if (!rect) { URL.revokeObjectURL(url); return ""; }
      const canvas = document.createElement("canvas");
      canvas.width = rect.width;
      canvas.height = rect.height;
      const ctx = canvas.getContext("2d");
      if (!ctx) { URL.revokeObjectURL(url); return ""; }
      ctx.drawImage(img, rect.x, rect.y, rect.width, rect.height, 0, 0, rect.width, rect.height);
      const data = canvas.toDataURL("image/png");
      URL.revokeObjectURL(url);
      return data.startsWith("data:image/png;base64,") ? data.slice("data:image/png;base64,".length) : "";
    } catch {
      return "";
    }
  };

  // Send ships the whole set as one package (ADR-0152): one row per pin in
  // the store, then ONE message through the prompt door carrying every path
  // — the reference's "N annotations" context. An open draft blocks the
  // Send (its note is unfinished); with no agent terminal running nothing is
  // staged and nothing is lost.
  const sendAll = async () => {
    const items = annotItemsRef.current;
    const ready = items.filter((it) => it.saved);
    if (items.length > ready.length) { toast("Finish the open note first — Save or Cancel it."); return; }
    if (ready.length === 0) { toast("Pin a note first — click any element."); return; }
    setAnnotSending(true);
    try {
      const list = await fetch("/api/terminals").then((r) => (r.ok ? r.json() : null)).catch(() => null);
      const terminals = Array.isArray(list) ? list : list?.terminals || [];
      // A pane open on a session delivers to THAT session (owner
      // 2026-09-18) — a bound miss never falls through to a stranger's
      // terminal; a standalone tab keeps the first-running fallback.
      const target = resolveSendTarget({ boundSession, terminals });
      if (target.none) { toast(target.reason); return; }
      const live = { id: target.kind === "agent" ? target.terminalId : target.id, name: target.name };
      const prefs = await fetch("/api/browser/prefs").then((r) => (r.ok ? r.json() : null)).catch(() => null);
      const policy = prefs && (prefs.annotationShots === "always" || prefs.annotationShots === "never") ? prefs.annotationShots : "ask";
      // "never" means no picture at all; "always" always crops; "ask"
      // follows the strip's camera toggle for this session.
      const includeShots = policy === "never" ? false : policy === "always" ? true : annotShots;
      const paths = [];
      const staged = [];
      for (const it of ready) {
        const image = includeShots ? await cropFromPreview(it) : "";
        const res = await fetch("/api/browser/annotations", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            terminalId: live.id,
            url: liveUrlRef.current || url,
            title: "",
            selector: it.selector || "",
            comment: it.comment || "",
            dom: it.html || "",
            css: stylesToCSS(it.styles || {}),
            image,
          }),
        });
        if (!res.ok) throw new Error(`annotation ${it.n} was not saved (${res.status})`);
        const body = await res.json().catch(() => ({}));
        for (const p of Array.isArray(body.paths) ? body.paths : []) paths.push(p);
        staged.push({ n: it.n, selector: it.selector, comment: it.comment });
      }
      const message = batchMessage({ url: liveUrlRef.current || url, items: staged });
      // Delivery follows the target: an agent-bound pane pastes through the
      // agent's own prompt door (its chat), a terminal-bound one through the
      // terminal's. Staging always rides the terminal row, so the files land
      // where the prompt door resolves its paths.
      const promptBase = target.kind === "agent"
        ? `/api/agents/${encodeURIComponent(target.agentId)}`
        : `/api/terminals/${encodeURIComponent(live.id)}`;
      const prompt = await fetch(`${promptBase}/prompt`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message, paths }),
      }).catch(() => null);
      if (prompt && prompt.ok) {
        toast.ok(`${ready.length} annotation${ready.length === 1 ? "" : "s"} sent to ${live.name || live.id}.`);
        if (invoke) invoke("btab_annotate_clear", { id: tabId }).catch(() => {});
        setAnnotItems([]);
      } else {
        const failure = await prompt?.json().catch(() => null);
        const reason = failure && typeof failure.error === "string" && failure.error.trim()
          ? failure.error.trim()
          : "the terminal did not accept it";
        toast(`Annotations saved, but not sent: ${reason}.`);
      }
    } catch (e) {
      toast("Annotation failed: " + (e?.message || e));
    } finally {
      setAnnotSending(false);
    }
  };

  // The pull door: while the mode is on, ask the shell for the page's state
  // on a slow tick. The push channel (the page's own postMessage) is the
  // fast path; this one needs no page-side bridge at all and is what makes
  // Send light up in a document whose bridge is dead (owner 2026-09-19: the
  // card and the chips worked while every message vanished). A poll is cheap
  // — ExecuteScript in the same process — and it also repairs a state the
  // push path dropped.
  useEffect(() => {
    if (!invoke || !annotOn || hidden) return undefined;
    let stop = false;
    const tick = () => {
      invoke("btab_annotate_state", { id: tabId })
        .then((raw) => {
          const state = parseStatePayload(raw);
          if (stop || !state) return;
          annotSeenRef.current = true;
          if (state.kind === "off") { leaveAnnotate(true); return; }
          setAnnotItems(stateItems(state));
        })
        .catch(() => {});
    };
    tick();
    const t = setInterval(tick, 1500);
    return () => {
      stop = true;
      clearInterval(t);
    };
  }, [annotOn, hidden, tabId]);

  // Arm the mode, and make silence visible. The page answers an arm with its
  // `enter` (+ the state it holds), so no answer means the channel is dead:
  // the card and the chips keep working (they are the page's own DOM) while
  // Send never lights up — the failure the owner met with no signal at all
  // (2026-09-19). Re-arming re-subscribes in the shell and the page keeps its
  // pins, so one silent retry is the repair; a second silence is the truth,
  // said out loud.
  const armAnnotate = (retriesLeft = 1) => {
    invoke("btab_annotate_mode", { id: tabId, on: true })
      .then(() => {
        setTimeout(() => {
          if (!annotOnRef.current || annotSeenRef.current) return;
          if (retriesLeft > 0) { armAnnotate(retriesLeft - 1); return; }
          toast("Annotate mode: this page is not answering — annotations will not reach the strip.");
        }, 2500);
      })
      .catch((e) => {
        annotOnRef.current = false;
        setAnnotOn(false);
        toast("Annotate mode failed: " + (e?.message || e));
      });
  };

  const toggleAnnotate = () => {
    if (annotOnRef.current) { exitAnnotate(); return; }
    // The plain browser has no live page to inject into: the preview shows
    // the mode's shape over a sample page instead.
    if (!invoke) {
      annotOnRef.current = true;
      setAnnotOn(true);
      return;
    }
    annotOnRef.current = true;
    annotSeenRef.current = false;
    setAnnotOn(true);
    armAnnotate();
  };
  useEffect(() => {
    const onKey = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === ".") { e.preventDefault(); toggleAnnotate(); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });


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
            liveUrlRef.current = m.url;
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
    // Navigating drops the document the script was injected into: leaving
    // first keeps the chrome honest instead of counting pins on a dead page.
    if (annotOnRef.current) exitAnnotate();
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

  // One guard for the buttons that drive the page (Back/Forward/Reload
  // call it inline); go() above calls exitAnnotate directly.
  const navGuard = () => { if (annotOnRef.current) exitAnnotate(); };

  if (!invoke) {
    const typed = (frameUrl || "").trim();
    if (annotOn) {
      return (
        <section className="web-tab-surface" hidden={hidden} aria-label="Work browser">
          <AnnotatePreview onExit={toggleAnnotate} />
          {onClose ? (
            <div className="web-tab-toolbar">
              <button type="button" className="web-tab-menu" title="Close browser pane" aria-label="Close browser pane" onClick={onClose}><IconX /></button>
            </div>
          ) : null}
        </section>
      );
    }
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
          <button
            type="button"
            className="web-tab-annot-btn"
            title="Preview annotate mode (Ctrl+.)"
            aria-label="Preview annotate mode"
            aria-pressed={false}
            onClick={toggleAnnotate}
          >✎</button>
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
          <button type="button" title="Back" onClick={() => { navGuard(); invoke("btab_back", { id }).catch(() => {}); }}>←</button>
          <button type="button" title="Forward" onClick={() => { navGuard(); invoke("btab_forward", { id }).catch(() => {}); }}>→</button>
          <button type="button" title="Reload" onClick={() => { navGuard(); invoke("btab_reload", { id }).catch(() => {}); }}>⟳</button>
          {annotOn ? (
            <AnnotateStrip
              count={annotItems.filter((it) => it.saved).length}
              total={annotItems.length}
              shotsOn={annotShots}
              sending={annotSending}
              onExit={exitAnnotate}
              onClear={clearAnnotate}
              onUndo={undoAnnotate}
              onToggleShots={() => setAnnotShots((v) => !v)}
              onHint={() => toast.ok("Click an element to pin it · Enter saves · Esc exits.")}
              onSend={sendAll}
            />
          ) : (
          <><div className="web-tab-urlbar">
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
          className="web-tab-annot-btn"
          title="Annotate this page (Ctrl+.)"
          aria-label="Annotate this page"
          aria-pressed={false}
          onClick={toggleAnnotate}
        >✎</button></>
          )}
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
