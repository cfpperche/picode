// Search stays inside the server's owner-scoped browse API. It does not
// reinterpret a root as a filesystem address or follow file/symlink rows.
export async function searchFileFolders({ read, dir = "", query, signal, onProgress, maxFolders = 300, maxEntries = 12000 }) {
  const pending = [dir], seen = new Set(), results = [];
  const words = query.toLocaleLowerCase().trim().split(/\s+/).filter(Boolean);
  let entries = 0, skipped = 0;
  if (!words.length) return { results, limited: false, skipped };
  while (pending.length && seen.size < maxFolders && entries < maxEntries) {
    signal?.throwIfAborted();
    const path = pending.shift();
    if (seen.has(path)) continue;
    seen.add(path);
    let page;
    try { page = await read(path, signal); }
    catch (error) {
      if (signal?.aborted || /folder changed/i.test(error.message || "") || path === dir) throw error;
      skipped++;
      continue;
    }
    for (const [isDir, rows] of [[true, page.dirs || []], [false, page.files || []]]) {
      for (const row of rows) {
        if (++entries > maxEntries) break;
        if (words.every(word => row.path.toLocaleLowerCase().includes(word))) results.push({ ...row, isDir });
        if (isDir && !seen.has(row.path)) pending.push(row.path);
      }
    }
    onProgress?.(seen.size);
  }
  return { results: results.sort((a, b) => a.path.localeCompare(b.path)), limited: pending.length > 0 || entries > maxEntries, skipped };
}

export const parentFolder = path => String(path || "").split("/").slice(0, -1).join("/");

export function folderPage(page) {
  if (!page || page.cwdOk === false || !page.root) throw new Error("This working folder is unavailable.");
  if (!Array.isArray(page.dirs) || !Array.isArray(page.files)) throw new Error("Could not read this folder.");
  return page;
}
