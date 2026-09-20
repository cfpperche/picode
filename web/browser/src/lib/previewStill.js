// The options menu opens over a frozen page (a windowed WebView2 paints
// over HTML): this module keeps the "is this capture worth hiding the live
// page behind?" decision in one testable place. The shell answers
// btab_preview with raw PNG bytes; anything else — or nothing — means the
// page stays live behind the host background while the menu is up.

export function previewBytes(bytes) {
  if (!bytes) return null;
  if (bytes instanceof ArrayBuffer) return bytes.byteLength ? bytes : null;
  if (ArrayBuffer.isView(bytes)) {
    return bytes.byteLength
      ? bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
      : null;
  }
  if (Array.isArray(bytes)) {
    return bytes.length ? new Uint8Array(bytes).buffer : null;
  }
  return null;
}

export function previewUrl(bytes) {
  const buf = previewBytes(bytes);
  if (!buf) return "";
  try {
    return URL.createObjectURL(new Blob([buf], { type: "image/png" }));
  } catch {
    return "";
  }
}

// A blob URL only becomes the still once it decodes to real pixels. An
// empty capture (the page had not composited a frame yet) resolves the IPC
// call fine and makes a truthy URL — hiding the live page behind that is
// the uniform gray the owner saw on x.com (2026-09-16).
export function verifyPreviewUrl(url, timeoutMs = 2500) {
  return new Promise((resolve, reject) => {
    if (!url || typeof Image === "undefined") {
      reject(new Error("preview: no image"));
      return;
    }
    const img = new Image();
    const t = setTimeout(() => {
      img.removeAttribute("src");
      reject(new Error("preview: decode timeout"));
    }, timeoutMs);
    img.onload = () => {
      clearTimeout(t);
      if (img.naturalWidth > 0 && img.naturalHeight > 0) resolve(url);
      else reject(new Error("preview: empty frame"));
    };
    img.onerror = () => {
      clearTimeout(t);
      reject(new Error("preview: decode failed"));
    };
    img.src = url;
  });
}

// The legacy hide/restore decision (shells without the live-layers protocol,
// pre-ADR-0161): the native page must hide — and, uncovering, restore —
// exactly for these rows. The restore side is the same rows read backwards:
// uncover the page the moment no row holds, and never park it behind a still
// or a failure that outlived its menu.
//
// covered =
// | live layers | overlay cover | menu | verified still | capture failed | decision |
// | ----------- | ------------- | ---- | -------------- | -------------- | -------- |
// | yes         | —             | —    | —              | —              | never (ADR-0161: geometry, not hiding) |
// | no          | yes           | —    | —              | —              | covered (a dialog or toast needs the page out of the way) |
// | no          | no            | open | yes            | —              | covered (the frozen page sits behind the menu) |
// | no          | no            | open | no             | no             | **visible** (the menu waits for the capture) |
// | no          | no            | open | no             | yes            | covered (host gray; the menu still works) |
// | no          | no            | shut | —              | —              | visible (restore, even with a stale still) |
//
// What stays in the component: the capture itself, the 600 ms open-wait, and
// parking the cover state when the tab hides without unmounting.
export function coverDecision({ liveLayers, overlayCover, menuOpen, still, previewFailed }) {
  if (liveLayers) return false;
  if (overlayCover) return true;
  return !!(menuOpen && (still || previewFailed));
}
