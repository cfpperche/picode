export function previewKind(path) {
  const n = String(path || "").toLowerCase();
  const i = n.lastIndexOf(".");
  const ext = i >= 0 ? n.slice(i) : "";
  if (ext === ".svg") return "svg";
  if (ext === ".mmd" || ext === ".mermaid") return "mermaid";
  return "";
}

export function svgDataUrl(text) {
  const t = String(text || "").trim();
  if (!t) return "";
  if (!/<svg[\s>/]/i.test(t)) return "";
  return "data:image/svg+xml;charset=utf-8," + encodeURIComponent(t);
}

export function previewEmpty(text) {
  return !String(text || "").trim();
}
