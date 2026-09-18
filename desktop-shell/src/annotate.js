// annotate.js — injected into a work-browser tab while "Annotate this page" is
// on (v2c, the ChatGPT-Work shape: the page stays LIVE and the annotation UI
// lives *inside* it, because HTML can never paint over a WebView2 child).
//
// Slice 1: hint chip, hover highlight, click to pick (outline plus pin), Esc.
// Slice 2: a comment box anchored to the picked element, Send, and the box
// following the element when the page scrolls or resizes.
//
// Every event is one `postMessage`; the shell relays it and the chrome decides.
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
      /* no channel: stay silent, the host notices that nothing arrived */
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
      .box { position: fixed; display: none; pointer-events: auto; align-items: center; gap: 6px; background: #fff; color: #111; border: 1px solid rgba(0,0,0,.12); border-radius: 10px; box-shadow: 0 6px 22px rgba(0,0,0,.28); padding: 6px 8px; font: 13px/1.4 system-ui, sans-serif; max-width: min(520px, 90vw); }
      .box input { pointer-events: auto; border: 0; outline: 0; font: inherit; color: inherit; background: transparent; min-width: 220px; flex: 1; }
      .box button { pointer-events: auto; border: 0; border-radius: 7px; background: #1f6feb; color: #fff; font: 600 12px/1 system-ui, sans-serif; padding: 7px 10px; cursor: pointer; }
      .box button.ghost { background: rgba(0,0,0,.06); color: #333; }
    </style>
    <div class="hl"></div>
    <div class="sel"></div>
    <div class="pin">1</div>
    <div class="chip">Annotating — click an element · Esc to exit</div>
    <div class="box">
      <input type="text" placeholder="add a comment..." aria-label="Annotation comment" />
      <button class="save">Send</button>
      <button class="ghost cancel">Cancel</button>
    </div>
  `;
  const hl = root.querySelector(".hl");
  const sel = root.querySelector(".sel");
  const pin = root.querySelector(".pin");
  const box = root.querySelector(".box");
  const input = box.querySelector("input");
  const saveBtn = box.querySelector(".save");
  const cancelBtn = box.querySelector(".cancel");

  let on = false;
  let picked = null;

  const boxRect = (el, node) => {
    const r = el.getBoundingClientRect();
    node.style.left = r.left + "px";
    node.style.top = r.top + "px";
    node.style.width = Math.max(0, r.width) + "px";
    node.style.height = Math.max(0, r.height) + "px";
    node.style.display = "block";
    return r;
  };

  const placeBox = (el) => {
    const r = el.getBoundingClientRect();
    box.style.display = "flex";
    const h = box.offsetHeight || 40;
    const w = box.offsetWidth || 300;
    const top = r.bottom + 8 + h > window.innerHeight ? Math.max(4, r.top - h - 8) : r.bottom + 8;
    const left = Math.min(Math.max(4, r.left), Math.max(4, window.innerWidth - w - 4));
    box.style.top = top + "px";
    box.style.left = left + "px";
  };

  const draw = (el) => {
    boxRect(el, sel);
    const r = el.getBoundingClientRect();
    pin.style.left = Math.max(0, r.left - 9) + "px";
    pin.style.top = Math.max(0, r.top - 9) + "px";
    pin.style.display = "block";
    placeBox(el);
  };

  const reposition = () => {
    if (!on || !picked || !picked.isConnected) return;
    draw(picked);
  };

  const under = (e) => {
    const path = e.composedPath ? e.composedPath() : [];
    if (path.includes(host)) return null;
    return document.elementFromPoint(e.clientX, e.clientY);
  };

  const onMove = (e) => {
    if (!on || box.style.display !== "none") return;
    const el = under(e);
    if (!el) {
      hl.style.display = "none";
      return;
    }
    boxRect(el, hl);
  };

  const payload = (el) => {
    const r = el.getBoundingClientRect();
    return {
      selector: readable(el),
      tag: String(el.tagName || "").toLowerCase(),
      html: String(el.outerHTML || "").slice(0, 20000),
      rect: { x: r.x, y: r.y, width: r.width, height: r.height },
      styles: computed(el),
      vw: window.innerWidth,
      vh: window.innerHeight,
    };
  };

  const pick = (e) => {
    if (!on) return;
    const el = under(e);
    if (!el) return;
    e.preventDefault();
    e.stopPropagation();
    picked = el;
    hl.style.display = "none";
    draw(el);
    input.value = "";
    input.focus();
    post(Object.assign({ kind: "pick" }, payload(el)));
  };

  const send = () => {
    if (!on || !picked) return;
    const comment = String(input.value || "").trim();
    post(Object.assign({ kind: "comment", comment }, payload(picked)));
    box.style.display = "none";
    input.blur();
  };

  const closeBox = () => {
    box.style.display = "none";
    input.blur();
  };

  const onKey = (e) => {
    if (!on) return;
    if (e.key === "Enter" && box.style.display !== "none" && e.target === input) {
      e.preventDefault();
      e.stopPropagation();
      send();
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      if (box.style.display !== "none") closeBox();
      else exit();
    }
  };

  saveBtn.addEventListener("click", (e) => { e.preventDefault(); e.stopPropagation(); send(); });
  cancelBtn.addEventListener("click", (e) => { e.preventDefault(); e.stopPropagation(); closeBox(); });
  box.addEventListener("click", (e) => e.stopPropagation());

  function enter() {
    if (on) return;
    on = true;
    if (!host.isConnected) document.documentElement.appendChild(host);
    document.addEventListener("mousemove", onMove, true);
    document.addEventListener("click", pick, true);
    document.addEventListener("keydown", onKey, true);
    window.addEventListener("scroll", reposition, true);
    window.addEventListener("resize", reposition, true);
    post({ kind: "enter", url: location.href });
  }

  function exit() {
    on = false;
    picked = null;
    document.removeEventListener("mousemove", onMove, true);
    document.removeEventListener("click", pick, true);
    document.removeEventListener("keydown", onKey, true);
    window.removeEventListener("scroll", reposition, true);
    window.removeEventListener("resize", reposition, true);
    hl.style.display = "none";
    sel.style.display = "none";
    pin.style.display = "none";
    closeBox();
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
