import { api, humanizeError } from "@picode/shared/client/api.js";
import { treeApiBase } from "./fileTree.js";
import { previewKind, isBlobKind } from "@picode/shared/domain/filePreview.js";

export function withFileRoot(url, root) {
  return url && root ? url + (url.includes("?") ? "&" : "?") + "root=" + encodeURIComponent(root) : url;
}

// root is a precondition, never a filesystem address to read through.
// worktree names a sibling checkout of the owner's repository (a branch, or
// a commit hash when detached): the server resolves it against `git worktree
// list`, so no filesystem path ever travels in the URL.
export function ownerFileURL(owner, resource, path = "", root = "", worktree = "") {
  const query = new URLSearchParams();
  if (path) query.set(resource === "browse" ? "dir" : "path", path);
  if (root) query.set("root", root);
  if (worktree) query.set("worktree", worktree);
  return `${treeApiBase(owner.kind)}${encodeURIComponent(owner.id)}/${resource}${query.size ? "?" + query : ""}`;
}

export async function readFile(owner, path, root, signal, worktree = "") {
  if (!isBlobKind(previewKind(path))) {
    const page = await api(ownerFileURL(owner, "text", path, root, worktree), { signal });
    return { ...page, kind: "text", text: page.text || "" };
  }
  const res = await fetch(ownerFileURL(owner, "blob", path, root, worktree), { signal });
  if (!res.ok) {
    let message = res.statusText;
    try { message = (await res.json()).error || message; } catch { /* keep status */ }
    throw new Error(message);
  }
  return { kind: "bin", path, src: URL.createObjectURL(await res.blob()) };
}

export function fileMessage(raw = "") {
  if (/changed on disk/i.test(raw)) return "This file changed on disk.";
  if (/gone|not found|no such file/i.test(raw)) return "That file is gone.";
  if (/too large/i.test(raw)) return "This file is too large to display.";
  if (/not available|can't show|can't write|unsupported/i.test(raw)) return "Can't display this file.";
  if (/escapes/i.test(raw)) return "That path is outside this project.";
  if (/permission denied/i.test(raw)) return "You don't have access to this file.";
  return humanizeError(raw);
}
