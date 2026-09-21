export const MAX_ATTACH = 4;
export const MAX_ATTACH_BYTES = 4 * 1024 * 1024;

const IMAGE = {
  "image/png": "image/png",
  "image/jpeg": "image/jpeg",
  "image/jpg": "image/jpeg",
  "image/gif": "image/gif",
  "image/webp": "image/webp",
};

export function isImageFile(file) {
  if (!file) return false;
  if (IMAGE[(file.type || "").toLowerCase()]) return true;
  return /\.(png|jpe?g|gif|webp)$/i.test(file.name || "");
}

export function planAttachFiles(files, already) {
  const list = [...(files || [])].filter(Boolean);
  const out = [];
  let tooLarge = 0;
  let tooMany = false;
  const have = Math.max(0, already | 0);
  for (const f of list) {
    const size = f.size || 0;
    if (size > MAX_ATTACH_BYTES) {
      tooLarge += 1;
      continue;
    }
    if (have + out.length >= MAX_ATTACH) {
      tooMany = true;
      continue;
    }
    out.push(f);
  }
  return { files: out, tooLarge, tooMany };
}

export function readAttachFile(file) {
  return new Promise((resolve, reject) => {
    if (!file) {
      reject(new Error("empty"));
      return;
    }
    if ((file.size || 0) > MAX_ATTACH_BYTES) {
      reject(new Error("too-large"));
      return;
    }
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result || "");
      const i = s.indexOf(",");
      resolve({
        name: file.name || "file",
        mime: file.type || "application/octet-stream",
        data: i >= 0 ? s.slice(i + 1) : s,
        url: s,
        image: isImageFile(file),
      });
    };
    r.onerror = () => reject(new Error("read"));
    r.readAsDataURL(file);
  });
}
// clipboardFiles(clipboardData): the files a paste carries, if any. A
// text-only or empty paste answers [] so the terminal keeps its native
// paste untouched; any staged file routes to the attach bar instead
// (VSCode #301603's rule — readText empty, try the file — delivered by
// path through the drop door, never pasted as bytes).
export function clipboardFiles(clipboardData) {
  const files = clipboardData && clipboardData.files ? [...clipboardData.files] : [];
  return files.filter(Boolean);
}

// termHasPromptDoor(term): true when the prompt door serves this pane — a
// running CLI or the Pi TUI. Plain and stopped shells have no door (the
// server answers 409), so a files-paste there teaches instead of opening.
export function termHasPromptDoor(term) {
  return !!(term && term.launchCli && term.running);
}
// promptDoorFor({kind, agentMode, term}): the one door predicate every
// terminal surface answers — the same rule the context menu uses
// (termMenu.js paneCapabilities): an interactive agent pane, or a running
// CLI. Plain and stopped shells have no door, whatever record they arrive
// with (a bound store record carries no launch fields).
export function promptDoorFor({ kind, agentMode, term } = {}) {
  if (kind === "agent" && agentMode === "interactive") return true;
  return termHasPromptDoor(term);
}

// clipboardText(clipboardData): the text riding alongside a paste, if any.
// A files-paste that also carries text seeds the attach message with it
// instead of dropping it (the keydown path's late text is claimed away by
// termPasteClaim.js, so the message is the text's only landing).
export function clipboardText(clipboardData) {
  try {
    if (clipboardData && typeof clipboardData.getData === "function") {
      return String(clipboardData.getData("text/plain") || "");
    }
  } catch {
    return "";
  }
  return "";
}
