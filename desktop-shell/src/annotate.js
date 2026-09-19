// annotate.js — injected into a work-browser tab while "Annotate this page" is
// on (v2c, the ChatGPT-Work shape: the page stays LIVE and the annotation UI
// lives *inside* it, because HTML can never paint over a WebView2 child).
//
// The reference (owner screenshots 2026-09-18): annotations accumulate, each
// with a numbered pin; the picked one is outlined; an anchored card edits one
// note; a saved note collapses to a chip; nothing leaves the page until the
// chrome strip's Send ships the whole set as one package.
//
// Card controls decision table:
// | card state    | Cancel                        | Save                     | trash (this one) |
// |---------------|-------------------------------|--------------------------|------------------|
// | new draft     | removes the pin, closes       | keeps it, collapses chip | removes, closes  |
// | editing saved | closes, keeps the old text    | updates the chip text    | removes, closes  |
//
// While a card is open every other open path is LOCKED (page clicks create
// nothing, pins/chips/menus do not reopen): a stray click off the card
// used to steal it onto a new pin and evaporate the unsaved draft (owner
// 2026-09-18). The strip already teaches this ("Save the open note").
//
// Page → chrome messages (one postMessage each; the shell relays them):
//   enter {url} · pick {n, …payload} · saved {n, comment, …payload} ·
//   removed {n} · state {url, count, items} · note {message} · warn {message} ·
//   exit {}. The chrome owns the Send count and the batch payload from the
//   last state; the page owns the pins, the card and the chips.
// Chrome → page: window.__picodeAnnotateV1.clear() (the strip's trash) and
// .exit() (leaving the mode). The script is idempotent: entering twice
// re-enters the same instance and keeps its items.
(() => {
  const KEY = "__picodeAnnotateV1";
  const prior = window[KEY];
  if (prior && typeof prior.enter === "function") {
    prior.enter();
    return;
  }

  // The message goes as an OBJECT, never pre-stringified: the host reads it
  // through WebMessageAsJson, which returns the value's JSON serialization —
  // deterministic for an object. A pre-stringified text risks coming back
  // JSON-encoded a second time (a quoted string), which the chrome would
  // parse into a string with no kind and drop silently: saved chip in the
  // page, Send 0 in the strip (owner 2026-09-18).
  const post = (msg) => {
    try {
      window.chrome?.webview?.postMessage(msg);
    } catch (e) {
      /* no channel: stay silent, the host notices that nothing arrived */
    }
  };

  const STYLE_PROPS = [
    "color",
    "background-color",
    "opacity",
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

  // ---- The style inspector (v2c step 5) --------------------------------
  // The card offers the element's own computed styles (the reference's list:
  // text colour, background, opacity, font family · size · weight) and every
  // change is a PREVIEW on the live page, so the human sees the result before
  // sending it. What travels to the agent is original → proposed — the
  // mutated value alone would read as "the page looks like this", which is
  // the one thing it does not. Cancel, the trash and leaving the mode put the
  // element back exactly as the page had it.
  const INSPECT = [
    { key: "color", label: "Text color", kind: "color" },
    { key: "background-color", label: "Background", kind: "color" },
    { key: "opacity", label: "Opacity", kind: "range" },
    { key: "font-family", label: "Font", kind: "font" },
    { key: "font-size", label: "Size", kind: "number" },
    { key: "font-weight", label: "Weight", kind: "weight" },
  ];
  const FONT_STACKS = [
    { label: "System", value: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" },
    { label: "Serif", value: "Georgia, 'Times New Roman', serif" },
    { label: "Monospace", value: "ui-monospace, Consolas, 'Courier New', monospace" },
  ];
  const WEIGHTS = [[300, "Light"], [400, "Regular"], [500, "Medium"], [600, "Semibold"], [700, "Bold"]];
  const SIZE_MIN = 8;
  const SIZE_MAX = 96;

  const controlHTML = (d) => {
    if (d.kind === "color") {
      return `<input type="color" data-p="${d.key}" data-k="pick" aria-label="${d.label}">` +
        `<input type="text" data-p="${d.key}" data-k="text" spellcheck="false" aria-label="${d.label} value">`;
    }
    if (d.kind === "range") {
      return `<input type="range" data-p="${d.key}" data-k="range" min="0" max="1" step="0.05" aria-label="${d.label}">` +
        `<span class="txt" data-p="${d.key}" data-k="read"></span>`;
    }
    if (d.kind === "number") {
      return `<input type="number" data-p="${d.key}" data-k="num" min="${SIZE_MIN}" max="${SIZE_MAX}" step="1" aria-label="${d.label}">` +
        `<span class="txt">px</span>`;
    }
    if (d.kind === "font") {
      return `<select data-p="${d.key}" data-k="select" aria-label="${d.label}"><option value="">As on the page</option>` +
        FONT_STACKS.map((f) => `<option value="${f.value}">${f.label}</option>`).join("") + `</select>`;
    }
    return `<select data-p="${d.key}" data-k="select" aria-label="${d.label}"><option value="">As on the page</option>` +
      WEIGHTS.map(([v, l]) => `<option value="${v}">${l} ${v}</option>`).join("") + `</select>`;
  };

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
      .pin { position: fixed; min-width: 18px; height: 18px; padding: 0 2px; border-radius: 999px; background: #4a9eff; color: #fff; font: 600 11px/18px system-ui, sans-serif; text-align: center; pointer-events: auto; cursor: pointer; display: none; box-shadow: 0 1px 4px rgba(0,0,0,.35); box-sizing: border-box; }
      .pin.on { outline: 2px solid #fff; box-shadow: 0 0 0 4px rgba(74,158,255,.55), 0 1px 4px rgba(0,0,0,.35); }
      .card { position: fixed; display: none; pointer-events: auto; background: #fff; color: #111; border: 1px solid rgba(0,0,0,.12); border-radius: 10px; box-shadow: 0 6px 22px rgba(0,0,0,.28); padding: 8px; font: 13px/1.4 system-ui, sans-serif; width: 320px; max-width: calc(100vw - 8px); box-sizing: border-box; }
      .card .row { display: flex; align-items: center; gap: 6px; }
      .card .mark { flex: none; width: 22px; height: 22px; border-radius: 6px; background: rgba(74,158,255,.14); color: #1f6feb; font: 600 13px/22px system-ui, sans-serif; text-align: center; }
      .card input { border: 0; outline: 0; font: inherit; color: inherit; background: transparent; min-width: 0; flex: 1; }
      .card .icon { flex: none; border: 0; background: transparent; color: #666; font: 13px/1 system-ui, sans-serif; padding: 5px; border-radius: 6px; cursor: pointer; }
      .card .icon:hover { background: rgba(0,0,0,.06); color: #111; }
      .card .actions { display: flex; justify-content: flex-end; gap: 6px; margin-top: 8px; }
      .card .actions button { border: 0; border-radius: 7px; font: 600 12px/1 system-ui, sans-serif; padding: 8px 12px; cursor: pointer; }
      .card .save { background: #1f6feb; color: #fff; }
      .card .cancel { background: rgba(0,0,0,.06); color: #333; }
      .card .styles { margin-top: 8px; border-top: 1px solid rgba(0,0,0,.08); padding-top: 6px; }
      .card .styles > summary { display: flex; align-items: center; gap: 6px; cursor: pointer; list-style: none; padding: 2px 0; font-weight: 600; color: #333; }
      .card .styles > summary::-webkit-details-marker { display: none; }
      .card .styles .swatch { flex: none; width: 12px; height: 12px; border-radius: 3px; border: 1px solid rgba(0,0,0,.25); background: #fff; }
      .card .styles .scount { color: #1f6feb; font-weight: 600; }
      .card .styles .chev { margin-left: auto; color: #888; font-weight: 400; }
      .card .styles[open] .chev { transform: rotate(180deg); }
      .card .sgrid { display: grid; grid-template-columns: 64px minmax(0, 1fr); gap: 5px 6px; align-items: center; margin-top: 7px; }
      .card .slab { color: #666; font-size: 11px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
      .card .sctl { display: flex; align-items: center; gap: 4px; min-width: 0; }
      .card .sctl > input, .card .sctl > select { height: 22px; min-width: 0; box-sizing: border-box; border: 1px solid rgba(0,0,0,.16); border-radius: 5px; background: #fff; color: #111; font: 11px/1.2 system-ui, sans-serif; }
      .card .sctl input[type="color"] { flex: none; width: 24px; padding: 0; cursor: pointer; }
      .card .sctl input[type="text"] { flex: 1; padding: 0 4px; }
      .card .sctl input[type="number"] { flex: 1; padding: 0 2px 0 4px; }
      .card .sctl input[type="range"] { flex: 1; border: 0; background: transparent; padding: 0; }
      .card .sctl select { flex: 1; padding: 0 2px; }
      .card .sctl .txt { flex: none; color: #666; font-size: 11px; }
      .card .sctl .bad { border-color: #d1242f; box-shadow: 0 0 0 2px rgba(209,36,47,.15); }
      .card .sreset { flex: none; border: 0; background: transparent; color: #888; font: 12px/1 system-ui, sans-serif; padding: 3px; border-radius: 5px; cursor: pointer; visibility: hidden; }
      .card .sreset:hover { background: rgba(0,0,0,.06); color: #111; }
      .anchip .aa { display: none; margin-right: 4px; padding: 0 3px; border-radius: 3px; background: rgba(31,111,235,.12); color: #1f6feb; font: 600 10px/14px system-ui, sans-serif; }
      .anchip.styled .aa { display: inline-block; }
      .anchip { position: fixed; display: none; pointer-events: auto; align-items: center; gap: 2px; background: #fff; color: #111; border: 1px solid rgba(0,0,0,.12); border-radius: 999px; box-shadow: 0 2px 10px rgba(0,0,0,.22); padding: 3px 4px 3px 8px; font: 12px/1.5 system-ui, sans-serif; max-width: 240px; box-sizing: border-box; }
      .anchip .txt { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .anchip button { flex: none; border: 0; background: transparent; color: #666; font: 12px/1 system-ui, sans-serif; padding: 4px 5px; border-radius: 6px; cursor: pointer; }
      .anchip button:hover { background: rgba(0,0,0,.07); color: #111; }
      .menu { position: fixed; display: none; pointer-events: auto; background: #fff; color: #111; border: 1px solid rgba(0,0,0,.12); border-radius: 8px; box-shadow: 0 6px 22px rgba(0,0,0,.28); padding: 4px; font: 13px/1.4 system-ui, sans-serif; min-width: 130px; box-sizing: border-box; }
      .menu button { display: block; width: 100%; text-align: left; border: 0; background: transparent; color: inherit; font: inherit; padding: 7px 10px; border-radius: 6px; cursor: pointer; }
      .menu button:hover { background: rgba(0,0,0,.06); }
      .menu button.danger { color: #c22; }
    </style>
    <div class="hl"></div>
    <div class="sel"></div>
    <div class="card" role="dialog" aria-label="Annotation">
      <div class="row">
        <span class="mark" aria-hidden="true">✎</span>
        <input type="text" placeholder="add a comment..." aria-label="Annotation comment" />
        <button class="icon trash" title="Discard this annotation" aria-label="Discard this annotation">🗑</button>
        <button class="icon mic" title="Dictate the note" aria-label="Dictate the note">🎙</button>
      </div>
      <details class="styles">
        <summary>
          <span class="swatch" aria-hidden="true"></span>
          <span>Styles</span>
          <span class="scount"></span>
          <span class="chev" aria-hidden="true">▾</span>
        </summary>
        <div class="sgrid">
          ${INSPECT.map((d) => `<span class="slab" title="${d.label}">${d.label}</span><span class="sctl">${controlHTML(d)}<button class="sreset" data-p="${d.key}" title="Back to the page's own ${d.label.toLowerCase()}" aria-label="Reset ${d.label}">↺</button></span>`).join("")}
        </div>
      </details>
      <div class="actions">
        <button class="cancel">Cancel</button>
        <button class="save">Save</button>
      </div>
    </div>
    <div class="menu" role="menu">
      <button data-act="edit" role="menuitem">Edit</button>
      <button data-act="copy" role="menuitem">Copy text</button>
      <button data-act="remove" role="menuitem" class="danger">Remove</button>
    </div>
  `;
  const hl = root.querySelector(".hl");
  const sel = root.querySelector(".sel");
  const card = root.querySelector(".card");
  const input = card.querySelector("input");
  const trashBtn = card.querySelector(".trash");
  const micBtn = card.querySelector(".mic");
  const saveBtn = card.querySelector(".save");
  const cancelBtn = card.querySelector(".cancel");
  const stylesBox = card.querySelector(".styles");
  const styleSwatch = card.querySelector(".styles .swatch");
  const styleCount = card.querySelector(".styles .scount");
  const menu = root.querySelector(".menu");

  // The inspector's own helpers. `styles0` is the computed snapshot taken at
  // pick time — reading it live would return the preview's own mutation and
  // turn "what the page had" into "what we just set". `inline0` remembers the
  // element's own inline values for those six properties, so putting a page
  // back never deletes a style the page itself wrote.
  const toHex = (v) => {
    const s = String(v || "").trim();
    const m = s.match(/rgba?\(([^)]+)\)/i);
    if (!m) return /^#[0-9a-f]{6}$/i.test(s) ? s : "#ffffff";
    const parts = m[1].split(/[,\s\/]+/).filter((x) => x !== "").map(Number);
    const [r, g, b, a] = parts;
    if (parts.length > 3 && a === 0) return "#ffffff"; // transparent reads as the card's own white
    const hex = [r, g, b]
      .map((n) => Math.max(0, Math.min(255, Math.round(Number.isFinite(n) ? n : 0))).toString(16).padStart(2, "0"))
      .join("");
    return "#" + hex;
  };

  const sameStyle = (a) => String(a || "").replace(/\s+/g, "").replace(/'/g, "\"").toLowerCase();

  const inline0Of = (el) => {
    const out = {};
    for (const d of INSPECT) {
      out[d.key] = [el.style.getPropertyValue(d.key), el.style.getPropertyPriority(d.key)];
    }
    return out;
  };

  // backToOne puts one property back the way the page had it: the element's
  // own inline value if it had one, the page's rule otherwise.
  const backToOne = (it, prop) => {
    const own = (it.inline0 && it.inline0[prop]) || ["", ""];
    if (own[0]) it.el.style.setProperty(prop, own[0], own[1]);
    else it.el.style.removeProperty(prop);
  };

  // restoreStyles drops what THIS annotation proposed. Two pins can sit on
  // one element, so a property the other one still proposes is re-applied
  // instead of being wiped with ours.
  const restoreStyles = (it) => {
    if (!it || !it.edits) return;
    const props = Object.keys(it.edits);
    it.edits = {};
    for (const p of props) {
      const other = items.find((x) => x !== it && x.el === it.el && x.edits && x.edits[p]);
      if (other) it.el.style.setProperty(p, other.edits[p].to, "important");
      else backToOne(it, p);
    }
  };

  const restoreAllStyles = () => {
    items.forEach((it) => restoreStyles(it));
  };

  const applyEdit = (it, prop, value) => {
    const original = (it.styles0 && it.styles0[prop]) || "";
    const next = String(value == null ? "" : value).trim();
    if (!next || sameStyle(next) === sameStyle(original)) {
      delete it.edits[prop];
      backToOne(it, prop);
    } else {
      it.edits[prop] = { from: original, to: next };
      // !important: a page rule that is itself !important would swallow the
      // preview and the human would watch nothing happen.
      it.el.style.setProperty(prop, next, "important");
    }
    renderStyles(it);
  };

  const matchOption = (prop, value) => {
    const sel = stylesBox.querySelector(`select[data-p="${prop}"]`);
    if (!sel) return false;
    return Array.from(sel.options).some((o) => o.value && sameStyle(o.value) === sameStyle(value));
  };

  // renderStyles paints every control from the item's truth (edits first, the
  // page's own value otherwise). The DOM never decides what is proposed.
  const renderStyles = (it) => {
    const cur = (p) => (it.edits && it.edits[p] ? it.edits[p].to : (it.styles0 && it.styles0[p]) || "");
    for (const d of INSPECT) {
      const p = d.key;
      const v = cur(p);
      const pick = stylesBox.querySelector(`[data-p="${p}"][data-k="pick"]`);
      const text = stylesBox.querySelector(`[data-p="${p}"][data-k="text"]`);
      const range = stylesBox.querySelector(`[data-p="${p}"][data-k="range"]`);
      const read = stylesBox.querySelector(`[data-p="${p}"][data-k="read"]`);
      const num = stylesBox.querySelector(`[data-p="${p}"][data-k="num"]`);
      const sel = stylesBox.querySelector(`select[data-p="${p}"]`);
      const reset = stylesBox.querySelector(`.sreset[data-p="${p}"]`);
      if (pick) pick.value = toHex(v);
      if (text) { text.value = v; text.classList.remove("bad"); }
      if (range) range.value = String(Math.max(0, Math.min(1, parseFloat(v) || 0)));
      if (read) read.textContent = Math.round((parseFloat(v) || 0) * 100) + "%";
      if (num) num.value = String(Math.round(parseFloat(v) || 0));
      if (sel) sel.value = matchOption(p, v) ? v : "";
      if (reset) reset.style.visibility = it.edits && it.edits[p] ? "visible" : "hidden";
    }
    const n = Object.keys(it.edits || {}).length;
    styleCount.textContent = n ? n + (n === 1 ? " change" : " changes") : "";
    styleSwatch.style.background = cur("color") || "#fff";
    markChip(it);
  };

  const markChip = (it) => {
    if (!it || !it.chip) return;
    it.chip.classList.toggle("styled", Object.keys(it.edits || {}).length > 0);
  };

  const announce = (it) => {
    post(Object.assign({ kind: "styled" }, payload(it)));
  };

  const onStyleEvent = (e) => {
    const it = editing;
    if (!on || !it) return;
    const t = e.target;
    if (!t || !t.getAttribute) return;
    const prop = t.getAttribute("data-p");
    const kind = t.getAttribute("data-k");
    if (!prop || !kind || kind === "read") return;
    // Text, number and the selects speak on change (Enter/blur), the colour
    // picker and the slider live while dragging: on `input` only those two.
    const isChange = e.type === "change";
    if (kind !== "pick" && kind !== "range" && !isChange) return;
    if (kind === "pick" || kind === "range") {
      e.stopPropagation();
      applyEdit(it, prop, t.value);
      announce(it);
      return;
    }
    if (kind === "num") {
      const n = Math.round(parseFloat(t.value) || 0);
      if (!(n >= SIZE_MIN && n <= SIZE_MAX)) {
        post({ kind: "warn", message: "A font size between " + SIZE_MIN + " and " + SIZE_MAX + " px, please." });
        renderStyles(it);
        return;
      }
      applyEdit(it, prop, n + "px");
      announce(it);
      return;
    }
    if (kind === "text") {
      const v = String(t.value || "").trim();
      if (v && !(window.CSS && CSS.supports && CSS.supports("color", v))) {
        // Say it instead of silently keeping the old colour: an ignored edit
        // looks exactly like a page that refuses to change.
        t.classList.add("bad");
        post({ kind: "warn", message: '"' + v + '" is not a colour this page accepts.' });
        return;
      }
      applyEdit(it, prop, v);
      announce(it);
      return;
    }
    if (kind === "select") {
      applyEdit(it, prop, String(t.value || "").trim());
      announce(it);
    }
  };

  // A new card opens the section the last one left open: the human who works
  // with styles keeps them in view, the one who never touches them keeps a
  // small card (the disclosure is remembered for the session).
  let stylesOpen = false;
  stylesBox.addEventListener("toggle", () => { stylesOpen = stylesBox.open; });
  stylesBox.addEventListener("input", onStyleEvent);
  stylesBox.addEventListener("change", onStyleEvent);
  stylesBox.addEventListener("click", (e) => {
    const btn = e.target && e.target.closest ? e.target.closest(".sreset") : null;
    if (!btn) return;
    e.preventDefault();
    e.stopPropagation();
    const it = editing;
    if (!on || !it) return;
    applyEdit(it, btn.getAttribute("data-p"), "");
    announce(it);
  });

  // Page hotkeys must never see keystrokes meant for the card. The event is
  // retargeted to the host once it leaves the shadow root — a div, not a
  // form field — so a page listener's "ignore while typing" check fails and
  // its single-letter shortcut fires: GitHub's "s" opened its search over
  // the card mid-word (owner 2026-09-18). The shield bubbles after the input
  // already took the key (insertion is a default action) and stops before
  // document/window listeners. The pointer shields close the same leak for
  // clicks that land on the card over a page control.
  const SHIELD_EVENTS = [
    "keydown",
    "keypress",
    "keyup",
    "pointerdown",
    "pointerup",
    "mousedown",
    "mouseup",
    "dblclick",
  ];
  for (const type of SHIELD_EVENTS) {
    root.addEventListener(type, (e) => e.stopPropagation(), false);
  }

  let on = false;
  let seq = 0;
  let cardOpen = false;
  let menuOpen = false;
  // Open/closed travels in flags, never in inline styles: reading
  // `node.style.display` as state inverts the mode (inline display starts
  // as "", so `!== "none"` is true before anything ever opened — no
  // hover until the first pick, hover forever after it; owner 2026-09-18).
  // items: {id, n, el, selector, tag, html, rect, styles, vw, vh, comment, saved,
  //          pin, chip, chipTxt, menuFor}
  let items = [];
  let editing = null; // the item whose card is open (draft or saved)
  let menuFor = null;

  const outline = (el, node) => {
    const r = el.getBoundingClientRect();
    node.style.left = r.left + "px";
    node.style.top = r.top + "px";
    node.style.width = Math.max(0, r.width) + "px";
    node.style.height = Math.max(0, r.height) + "px";
    node.style.display = "block";
    return r;
  };

  const payload = (item) => {
    const el = item.el;
    const r = el.getBoundingClientRect();
    item.rect = { x: r.x, y: r.y, width: r.width, height: r.height };
    item.vw = window.innerWidth;
    item.vh = window.innerHeight;
    return {
      n: item.n,
      selector: item.selector,
      tag: item.tag,
      html: String(el.outerHTML || "").slice(0, 20000),
      rect: item.rect,
      // The snapshot from pick time, never a live read: after a preview the
      // element's own computed values carry our mutation (v2c step 5).
      styles: item.styles0 || computed(el),
      styleEdits: item.edits || {},
      vw: item.vw,
      vh: item.vh,
      comment: item.comment,
      saved: item.saved,
    };
  };

  // statePayload is what the strip needs, in one shape, from two doors: the
  // message channel (push, post()) and ExecuteScript's return value (pull —
  // the host asks; the answer rides the command's own result). The pull is
  // the guaranteed one: it needs no page→host bridge, so a page that cannot
  // post still lights the strip up (owner 2026-09-19).
  const statePayload = () => ({
    kind: "state",
    url: location.href,
    count: items.filter((it) => it.saved).length,
    total: items.length,
    items: items.map(payload),
  });

  const sync = () => {
    // Renumber by position so pins read 1..N after a removal.
    items.forEach((it, i) => {
      it.n = i + 1;
      it.pin.textContent = String(it.n);
      it.pin.classList.toggle("on", editing === it);
    });
    outlineSelected();
    post(statePayload());
  };

  const outlineSelected = () => {
    if (editing && editing.el.isConnected) {
      outline(editing.el, sel);
    } else {
      sel.style.display = "none";
    }
  };

  const placePin = (it) => {
    if (!it.el.isConnected) return;
    const r = it.el.getBoundingClientRect();
    it.pin.style.left = Math.max(0, r.left - 9) + "px";
    it.pin.style.top = Math.max(0, r.top - 9) + "px";
    it.pin.style.display = "block";
  };

  const placeChip = (it) => {
    if (!it.saved || !it.el.isConnected) {
      it.chip.style.display = "none";
      return;
    }
    it.chip.style.display = "inline-flex";
    const r = it.el.getBoundingClientRect();
    const w = it.chip.offsetWidth || 120;
    const top = Math.min(window.innerHeight - 34, r.top - 2);
    const left = Math.min(Math.max(2, r.left + 12), Math.max(2, window.innerWidth - w - 2));
    it.chip.style.top = Math.max(2, top) + "px";
    it.chip.style.left = left + "px";
  };

  const placeCard = (it) => {
    const r = it.el.getBoundingClientRect();
    card.style.display = "block";
    const h = card.offsetHeight || 96;
    const w = card.offsetWidth || 300;
    const below = r.bottom + 8 + h <= window.innerHeight;
    const top = below ? r.bottom + 8 : Math.max(4, r.top - h - 8);
    const left = Math.min(Math.max(4, r.left), Math.max(4, window.innerWidth - w - 4));
    card.style.top = top + "px";
    card.style.left = left + "px";
  };

  const closeMenu = () => {
    menu.style.display = "none";
    menuFor = null;
    menuOpen = false;
  };

  const openCard = (it) => {
    editing = it;
    closeMenu();
    hl.style.display = "none";
    input.value = it.comment || "";
    // The disclosure keeps the session's answer, and every control is painted
    // from this item's truth before it is shown.
    stylesBox.open = stylesOpen;
    renderStyles(it);
    cardOpen = true;
    placeCard(it);
    input.focus();
    items.forEach((x) => x.pin.classList.toggle("on", x === it));
    outlineSelected();
  };

  const closeCard = () => {
    card.style.display = "none";
    cardOpen = false;
    input.blur();
    editing = null;
    items.forEach((x) => x.pin.classList.toggle("on", false));
    outlineSelected();
  };

  const removeItem = (it) => {
    if (editing === it) closeCard();
    if (menuFor === it) closeMenu();
    // The annotation is gone, so its proposal is gone: the page goes back to
    // what it had (a preview nobody can see the note for is a lie).
    restoreStyles(it);
    it.pin.remove();
    it.chip.remove();
    items = items.filter((x) => x !== it);
    post({ kind: "removed", n: it.n, selector: it.selector });
    sync();
  };

  const makeNodes = (it) => {
    const pin = document.createElement("div");
    pin.className = "pin";
    pin.textContent = String(it.n);
    pin.title = "Edit annotation " + it.n;
    pin.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (cardOpen) return; // locked: finish the open note first
      openCard(it);
    });
    const chip = document.createElement("div");
    chip.className = "anchip";
    const txt = document.createElement("span");
    txt.className = "txt";
    const dots = document.createElement("button");
    dots.textContent = "…";
    dots.title = "Annotation options";
    dots.setAttribute("aria-label", "Annotation options");
    dots.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (cardOpen) return; // locked: finish the open note first
      openMenu(it);
    });
    const x = document.createElement("button");
    x.textContent = "×";
    x.title = "Remove this annotation";
    x.setAttribute("aria-label", "Remove this annotation");
    x.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      removeItem(it);
    });
    chip.append(txt, dots, x);
    chip.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (cardOpen) return; // locked: finish the open note first
      openCard(it);
    });
    root.append(pin, chip);
    it.pin = pin;
    it.chip = chip;
    it.chipTxt = txt;
  };

  const openMenu = (it) => {
    menuFor = it;
    menuOpen = true;
    menu.style.display = "block";
    const r = it.chip.getBoundingClientRect();
    const mw = menu.offsetWidth || 130;
    const mh = menu.offsetHeight || 110;
    menu.style.top = Math.min(window.innerHeight - mh - 4, r.bottom + 4) + "px";
    menu.style.left = Math.max(4, Math.min(r.left, window.innerWidth - mw - 4)) + "px";
  };

  menu.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    const btn = e.target.closest("button[data-act]");
    const it = menuFor;
    if (!btn || !it) {
      closeMenu();
      return;
    }
    const act = btn.getAttribute("data-act");
    closeMenu();
    if (act === "edit") openCard(it);
    else if (act === "remove") removeItem(it);
    else if (act === "copy") {
      const text = String(it.comment || it.selector || "");
      if (text && navigator.clipboard) {
        navigator.clipboard.writeText(text).then(
          () => post({ kind: "note", message: "Annotation text copied." }),
          () => post({ kind: "warn", message: "Copy failed in this page." }),
        );
      } else {
        post({ kind: "warn", message: "Nothing to copy yet." });
      }
    }
  });

  const reposition = () => {
    if (!on) return;
    const gone = items.filter((it) => !it.el.isConnected);
    gone.forEach((it) => {
      it.pin.remove();
      it.chip.remove();
    });
    if (gone.length) {
      const keptEditing = editing && editing.el.isConnected ? editing : null;
      items = items.filter((it) => it.el.isConnected);
      if (editing && !keptEditing) closeCard();
      else editing = keptEditing;
      if (cardOpen && editing) placeCard(editing);
      sync();
      return;
    }
    items.forEach((it) => {
      placePin(it);
      placeChip(it);
    });
    if (cardOpen && editing) placeCard(editing);
    if (menuFor) openMenu(menuFor);
  };

  const inHost = (e) => {
    const path = e.composedPath ? e.composedPath() : [];
    return path.includes(host);
  };

  const under = (e) => {
    if (inHost(e)) return null;
    return document.elementFromPoint(e.clientX, e.clientY);
  };

  const onMove = (e) => {
    if (!on || cardOpen) return;
    const el = under(e);
    if (!el) {
      hl.style.display = "none";
      return;
    }
    outline(el, hl);
  };

  const pick = (e) => {
    // Locked while a card is open: a click that misses the card must not
    // steal it onto a new pin and drop the unsaved draft (owner 2026-09-18).
    // Save/Cancel/Esc/trash close the card; the next click then picks.
    if (!on || cardOpen || inHost(e)) return;
    const el = under(e);
    if (!el) return;
    // A click on an existing pin or chip reopens it (their own listeners run
    // first and stop propagation; this is the belt for text inside them).
    e.preventDefault();
    e.stopPropagation();
    const it = {
      id: ++seq,
      n: items.length + 1,
      el,
      selector: readable(el),
      tag: String(el.tagName || "").toLowerCase(),
      rect: null,
      // The style inspector's two snapshots: the computed values the human is
      // looking at now, and the element's own inline values (so cancelling
      // never deletes a style the page itself wrote).
      styles0: computed(el),
      inline0: inline0Of(el),
      edits: {},
      vw: window.innerWidth,
      vh: window.innerHeight,
      comment: "",
      saved: false,
    };
    makeNodes(it);
    items.push(it);
    placePin(it);
    hl.style.display = "none";
    post(Object.assign({ kind: "pick" }, payload(it)));
    openCard(it);
    sync();
  };

  const save = () => {
    const it = editing;
    if (!on || !it) return;
    it.comment = String(input.value || "").trim();
    it.saved = true;
    it.chipTxt.textContent = it.comment || it.selector;
    it.chip.title = it.comment || it.selector;
    closeCard();
    placeChip(it);
    post(Object.assign({ kind: "saved" }, payload(it)));
    sync();
  };

  const cancelCard = () => {
    const it = editing;
    // Cancel on a never-saved draft removes its pin (nothing was kept);
    // cancel while editing a saved note keeps the old text but drops the
    // styles the card was previewing.
    if (it && !it.saved) {
      closeCard();
      removeItem(it);
      return;
    }
    if (it) restoreStyles(it);
    closeCard();
  };

  const onKey = (e) => {
    if (!on) return;
    // Enter inside a style control applies that control — it must not save the
    // whole card mid-edit. The shadow root's own activeElement is the only
    // place that answer survives the retargeting below, and it has to be read
    // for BOTH branches: a keystroke in the card arrives at this listener with
    // the host as its target.
    const focused = cardOpen ? root.activeElement : null;
    const inStyles = !!(focused && focused.getAttribute && focused.getAttribute("data-p"));
    // This listener lives on the document, outside the shadow tree, where the
    // event target is retargeted to the host — so "is this the card's input?"
    // cannot be asked by comparing the event target to the input element (it
    // is the host; Enter-to-save was dead for exactly that reason until
    // 2026-09-18). The open card IS the answer: Enter saves it, Escape
    // cancels or closes it.
    if (inHost(e)) {
      if (e.key === "Enter" && cardOpen) {
        if (inStyles) return;
        e.preventDefault();
        e.stopPropagation();
        save();
      } else if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        if (menuOpen) closeMenu();
        else cancelCard();
      }
      return;
    }
    if (e.key === "Enter" && cardOpen) {
      if (inStyles) return;
      e.preventDefault();
      e.stopPropagation();
      save();
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      if (menuOpen) closeMenu();
      else if (cardOpen) cancelCard();
      else exit();
    }
  };

  const onDocClick = (e) => {
    // A click anywhere outside the card/menu dismisses the menu; a click on
    // the open card's own chrome stays (its buttons act). Clicks on pins and
    // chips are handled by their listeners.
    if (!on || inHost(e)) return;
    if (menuOpen) closeMenu();
  };

  saveBtn.addEventListener("click", (e) => { e.preventDefault(); e.stopPropagation(); save(); });
  cancelBtn.addEventListener("click", (e) => { e.preventDefault(); e.stopPropagation(); cancelCard(); });
  trashBtn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (editing) removeItem(editing);
  });
  micBtn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SR) {
      post({ kind: "warn", message: "Voice input is not available in this page — type the note." });
      return;
    }
    try {
      const rec = new SR();
      rec.lang = navigator.language || "en-US";
      rec.interimResults = false;
      rec.maxAlternatives = 1;
      rec.onresult = (ev) => {
        const text = ev.results && ev.results[0] && ev.results[0][0] ? ev.results[0][0].transcript : "";
        if (text) {
          input.value = text;
          input.focus();
        }
      };
      rec.onerror = () => post({ kind: "warn", message: "Dictation did not start — check the microphone." });
      rec.start();
      post({ kind: "note", message: "Listening — speak the note." });
    } catch (err) {
      post({ kind: "warn", message: "Dictation did not start — check the microphone." });
    }
  });
  card.addEventListener("click", (e) => e.stopPropagation());

  function enter() {
    // Re-arming (the chrome's recovery path) must ANNOUNCE, not return
    // early: the listener setup is idempotent, the report is the point. An
    // early `if (on) return;` here is exactly what made a re-arm silent.
    if (!on) {
      on = true;
      if (!host.isConnected) document.documentElement.appendChild(host);
      document.addEventListener("mousemove", onMove, true);
      document.addEventListener("click", pick, true);
      document.addEventListener("click", onDocClick, true);
      document.addEventListener("keydown", onKey, true);
      window.addEventListener("scroll", reposition, true);
      window.addEventListener("resize", reposition, true);
    }
    post({ kind: "enter", url: location.href });
    // The arm also reports what the page already holds: re-arming is the
    // recovery path (the chrome retries once when the page does not answer),
    // and the pins here survive it — only turning the mode OFF discards them.
    sync();
  }

  function clear() {
    closeMenu();
    closeCard();
    restoreAllStyles();
    items.forEach((it) => {
      it.pin.remove();
      it.chip.remove();
    });
    items = [];
    hl.style.display = "none";
    sel.style.display = "none";
    sync();
  }

  function dropLast() {
    // The strip's undo: remove the most recent pin, whatever it is.
    const it = items[items.length - 1];
    if (!it) return;
    removeItem(it);
  }

  function exit() {
    on = false;
    closeMenu();
    closeCard();
    // Leaving the mode takes every preview with it: the page is handed back
    // exactly as it was found.
    restoreAllStyles();
    items.forEach((it) => {
      it.pin.remove();
      it.chip.remove();
    });
    items = [];
    editing = null;
    document.removeEventListener("mousemove", onMove, true);
    document.removeEventListener("click", pick, true);
    document.removeEventListener("click", onDocClick, true);
    document.removeEventListener("keydown", onKey, true);
    window.removeEventListener("scroll", reposition, true);
    window.removeEventListener("resize", reposition, true);
    hl.style.display = "none";
    sel.style.display = "none";
    host.remove();
    post({ kind: "exit" });
    // The handle STAYS, marked off: enter() re-arms this same instance, and
    // the pull door can answer "off" instead of "nothing" — the strip has to
    // tell a page that left the mode (Esc in the page) from a document with
    // no script at all (a navigation in flight). Deleting the handle made
    // those two answers identical (2026-09-19).
  }

  window[KEY] = {
    enter,
    exit,
    clear,
    dropLast,
    // The pull door: the shell calls this through ExecuteScript and hands the
    // JSON straight to the strip. Null while the mode is off.
    state: () => (on ? statePayload() : null),
    // The picture the agent gets must show the PAGE, not our own pin and
    // chip: the chrome hides this overlay around its capture and shows it
    // again. The first clean shot proved why — the crop came back as a photo
    // of the annotation card covering the element (owner 2026-09-19).
    hide: () => { host.style.visibility = "hidden"; },
    show: () => { host.style.visibility = ""; },
  };
  enter();
})();
