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
