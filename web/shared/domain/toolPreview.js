// ADR-0075: a capture is historical metadata, never an arbitrary image URL.
export const MAX_CAPTURE_BYTES = 200 * 1024;
export const MAX_CAPTURE_SIDE = 1600;
export const MAX_CAPTURE_PIXELS = 1600000;
const MAX_DATA_LENGTH = Math.ceil(MAX_CAPTURE_BYTES / 3) * 4 + 32;
const unavailable = "Capture unavailable";
const text = (s, max) => typeof s === "string" ? s.replace(/[\u0000-\u001f\u007f]/g, "").slice(0, max) : "";

function dimensions(bytes, mime) {
  const view = new DataView(bytes.buffer);
  if (mime === "png") {
    if (bytes.length < 24 || view.getUint32(0) !== 0x89504e47 || view.getUint32(4) !== 0x0d0a1a0a ||
        view.getUint32(8) !== 13 || view.getUint32(12) !== 0x49484452) return null;
    return [view.getUint32(16), view.getUint32(20)];
  }
  if (bytes.length < 4 || view.getUint16(0) !== 0xffd8) return null;
  for (let i = 2; i + 3 < bytes.length;) {
    if (bytes[i++] !== 0xff) return null;
    while (bytes[i] === 0xff) i++;
    const marker = bytes[i++];
    if (marker === 0xd9 || marker === 0xda || i + 2 > bytes.length) return null;
    const size = view.getUint16(i);
    if (size < 2 || i + size > bytes.length) return null;
    if ([0xc0, 0xc1, 0xc2].includes(marker)) {
      if (size < 8) return null;
      return [view.getUint16(i + 5), view.getUint16(i + 3)];
    }
    i += size;
  }
  return null;
}

export function captureURL(value) {
  try {
    if (typeof value !== "string" || value.length > 4096) return "";
    const u = new URL(value);
    if (u.protocol !== "https:" && u.protocol !== "http:") return "";
    return (u.origin + u.pathname).slice(0, 512);
  } catch { return ""; }
}

export function previewFromDetails(details) {
  const p = details && typeof details === "object" ? details.preview : null;
  if (!p || typeof p.image !== "string" || p.image.length > MAX_DATA_LENGTH) return null;
  const match = /^data:image\/(png|jpeg);base64,([A-Za-z0-9+/]+={0,2})$/.exec(p.image);
  if (!match || match[2].length % 4 !== 0) return null;
  try {
    const raw = atob(match[2]);
    if (!raw.length || raw.length > MAX_CAPTURE_BYTES) return null;
    const size = dimensions(Uint8Array.from(raw, (c) => c.charCodeAt(0)), match[1]);
    if (!size || size.some((n) => n <= 0 || n > MAX_CAPTURE_SIDE) || size[0] * size[1] > MAX_CAPTURE_PIXELS) return null;
    return {
      image: p.image, url: captureURL(p.url), title: text(p.title, 160),
      ...(Number.isSafeInteger(p.ts) && p.ts > 0 && p.ts <= 8640000000000000 ? { ts: p.ts } : {}),
      ...(text(p.source, 120) ? { source: text(p.source, 120) } : {}),
    };
  } catch { return null; }
}

export function captureState(details) {
  const preview = previewFromDetails(details);
  return { preview, previewError: !preview && details?.preview != null ? unavailable : "" };
}

// Called by both application reducers AND the desktop's event handler.
export function updateCapture(item, details) {
  if (item.status !== "···") return item;
  const next = captureState(details);
  if (!next.preview && !next.previewError) return item;
  if (next.preview?.ts && item.preview?.ts && next.preview.ts < item.preview.ts) return item;
  return { ...item, ...next };
}

export function toolResultDetail(result) {
  // Do not serialize image bytes into the expandable text (even a blocked URI).
  return JSON.stringify(result || {}, (key, value) => key === "preview" && value && typeof value === "object"
    ? { ...value, image: value.image == null ? undefined : "[capture omitted]" } : value, 2);
}
