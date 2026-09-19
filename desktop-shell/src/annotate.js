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
      .pin { position: fixed; min-width: 18px; height: 18px; padding: 0 2px; border-radius: 999px; background: #4a9eff; color: #fff; font: 600 11px/18px system-ui, sans-serif; text-align: center; pointer-events: auto; cursor: pointer; display: none; box-shadow: 0 1px 4px rgba(0,0,0,.35); box-sizing: border-box; }
      .pin.on { outline: 2px solid #fff; box-shadow: 0 0 0 4px rgba(74,158,255,.55), 0 1px 4px rgba(0,0,0,.35); }
      .card { position: fixed; display: none; pointer-events: auto; background: #fff; color: #111; border: 1px solid rgba(0,0,0,.12); border-radius: 10px; box-shadow: 0 6px 22px rgba(0,0,0,.28); padding: 8px; font: 13px/1.4 system-ui, sans-serif; width: 300px; max-width: calc(100vw - 8px); box-sizing: border-box; }
      .card .row { display: flex; align-items: center; gap: 6px; }
      .card .mark { flex: none; width: 22px; height: 22px; border-radius: 6px; background: rgba(74,158,255,.14); color: #1f6feb; font: 600 13px/22px system-ui, sans-serif; text-align: center; }
      .card input { border: 0; outline: 0; font: inherit; color: inherit; background: transparent; min-width: 0; flex: 1; }
      .card .icon { flex: none; border: 0; background: transparent; color: #666; font: 13px/1 system-ui, sans-serif; padding: 5px; border-radius: 6px; cursor: pointer; }
      .card .icon:hover { background: rgba(0,0,0,.06); color: #111; }
      .card .actions { display: flex; justify-content: flex-end; gap: 6px; margin-top: 8px; }
      .card .actions button { border: 0; border-radius: 7px; font: 600 12px/1 system-ui, sans-serif; padding: 8px 12px; cursor: pointer; }
      .card .save { background: #1f6feb; color: #fff; }
      .card .cancel { background: rgba(0,0,0,.06); color: #333; }
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
  const menu = root.querySelector(".menu");

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
      styles: computed(el),
      vw: item.vw,
      vh: item.vh,
      comment: item.comment,
      saved: item.saved,
    };
  };

  const sync = () => {
    // Renumber by position so pins read 1..N after a removal.
    items.forEach((it, i) => {
      it.n = i + 1;
      it.pin.textContent = String(it.n);
      it.pin.classList.toggle("on", editing === it);
    });
    outlineSelected();
    post({
      kind: "state",
      url: location.href,
      count: items.filter((it) => it.saved).length,
      total: items.length,
      items: items.map(payload),
    });
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
    // cancel while editing a saved note keeps the old text.
    if (it && !it.saved) {
      closeCard();
      removeItem(it);
      return;
    }
    closeCard();
  };

  const onKey = (e) => {
    if (!on) return;
    // This listener lives on the document, outside the shadow tree, where the
    // event target is retargeted to the host — so "is this the card's input?"
    // cannot be asked by comparing the event target to the input element (it
    // is the host; Enter-to-save was dead for exactly that reason until
    // 2026-09-18). The open card IS the answer: Enter saves it, Escape
    // cancels or closes it.
    if (inHost(e)) {
      if (e.key === "Enter" && cardOpen) {
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
    if (on) return;
    on = true;
    if (!host.isConnected) document.documentElement.appendChild(host);
    document.addEventListener("mousemove", onMove, true);
    document.addEventListener("click", pick, true);
    document.addEventListener("click", onDocClick, true);
    document.addEventListener("keydown", onKey, true);
    window.addEventListener("scroll", reposition, true);
    window.addEventListener("resize", reposition, true);
    post({ kind: "enter", url: location.href });
  }

  function clear() {
    closeMenu();
    closeCard();
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
    try {
      delete window[KEY];
    } catch (e) {
      window[KEY] = null;
    }
  }

  window[KEY] = { enter, exit, clear, dropLast };
  enter();
})();
