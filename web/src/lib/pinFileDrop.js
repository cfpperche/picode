export function pinFileFromDrop(event) {
  const dt = event && event.dataTransfer;
  if (!dt) return null;
  const custom = dt.getData("application/x-picode-pin-file");
  if (custom) {
    try { return JSON.parse(custom); } catch { /* ignore */ }
  }
  const uri = dt.getData("text/uri-list") || dt.getData("text/plain") || "";
  const m = /\/api\/pins\/([^/\s]+)\/files\/([^/\s?#]+)/.exec(uri);
  if (!m) return null;
  return { pinId: decodeURIComponent(m[1]), fileId: decodeURIComponent(m[2]) };
}

export function pinFileURL(ref) {
  if (!ref || !ref.pinId || !ref.fileId) return "";
  return "/api/pins/" + encodeURIComponent(ref.pinId) + "/files/" + encodeURIComponent(ref.fileId);
}
