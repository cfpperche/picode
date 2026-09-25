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
