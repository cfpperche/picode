// Pure helpers behind the pin studio (pins v2 review, findings 2, 4, 6, 7,
// 11, 15). No DOM here beyond sessionStorage through the injected store,
// so the rules are testable under node.

// The server's limits (internal/store/pins.go). The studio refuses at the
// same line so the person hears it before the round trip.
export const PIN_LIMITS = Object.freeze({ title: 200, tag: 40, tags: 16, bodyBytes: 100_000 });

// UTF-8 length, which is what the server measures a body by.
export function utf8Bytes(s) {
  const str = String(s == null ? "" : s);
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(str).length;
  let n = 0;
  for (let i = 0; i < str.length; i++) {
    const c = str.charCodeAt(i);
    if (c < 0x80) n += 1;
    else if (c < 0x800) n += 2;
    else if (c >= 0xd800 && c <= 0xdbff) { n += 4; i++; } else n += 3;
  }
  return n;
}

// bodyLimit reports where a body stands against the cap: `over` refuses
// Save, `near` (last 10 %) shows the counter.
export function bodyLimit(body) {
  const bytes = utf8Bytes(body);
  const max = PIN_LIMITS.bodyBytes;
  return { bytes, max, over: bytes > max, near: bytes > max * 0.9 };
}

// The one-line tag rule the server applies (normalizeTag): lower-case, no
// leading '#', whitespace runs fold to one '-'.
export function normalizeTag(raw) {
  const t = String(raw || "").trim().replace(/^#/, "").trim().toLowerCase();
  return t.split(/\s+/).filter(Boolean).join("-");
}

// A pin created on the way to an attachment takes its name from the first
// file, never "Untitled": a dropped screenshot reads as "screenshot", a
// sketch as "Sketch". The typed title always wins.
export function autoTitle(typed, files, fallback = "Sketch") {
  const t = String(typed || "").trim();
  if (t) return t;
  const first = (files || []).find((f) => f && f.name);
  if (!first) return fallback;
  const base = String(first.name).replace(/\.[a-z0-9]{1,8}$/i, "").trim();
  return base || fallback;
}

// Retained drafts (mobile v2's rule, now on the desk): the studio keeps
// what was typed under the pin's key, so navigating away and back — or
// a reload — loses nothing. `store` is sessionStorage in the app.
export function draftKey(id) {
  return "picode-pin-draft:" + (id || "new");
}

export function readDraft(store, id) {
  try {
    const raw = store && store.getItem(draftKey(id));
    if (!raw) return null;
    const d = JSON.parse(raw);
    if (!d || typeof d !== "object") return null;
    return { title: String(d.title || ""), tags: Array.isArray(d.tags) ? d.tags.map(String) : [], body: String(d.body || ""), base: d.base || "" };
  } catch { return null; }
}

export function writeDraft(store, id, draft, base) {
  try {
    store.setItem(draftKey(id), JSON.stringify({ title: draft.title || "", tags: draft.tags || [], body: draft.body || "", base: base || "" }));
  } catch { /* quota, private mode: the server copy still exists */ }
}

export function clearDraft(store, id) {
  try { store.removeItem(draftKey(id)); } catch { /* ignore */ }
}

// Same content, same pin: nothing to save and nothing to retain.
export function sameDraft(a, b) {
  if (!a || !b) return false;
  const ta = (a.tags || []).join(" ");
  const tb = (b.tags || []).join(" ");
  return (a.title || "") === (b.title || "") && (a.body || "") === (b.body || "") && ta === tb;
}

// A retained draft is worth restoring when it differs from the server copy
// and was taken from that same server version (`base` = updatedAt) — or
// from no version at all (a new pin).
export function draftToRestore(draft, server) {
  if (!draft) return null;
  if (!server) return sameDraft(draft, { title: "", tags: [], body: "" }) ? null : draft;
  if (draft.base && draft.base !== server.updatedAt) return null;
  return sameDraft(draft, server) ? null : draft;
}

// The annotated picture rides the Excalidraw scene as a file whose id
// starts with "bg:" (PinSketch). It is stripped before upload — the sketch
// row keeps the picture by reference (baseFileId) and the studio injects
// it again on open — so the scene never carries the image bytes and the
// 2 MB scene cap is about the drawing, not the screenshot.
export const BG_FILE_PREFIX = "bg:";

export function stripBackgroundFiles(scene) {
  if (!scene || typeof scene !== "object") return scene;
  const files = scene.files && typeof scene.files === "object" ? scene.files : {};
  const kept = {};
  for (const [k, v] of Object.entries(files)) {
    if (!String(k).startsWith(BG_FILE_PREFIX)) kept[k] = v;
  }
  return { ...scene, files: kept };
}

// Which background file id the scene expects and lacks, if any.
export function missingBackgroundId(scene) {
  if (!scene || !Array.isArray(scene.elements)) return "";
  const files = scene.files && typeof scene.files === "object" ? scene.files : {};
  for (const el of scene.elements) {
    const id = el && el.type === "image" ? String(el.fileId || "") : "";
    if (id.startsWith(BG_FILE_PREFIX) && !files[id]) return id;
  }
  return "";
}

// The preview URL carries the file's version: an edited sketch keeps its
// id, and the bytes are cached for an hour, so the version is what makes
// the browser fetch the new picture.
export function pinFileSrc(pinId, f) {
  if (!pinId || !f || !f.id) return "";
  const v = f.updatedAt || f.createdAt || "";
  return "/api/pins/" + encodeURIComponent(pinId) + "/files/" + encodeURIComponent(f.id) + (v ? "?v=" + encodeURIComponent(v) : "");
}
