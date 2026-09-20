// annotate.js — the picker's arithmetic and the page-side script (v2c step 1).
//
// The work tab's page is a native WebView2 child: HTML can never paint over
// it, so the picker works on the frozen still (`btab_preview`) — the mechanic
// the ⋮ menu already uses. A click on that still is a *viewport* coordinate,
// and `document.elementFromPoint` takes exactly that, so no scroll offset
// appears anywhere here: the still is the visible area, and the only
// conversion is its display scale. Pure functions, table-tested.

// pickStyles is the short list of computed properties worth shipping: enough
// to describe "what this looks like now" without dumping the cascade.
export const PICK_STYLES = [
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

// stillToViewport maps a click on the displayed still to the viewport point
// it represents. Returns null for a still with no measurable size or a click
// outside it — the caller says so rather than picking the wrong element.
export function stillToViewport({ clickX, clickY, naturalWidth, naturalHeight, displayWidth, displayHeight }) {
  const nw = Number(naturalWidth) || 0;
  const nh = Number(naturalHeight) || 0;
  const dw = Number(displayWidth) || 0;
  const dh = Number(displayHeight) || 0;
  if (nw <= 0 || nh <= 0 || dw <= 0 || dh <= 0) return null;
  const x = Math.round((Number(clickX) || 0) * (nw / dw));
  const y = Math.round((Number(clickY) || 0) * (nh / dh));
  if (x < 0 || y < 0 || x >= nw || y >= nh) return null;
  return { x, y };
}

// cropRect turns an element's viewport rect into a rectangle *inside the
// still image*, padded a little and clamped to the picture: the crop the
// human sees is the crop the agent gets. Null when the result would be a
// sliver (a rect from a page that has since moved, say).
export function cropRect({ rect, naturalWidth, naturalHeight, displayWidth, displayHeight, padding = 8 }) {
  const nw = Number(naturalWidth) || 0;
  const nh = Number(naturalHeight) || 0;
  const dw = Number(displayWidth) || 0;
  const dh = Number(displayHeight) || 0;
  const r = rect || {};
  if (nw <= 0 || nh <= 0 || dw <= 0 || dh <= 0) return null;
  const sx = nw / dw;
  const sy = nh / dh;
  const left = Math.max(0, Math.floor((Number(r.x) || 0) * sx) - padding);
  const top = Math.max(0, Math.floor((Number(r.y) || 0) * sy) - padding);
  const right = Math.min(nw, Math.ceil(((Number(r.x) || 0) + (Number(r.width) || 0)) * sx) + padding);
  const bottom = Math.min(nh, Math.ceil(((Number(r.y) || 0) + (Number(r.height) || 0)) * sy) + padding);
  const w = right - left;
  const h = bottom - top;
  if (w < 2 || h < 2) return null;
  return { x: left, y: top, width: w, height: h };
}

// pickScript is what runs in the page through the shell's CDP bridge
// (`Runtime.evaluate`, curated tier — the same door the agent's browser verbs
// use). It describes the element under the point: a readable selector, the
// outer HTML, its viewport rect and the short style list.
export function pickScript(x, y) {
  return `(() => {
    const el = document.elementFromPoint(${Number(x)}, ${Number(y)});
    if (!el) return "";
    const readable = (node) => {
      let s = node.tagName.toLowerCase();
      if (node.id) s += "#" + node.id;
      const cls = typeof node.className === "string" ? node.className.trim().split(/\\s+/) : [];
      if (cls.length && cls[0]) s += "." + cls.slice(0, 2).join(".");
      return s;
    };
    const cs = getComputedStyle(el);
    const styles = {};
    for (const p of ${JSON.stringify(PICK_STYLES)}) styles[p] = cs.getPropertyValue(p);
    const r = el.getBoundingClientRect();
    return JSON.stringify({
      selector: readable(el),
      tag: el.tagName.toLowerCase(),
      html: el.outerHTML.slice(0, 20000),
      rect: { x: r.x, y: r.y, width: r.width, height: r.height },
      styles,
    });
  })()`;
}

// parsePick accepts what the bridge hands back — a JSON string, or the
// Runtime.evaluate envelope around it — and returns a shape we trust, or
// null when there is nothing usable in it.
export function parsePick(value) {
  let raw = value;
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    raw = raw.result?.value ?? raw.value ?? raw.result ?? raw.text ?? "";
  }
  if (typeof raw !== "string" || !raw.trim()) return null;
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!parsed || typeof parsed !== "object") return null;
  const rect = parsed.rect || {};
  if (!Number.isFinite(Number(rect.width)) || !Number.isFinite(Number(rect.height))) return null;
  const styles = parsed.styles && typeof parsed.styles === "object" ? parsed.styles : {};
  return {
    selector: typeof parsed.selector === "string" ? parsed.selector : "",
    tag: typeof parsed.tag === "string" ? parsed.tag : "",
    html: typeof parsed.html === "string" ? parsed.html : "",
    rect: {
      x: Number(rect.x) || 0,
      y: Number(rect.y) || 0,
      width: Number(rect.width) || 0,
      height: Number(rect.height) || 0,
    },
    styles,
  };
}

// stylesToCSS is how the note carries the computed styles: one `prop: value`
// per line, empty ones dropped — readable by an agent, not a JSON blob.
//
// When the style inspector proposed changes, they follow a `/* proposed */`
// marker in the same shape, so the agent reads what the page HAD and what the
// human wants side by side. The two must not be one list: the preview mutates
// the live element, and a single list of values would read as "this is the
// page", which is the one thing it is not (v2c step 5, owner 2026-09-18).
export function stylesToCSS(styles, edits) {
  const lines = [];
  if (styles && typeof styles === "object") {
    for (const [k, v] of Object.entries(styles)) {
      if (typeof v === "string" && v.trim() !== "") lines.push(`${k}: ${v.trim()}`);
    }
  }
  const changed = Object.entries(edits && typeof edits === "object" ? edits : {})
    .filter(([, v]) => v && typeof v === "object" && String(v.to ?? "").trim() !== "");
  if (changed.length) {
    lines.push("/* proposed */");
    for (const [k, v] of changed) lines.push(`${k}: ${String(v.to).trim()}`);
  }
  return lines.join("\n");
}

// pickLabel is the one line the overlay shows for the chosen element.
export function pickLabel(pick) {
  if (!pick) return "";
  const r = pick.rect || {};
  return `${pick.selector || pick.tag || "element"} — ${Math.round(r.width || 0)}×${Math.round(r.height || 0)}`;
}

import { isTermTab, tabTermId } from "./routes.js";

// sendLabel is what rides the strip's Send button: the count of pending
// (saved) annotations, or a bare Send when there is nothing to ship yet.
export function sendLabel(count) {
  const n = Number(count) || 0;
  return n > 0 ? `Send ${n}` : "Send";
}

// normalizeEdits keeps the inspector's proposals trustworthy: a prop ->
// {from, to} map of non-empty strings, everything else dropped before it can
// reach the note or the store.
function normalizeEdits(raw) {
  const out = {};
  if (!raw || typeof raw !== "object") return out;
  for (const [k, v] of Object.entries(raw)) {
    if (!k || !v || typeof v !== "object") continue;
    const to = typeof v.to === "string" ? v.to.trim() : "";
    if (!to) continue;
    out[String(k)] = { from: typeof v.from === "string" ? v.from : "", to };
  }
  return out;
}

// stateItems normalizes a state sync from the page into trusted items: the
// chrome's Send count and batch payloads come from here, so junk (a rect
// that is not a rect, a comment that is not a string) is dropped, never
// rendered or posted.
export function stateItems(msg) {
  const raw = Array.isArray(msg?.items) ? msg.items : [];
  const out = [];
  for (const it of raw) {
    if (!it || typeof it !== "object") continue;
    const rect = it.rect || {};
    if (!Number.isFinite(Number(rect.width)) || !Number.isFinite(Number(rect.height))) continue;
    out.push({
      n: Number(it.n) || 0,
      selector: typeof it.selector === "string" ? it.selector : "",
      tag: typeof it.tag === "string" ? it.tag : "",
      html: typeof it.html === "string" ? it.html : "",
      rect: {
        x: Number(rect.x) || 0,
        y: Number(rect.y) || 0,
        width: Number(rect.width) || 0,
        height: Number(rect.height) || 0,
      },
      styles: it.styles && typeof it.styles === "object" ? it.styles : {},
      // What the inspector proposed, prop -> {from, to}: the note says both,
      // and the Send reads this (never the live element, which carries the
      // preview's own mutation).
      styleEdits: normalizeEdits(it.styleEdits),
      vw: Number(it.vw) || 0,
      vh: Number(it.vh) || 0,
      comment: typeof it.comment === "string" ? it.comment : "",
      saved: it.saved === true,
    });
  }
  return out;
}

// parseAnnotMessage unwraps one page→chrome event into {id, inner} or null.
// The shell relays the page's object through WebMessageAsJson, but a JSON
// text that arrives JSON-encoded a second time (a quoted string instead of
// an object) is unwrapped once more rather than dropped silently — that
// silent drop read as "chip saved, Send 0" (owner 2026-09-18). Junk in,
// null out, never a throw.
export function parseAnnotMessage(outer) {
  let msg = outer;
  if (typeof msg === "string") {
    try {
      msg = JSON.parse(msg);
    } catch {
      return null;
    }
  }
  if (!msg || typeof msg !== "object") return null;
  const id = msg.id != null ? String(msg.id) : "";
  let inner = msg.raw;
  if (typeof inner === "string") {
    try {
      inner = JSON.parse(inner);
    } catch {
      return null;
    }
  }
  if (typeof inner === "string") {
    try {
      inner = JSON.parse(inner);
    } catch {
      return null;
    }
  }
  if (!inner || typeof inner !== "object") return null;
  return { id, inner };
}

// resolveSendTarget picks where a Send delivers (owner 2026-09-18: a
// browser pane open on a session delivers to THAT session — never to
// whatever terminal happens to run first).
//
// Decision table (a bound miss never falls through to a stranger session):
// | bound session        | terminal in list     | result                                |
// |----------------------|----------------------|---------------------------------------|
// | term tab, running    | yes, running         | { kind: "terminal", id }              |
// | term tab, stopped    | yes, not running     | { none, reason } — keep notes         |
// | term tab, gone       | no                   | { none, reason } — keep notes         |
// | agent tab            | same id, running     | { kind: "agent", agentId, terminalId }|
// | agent tab            | missing / stopped    | { none, reason } — keep notes         |
// | none (standalone)    | any running          | { kind: "terminal", first running }   |
// | none (standalone)    | none running         | { none, "Nothing to send to…" }       |
export function resolveSendTarget({ boundSession, terminals }) {
  const list = Array.isArray(terminals) ? terminals : [];
  const live = (id) => list.find((t) => t && t.id === id);
  if (boundSession) {
    if (isTermTab(boundSession)) {
      const id = tabTermId(boundSession);
      const t = live(id);
      if (t && t.running) return { kind: "terminal", id, name: t.name || id };
      if (t) return { none: true, reason: `${t.name || id} is not running — start it and Send again.` };
      return { none: true, reason: "That terminal is gone." };
    }
    const t = live(boundSession);
    if (t && t.running) return { kind: "agent", agentId: boundSession, terminalId: t.id, name: t.name || t.id };
    return { none: true, reason: "That agent has no running terminal — open it in a terminal and Send again." };
  }
  const first = list.find((t) => t && t.running);
  if (first) return { kind: "terminal", id: first.id, name: first.name || first.id };
  return { none: true, reason: "Nothing to send to: no agent terminal is running." };
}

// parseStatePayload reads the pull door's answer: the shell hands back the
// JSON text the page returned, possibly JSON-encoded once more (an
// ExecuteScript result is a JSON value), or nothing at all for an unarmed
// document. Junk in, null out — the caller keeps its last good state.
export function parseStatePayload(raw) {
  let value = raw;
  for (let i = 0; i < 2; i += 1) {
    if (typeof value !== "string") break;
    if (!value.trim()) return null;
    try {
      value = JSON.parse(value);
    } catch {
      return null;
    }
  }
  if (!value || typeof value !== "object") return null;
  // "off" is an answer too: the page is alive and its mode is off (the human
  // pressed Esc in the page), which the strip mirrors. An empty answer means
  // something else — no script in this document (a navigation in flight).
  if (value.kind !== "state" && value.kind !== "off") return null;
  return value;
}

// batchMessage is the ONE context the agent gets for the whole set (the
// reference's "5 annotations"): a head line plus one numbered line per pin,
// each pin's sentence (or its selector when the note is empty).
export function batchMessage({ url, items, stagedShots }) {
  const list = Array.isArray(items) ? items : [];
  const head = `${list.length} annotation${list.length === 1 ? "" : "s"}${url ? ` on ${url}` : ""}`;
  const lines = list.map((it) => {
    const n = it && it.n != null ? it.n : "?";
    const comment = it && typeof it.comment === "string" ? it.comment.trim() : "";
    const sel = it && typeof it.selector === "string" ? it.selector : "";
    return `${n}. ${comment || sel || "element"}`;
  });
  const out = [head, ...lines];
  // Say where the pictures are when the paste could not carry them all: the
  // note names every crop, so the agent reads what it needs (the number of
  // annotations is never capped, only the attachments per paste).
  const left = Number(stagedShots) || 0;
  if (left > 0) out.push("", `${left} more screenshot${left === 1 ? "" : "s"} staged next to the note.`);
  return out.join("\n");
}

// pastePaths splits a staged package for the prompt door: the note first (it
// IS the package: every pin, its evidence, and the name of every crop), then
// as many crops as the door's own file cap leaves room for (ADR-0089 takes at
// most four files per paste). Returns the paths to paste and how many crops
// stayed behind, so the message can say so. A Send of fifty annotations still
// arrives as one message and one context (owner 2026-09-19).
export function pastePaths(paths, cap = PASTE_FILE_CAP) {
  const list = (Array.isArray(paths) ? paths : []).filter((p) => typeof p === "string" && p.trim() !== "");
  const limit = Math.max(1, Number(cap) || PASTE_FILE_CAP);
  return { paste: list.slice(0, limit), left: Math.max(0, list.length - limit) };
}

// The prompt door's per-paste ceiling (ADR-0089, decision 1).
export const PASTE_FILE_CAP = 4;
