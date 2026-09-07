import { api, humanizeError } from "@picode/shared/client/api.js";
const treeApiBase = kind => kind === "term" ? "/api/terminals/" : kind === "workspace" ? "/api/workspaces/" : "/api/agents/";
import { previewKind, isBlobKind } from "@picode/shared/domain/filePreview.js";

export function withFileRoot(url, root) {
  return url && root ? url + (url.includes("?") ? "&" : "?") + "root=" + encodeURIComponent(root) : url;
}

// root is a precondition, never a filesystem address to read through.
export function ownerFileURL(owner, resource, path = "", root = "") {
  const query = new URLSearchParams();
  if (path) query.set(resource === "browse" ? "dir" : "path", path);
  if (root) query.set("root", root);
  return `${treeApiBase(owner.kind)}${encodeURIComponent(owner.id)}/${resource}${query.size ? "?" + query : ""}`;
}

export async function readFile(owner, path, root, signal) {
  if (!isBlobKind(previewKind(path))) {
    const page = await api(ownerFileURL(owner, "text", path, root), { signal });
    return { ...page, kind: "text", text: page.text || "" };
  }
  const res = await fetch(ownerFileURL(owner, "blob", path, root), { signal });
  if (!res.ok) {
    let message = res.statusText;
    try { message = (await res.json()).error || message; } catch { /* keep status */ }
    throw new Error(message);
  }
  return { kind: "bin", path, src: URL.createObjectURL(await res.blob()) };
}

export function fileMessage(raw = "", subject = "file") {
  if (/changed on disk/i.test(raw)) return "This file changed on disk.";
  if (/gone|not found|no such file/i.test(raw)) return `That ${subject} is gone.`;
  if (/too large/i.test(raw)) return "This file is too large to display.";
  if (/can't show|can't write|unsupported/i.test(raw)) return "Can't display this file.";
  if (/escapes/i.test(raw)) return "That path is outside this project.";
  if (/permission denied/i.test(raw)) return `You don't have access to this ${subject}.`;
  return humanizeError(raw);
}
