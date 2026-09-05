const TEXT_KIND = {
  ".svg": "svg",
  ".mmd": "mermaid",
  ".mermaid": "mermaid",
  ".md": "markdown",
  ".mdx": "markdown",
};

const BLOB_KIND = {
  ".png": "image",
  ".jpg": "image",
  ".jpeg": "image",
  ".gif": "image",
  ".webp": "image",
  ".pdf": "pdf",
  ".mp3": "audio",
  ".wav": "audio",
  ".ogg": "audio",
  ".m4a": "audio",
  ".mp4": "video",
  ".webm": "video",
  ".mkv": "video",
  ".glb": "model3d",
  ".gltf": "model3d",
};

export function extOf(path) {
  const n = String(path || "").toLowerCase();
  const i = n.lastIndexOf(".");
  return i >= 0 ? n.slice(i) : "";
}

export function previewKind(path) {
  const ext = extOf(path);
  return TEXT_KIND[ext] || BLOB_KIND[ext] || "";
}

export function isBlobKind(kind) {
  return kind === "image" || kind === "pdf" || kind === "audio" || kind === "video" || kind === "model3d";
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

export function fileBlobUrl(agentId, termId, wsId, path) {
  const base = termId
    ? "/api/terminals/" + encodeURIComponent(termId) + "/blob"
    : wsId
      ? "/api/workspaces/" + encodeURIComponent(wsId) + "/blob"
      : "/api/agents/" + encodeURIComponent(agentId) + "/blob";
  return base + "?path=" + encodeURIComponent(path);
}
