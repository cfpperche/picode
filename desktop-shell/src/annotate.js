// annotate.js — injected into a work-browser tab while "Annotate this page" is
// on (v2c, the ChatGPT-Work shape: the page stays LIVE and the annotation UI
// lives *inside* it, because HTML can never paint over a WebView2 child).
//
// What it does in this slice: a hint chip, hover highlight, click to pick (an
// outline plus a pin), Esc to leave, and one `postMessage` per event. The
// comment box and the style editor ride on top of this contract later — the
// payload of a `pick` already carries what they will show.
//
// The script is idempotent: entering twice re-enters the same instance.
(() => {
  const KEY = "__picodeAnnotateV1";
  const prior = window[KEY];
  if (prior && typeof prior.enter === "function") {
    prior.enter();
    return;
  }

  const post = (msg) => {
    try {
      window.chrome?.webview?.postMessage(JSON.stringify(msg));
    } catch (e) {
      /* no channel: stay silent, the host will notice nothing arrived */
    }
  };

  const STYLE_PROPS = [
    "color",
    "background-color",
    "font-family",
    "font-size",
    "font-weight",
    "line-height",
    "padding",
    "margin",
    "border",
    "border-radius",
    "display",
    "width",
    "height",
  ];

  const readable = (el) => {
    let s = String(el.tagName || "").toLowerCase();
    if (el.id) s += "#" + el.id;
    const cls = typeof el.className === "string" ? el.className.trim().split(/\s+/) : [];
    if (cls.length && cls[0]) s += "." + cls.slice(0, 2).join(".");
    return s;
  };

  const computed = (el) => {
    const cs = getComputedStyle(el);
    const out = {};
    for (const p of STYLE_PROPS) out[p] = cs.getPropertyValue(p);
    return out;
  };

  const host = document.createElement("div");
  host.setAttribute("data-picode-annotate", "");
  host.style.cssText =
    "all: initial; position: fixed; inset: 0; z-index: 2147483647; pointer-events: none;";
  const root = host.attachShadow({ mode: "closed" });
  root.innerHTML = `
    <style>
      :host { all: initial; }
      .hl, .sel { position: fixed; border: 2px solid #4a9eff; border-radius: 3px; pointer-events: none; display: none; }
      .hl { background: rgba(74, 158, 255, 0.12); }
      .sel { background: rgba(74, 158, 255, 0.16); }
      .pin { position: fixed; width: 18px; height: 18px; border-radius: 50%; background: #4a9eff; color: #fff; font: 11px/18px system-ui, sans-serif; text-align: center; pointer-events: none; display: none; box-shadow: 0 1px 4px rgba(0,0,0,.35); }
      .chip { position: fixed; top: 12px; right: 12px; background: #1f6feb; color: #fff; font: 12px/1.6 system-ui, sans-serif; padding: 4px 10px; border-radius: 999px; box-shadow: 0 2px 8px rgba(0,0,0,.3); pointer-events: none; }
    </style>
    <div class="hl"></div>
    <div class="sel"></div>
    <div class="pin">1</div>
    <div class="chip">Annotating — click an element · Esc to exit</div>
  `;
  const hl = root.querySelector(".hl");
  const sel = root.querySelector(".sel");
  const pin = root.querySelector(".pin");

  let on = false;
  let picked = null;

  const box = (el, node) => {
    const r = el.getBoundingClientRect();
    node.style.left = r.left + "px";
    node.style.top = r.top + "px";
    node.style.width = Math.max(0, r.width) + "px";
    node.style.height = Math.max(0, r.height) + "px";
    node.style.display = "block";
  };

  const under = (e) => {
    if (e.target && (e.target === host || (e.target.getRootNode && e.target.getRootNode() === root))) return null;
    return document.elementFromPoint(e.clientX, e.clientY);
  };

  const onMove = (e) => {
    if (!on) return;
    const el = under(e);
    if (!el) {
      hl.style.display = "none";
      return;
    }
    box(el, hl);
  };

  const onClick = (e) => {
    if (!on) return;
    const el = under(e);
    if (!el) return;
    e.preventDefault();
    e.stopPropagation();
    picked = el;
    hl.style.display = "none";
    box(el, sel);
    const r = el.getBoundingClientRect();
    pin.style.left = Math.max(0, r.left - 9) + "px";
    pin.style.top = Math.max(0, r.top - 9) + "px";
    pin.style.display = "block";
    post({
      kind: "pick",
      selector: readable(el),
      tag: String(el.tagName || "").toLowerCase(),
      html: String(el.outerHTML || "").slice(0, 20000),
      rect: { x: r.x, y: r.y, width: r.width, height: r.height },
      styles: computed(el),
    });
  };

  const onKey = (e) => {
    if (!on) return;
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      exit();
    }
  };

  function enter() {
    if (on) return;
    on = true;
    if (!host.isConnected) document.documentElement.appendChild(host);
    document.addEventListener("mousemove", onMove, true);
    document.addEventListener("click", onClick, true);
    document.addEventListener("keydown", onKey, true);
    post({ kind: "enter", url: location.href });
  }

  function exit() {
    on = false;
    picked = null;
    document.removeEventListener("mousemove", onMove, true);
    document.removeEventListener("click", onClick, true);
    document.removeEventListener("keydown", onKey, true);
    hl.style.display = "none";
    sel.style.display = "none";
    pin.style.display = "none";
    host.remove();
    post({ kind: "exit" });
    try {
      delete window[KEY];
    } catch (e) {
      window[KEY] = null;
    }
  }

  window[KEY] = { enter, exit };
  enter();
})();
